export type KanbanStage = "Inbox" | "Assigned" | "InProgress" | "Review" | "Done";

export interface KanbanCard {
  id: string;
  title: string;
  stage: KanbanStage;
  agentName?: string;
  namespace: string;
  priority: "low" | "normal" | "high" | "urgent";
  createdAt: string;
  updatedAt: string;
}

export interface KanbanStageConfig {
  stage: KanbanStage;
  label: string;
  colorClass: string;
}

export const KANBAN_STAGES: KanbanStageConfig[] = [
  { stage: "Inbox",      label: "Inbox",       colorClass: "border-slate-400"  },
  { stage: "Assigned",   label: "Assigned",    colorClass: "border-blue-400"   },
  { stage: "InProgress", label: "In Progress", colorClass: "border-yellow-400" },
  { stage: "Review",     label: "Review",      colorClass: "border-purple-400" },
  { stage: "Done",       label: "Done",        colorClass: "border-green-400"  },
];
