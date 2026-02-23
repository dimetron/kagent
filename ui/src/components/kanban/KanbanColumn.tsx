"use client";

import { Inbox } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { KanbanCard as KanbanCardComponent } from "./KanbanCard";
import { KanbanCard, KanbanStage } from "@/types/kanban";

interface KanbanColumnProps {
  stage: KanbanStage;
  label: string;
  cards: KanbanCard[];
  colorClass: string;
}

export function KanbanColumn({ stage, label, cards, colorClass }: KanbanColumnProps) {
  const headerId = `kanban-col-${stage}`;
  return (
    <div className="flex flex-col w-64 flex-shrink-0 h-full">
      <div
        className={`border-t-4 ${colorClass} rounded-t-sm bg-muted/40 px-3 py-2 flex items-center justify-between`}
        id={headerId}
      >
        <span className="text-sm font-semibold">{label}</span>
        <Badge variant="secondary" className="text-xs">
          {cards.length}
        </Badge>
      </div>

      <div
        className="flex-1 overflow-y-auto flex flex-col gap-2 p-2 bg-muted/20 rounded-b-sm"
        aria-labelledby={headerId}
      >
        {cards.length === 0 ? (
          <div
            className="flex flex-col items-center justify-center h-32 gap-2 text-muted-foreground"
            aria-label={`No tasks in ${label} stage`}
          >
            <Inbox className="h-6 w-6" />
            <span className="text-xs">No tasks here</span>
          </div>
        ) : (
          cards.map((card) => <KanbanCardComponent key={card.id} card={card} />)
        )}
      </div>
    </div>
  );
}
