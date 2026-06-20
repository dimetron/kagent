"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AppRenderer, type AppRendererProps } from "@mcp-ui/client";
import type { CallToolResult, ReadResourceResult, ContentBlock } from "@modelcontextprotocol/sdk/types.js";
import { useTheme } from "next-themes";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useChatMcpApps, type McpAppVisibleToolCall } from "@/components/chat/ChatMcpAppsContext";

interface McpAppRendererProps {
  namespace: string;
  serverName: string;
  toolName: string;
  toolResourceUri: string;
  toolInput?: Record<string, unknown>;
  toolResult?: CallToolResult;
  /**
   * Delivers a message the app pushed into the conversation via the MCP Apps
   * `ui/message` channel. The host injects it as a new chat turn so the agent
   * can act on it (e.g. start monitoring a build after the app triggered it).
   */
  onSendMessage?: (text: string) => Promise<void>;
  /**
   * Promotes an iframe-initiated model-visible tools/call into the regular chat
   * turn/tool-call lifecycle. App-only calls stay in-iframe via the MCP proxy.
   */
  onVisibleToolCall?: (call: McpAppVisibleToolCall) => Promise<void>;
}

/** Join the text content blocks of an app `ui/message` into a single prompt. */
function extractMessageText(content: ContentBlock[] | undefined): string {
  if (!Array.isArray(content)) {
    return "";
  }
  return content
    .filter((block): block is ContentBlock & { type: "text"; text: string } => block?.type === "text" && typeof (block as { text?: unknown }).text === "string")
    .map((block) => block.text)
    .join("\n")
    .trim();
}

function requireData<T>(response: { data?: T; error?: string; message: string }): T {
  if (response.error || !response.data) {
    throw new Error(response.error || response.message);
  }
  return response.data;
}

function mcpAppApiPath(namespace: string, name: string): string {
  return `/api/mcp-apps/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`;
}

async function fetchMcpAppApi<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    cache: "no-store",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
  const payload = await response.json().catch(() => ({
    error: `MCP Apps request failed with status ${response.status}`,
    message: response.statusText,
  }));
  if (!response.ok) {
    throw new Error(payload.error || payload.message || `MCP Apps request failed with status ${response.status}`);
  }
  return requireData<T>(payload);
}

export function McpAppRenderer({
  namespace,
  serverName,
  toolName,
  toolResourceUri,
  toolInput,
  toolResult,
  onSendMessage,
  onVisibleToolCall,
}: McpAppRendererProps) {
  const { resolvedTheme } = useTheme();
  const { getMcpToolForAppCall } = useChatMcpApps();
  const [sandboxUrl, setSandboxUrl] = useState<URL | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const id = requestAnimationFrame(() => {
      // Tell the sandbox proxy which origin to trust for postMessage, so it can
      // reject messages from any other origin instead of accepting "*".
      const url = new URL("/sandbox_proxy.html", window.location.origin);
      url.searchParams.set("parentOrigin", window.location.origin);
      setSandboxUrl(url);
    });
    return () => cancelAnimationFrame(id);
  }, []);

  const hostContext = useMemo<NonNullable<AppRendererProps["hostContext"]>>(
    () => ({
      theme: resolvedTheme === "dark" ? "dark" : "light",
      locale: typeof navigator !== "undefined" ? navigator.language : "en-US",
    }),
    [resolvedTheme],
  );

  const sandbox = useMemo(() => sandboxUrl ? { url: sandboxUrl } : undefined, [sandboxUrl]);

  // Advertise the host channels we actually implement so capability-gated apps
  // know they can proxy tool/resource calls and push messages to the chat.
  const hostCapabilities = useMemo<NonNullable<AppRendererProps["hostCapabilities"]>>(() => ({
    serverTools: {},
    serverResources: {},
    openLinks: {},
    message: { text: {} },
  }), []);

  const handleReadResource = useCallback<NonNullable<AppRendererProps["onReadResource"]>>(async ({ uri }) => {
    return fetchMcpAppApi<ReadResourceResult>(
      `${mcpAppApiPath(namespace, serverName)}/resources?uri=${encodeURIComponent(uri)}`,
    );
  }, [namespace, serverName]);

  // Route iframe tools/call by MCP tool visibility metadata. App-only tools
  // (visibility:["app"]) are the in-iframe refresh/poll channel and are proxied
  // directly. Model-visible/default tools are promoted to the normal chat turn
  // so the agent owns the visible tool-call lifecycle. This is protocol-based:
  // no MCP-specific tool names or result keys are inspected.
  const handleCallTool = useCallback<NonNullable<AppRendererProps["onCallTool"]>>(async (params) => {
    const requestedToolName = typeof params.name === "string" && params.name ? params.name : toolName;
    const args = typeof params.arguments === "object" && params.arguments !== null
      ? (params.arguments as Record<string, unknown>)
      : {};
    const requestedTool = getMcpToolForAppCall(namespace, serverName, requestedToolName);

    if (requestedTool?.agentVisible && onVisibleToolCall) {
      await onVisibleToolCall({
        namespace,
        serverName,
        toolName: requestedToolName,
        arguments: args,
        sourceToolName: toolName,
      });
      return {
        content: [{
          type: "text",
          text: `Tool ${requestedToolName} was sent to the agent for the regular tool-call lifecycle.`,
        }],
      };
    }

    if (requestedTool && !requestedTool.appOnly && !requestedTool.agentVisible) {
      throw new Error(`MCP App requested tool ${requestedToolName}, but it is not configured as app-only or agent-visible.`);
    }

    return fetchMcpAppApi<CallToolResult>(
      `${mcpAppApiPath(namespace, serverName)}/tools/${encodeURIComponent(requestedToolName)}/call`,
      {
        method: "POST",
        body: JSON.stringify({ arguments: args }),
      },
    );
  }, [getMcpToolForAppCall, namespace, onVisibleToolCall, serverName, toolName]);

  // ui/message: the app asks the host to inject content into the conversation.
  // We forward it as a new chat turn so the agent can react (e.g. start
  // monitoring a build the app just triggered).
  const handleMessage = useCallback<NonNullable<AppRendererProps["onMessage"]>>(async (params) => {
    const text = extractMessageText(params.content as ContentBlock[] | undefined);
    if (!text) {
      return { isError: true };
    }
    if (!onSendMessage) {
      return { isError: true };
    }
    try {
      await onSendMessage(text);
      return {};
    } catch {
      return { isError: true };
    }
  }, [onSendMessage]);

  // Gracefully accept protocol requests we don't act on (e.g.
  // ui/update-model-context) so apps that send them don't surface errors.
  const handleFallbackRequest = useCallback<NonNullable<AppRendererProps["onFallbackRequest"]>>(async () => {
    return {};
  }, []);

  const handleOpenLink = useCallback<NonNullable<AppRendererProps["onOpenLink"]>>(async ({ url }) => {
    const target = new URL(String(url), window.location.href);
    if (target.protocol !== "http:" && target.protocol !== "https:") {
      const message = `Blocked unsupported link scheme: ${target.protocol}`;
      setError(message);
      throw new Error(message);
    }
    window.open(target.toString(), "_blank", "noopener,noreferrer");
    return {};
  }, []);

  const handleError = useCallback<NonNullable<AppRendererProps["onError"]>>((err) => {
    setError(err.message);
  }, []);

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>MCP App error</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (!sandboxUrl) {
    return <div className="rounded-md border p-4 text-sm text-muted-foreground">Preparing MCP App renderer...</div>;
  }

  return (
    <div className="isolate w-full overflow-x-auto overflow-y-visible overscroll-x-contain rounded-lg border bg-background [&_iframe]:w-full [&_iframe]:min-w-[760px]">
      <AppRenderer
        toolName={toolName}
        toolResourceUri={toolResourceUri}
        toolInput={toolInput}
        toolResult={toolResult}
        sandbox={sandbox as NonNullable<AppRendererProps["sandbox"]>}
        hostContext={hostContext}
        hostCapabilities={hostCapabilities}
        onReadResource={handleReadResource}
        onCallTool={handleCallTool}
        onMessage={handleMessage}
        onFallbackRequest={handleFallbackRequest}
        onOpenLink={handleOpenLink}
        onError={handleError}
      />
    </div>
  );
}
