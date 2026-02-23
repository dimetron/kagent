import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { AppSidebarNav } from "./AppSidebarNav";
import { NamespaceSelector } from "./NamespaceSelector";
import { StatusIndicator } from "./StatusIndicator";
import KAgentLogoWithText from "@/components/kagent-logo-text";
import KagentLogo from "@/components/kagent-logo";

export function AppSidebar() {
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        {/* Full logo — visible when expanded */}
        <div className="px-2 py-2 group-data-[collapsible=icon]:hidden">
          <KAgentLogoWithText className="h-5 w-auto" />
        </div>
        {/* Icon only — visible when collapsed */}
        <div className="hidden group-data-[collapsible=icon]:flex justify-center py-2">
          <KagentLogo className="h-6 w-6" />
        </div>
        <NamespaceSelector />
      </SidebarHeader>

      <SidebarContent>
        <AppSidebarNav />
      </SidebarContent>

      <SidebarFooter>
        <SidebarSeparator />
        <StatusIndicator />
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}
