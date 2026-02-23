"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Activity,
  Bot,
  GitBranch,
  KanbanSquare,
  Brain,
  Wrench,
  Server,
  Building2,
  Network,
  type LucideIcon,
} from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
  exact?: boolean;
}

interface NavSection {
  title: string;
  items: NavItem[];
}

const NAV_SECTIONS: NavSection[] = [
  {
    title: "OVERVIEW",
    items: [
      { label: "Dashboard", href: "/", icon: LayoutDashboard, exact: true },
      { label: "Live Feed", href: "/feed", icon: Activity },
    ],
  },
  {
    title: "AGENTS",
    items: [
      { label: "My Agents", href: "/agents", icon: Bot },
      { label: "Workflows", href: "/workflows", icon: GitBranch },
      { label: "Kanban", href: "/kanban", icon: KanbanSquare },
    ],
  },
  {
    title: "RESOURCES",
    items: [
      { label: "Models", href: "/models", icon: Brain },
      { label: "Tools", href: "/tools", icon: Wrench },
      { label: "MCP Servers", href: "/servers", icon: Server },
    ],
  },
  {
    title: "ADMIN",
    items: [
      { label: "Organization", href: "/admin/org", icon: Building2 },
      { label: "Gateways", href: "/admin/gateways", icon: Network },
    ],
  },
];

export function AppSidebarNav() {
  const pathname = usePathname();

  const isActive = (item: NavItem) => {
    if (item.exact) return pathname === item.href;
    return pathname === item.href || pathname.startsWith(item.href + "/");
  };

  return (
    <nav aria-label="Main navigation">
      {NAV_SECTIONS.map((section) => (
        <SidebarGroup key={section.title} role="group" aria-labelledby={`nav-section-${section.title}`}>
          <SidebarGroupLabel id={`nav-section-${section.title}`}>{section.title}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {section.items.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive(item)}
                    tooltip={item.label}
                  >
                    <Link href={item.href}>
                      <item.icon />
                      <span>{item.label}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ))}
    </nav>
  );
}
