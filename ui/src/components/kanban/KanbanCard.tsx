"use client";

import { Bot } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { KanbanCard as KanbanCardType } from "@/types/kanban";

interface KanbanCardProps {
  card: KanbanCardType;
}

const PRIORITY_DOT: Record<KanbanCardType["priority"], string> = {
  low:    "bg-slate-400",
  normal: "bg-blue-400",
  high:   "bg-orange-400",
  urgent: "bg-red-500",
};

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function KanbanCard({ card }: KanbanCardProps) {
  return (
    <div className="rounded-lg border bg-card p-3 shadow-sm space-y-2 text-sm">
      <div className="flex items-start justify-between gap-2">
        <span className="font-medium leading-tight">{card.title}</span>
        <span
          className={`mt-1 h-2 w-2 rounded-full flex-shrink-0 ${PRIORITY_DOT[card.priority]}`}
          aria-label={`Priority: ${card.priority}`}
        />
      </div>

      {card.agentName && (
        <div className="flex items-center gap-1 text-muted-foreground">
          <Bot className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{card.agentName}</span>
        </div>
      )}

      <div className="flex items-center justify-between">
        <Badge variant="outline" className="text-xs px-1.5 py-0">
          {card.namespace}
        </Badge>
        <span className="text-xs text-muted-foreground">{relativeTime(card.updatedAt)}</span>
      </div>
    </div>
  );
}
