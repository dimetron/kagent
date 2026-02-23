"use client";

import { useEffect, useState, useCallback } from "react";
import { KanbanSquare, RefreshCw } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { KanbanColumn } from "./KanbanColumn";
import { useNamespace } from "@/contexts/NamespaceContext";
import { getKanbanCards } from "@/app/actions/kanban";
import { KanbanCard, KANBAN_STAGES } from "@/types/kanban";

export function KanbanBoard() {
  const { namespace } = useNamespace();
  const [cards, setCards] = useState<KanbanCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCards = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getKanbanCards(namespace);
      if (response.error || !response.data) {
        setError(response.error || "Failed to load kanban cards");
      } else {
        setCards(response.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load kanban cards");
    } finally {
      setLoading(false);
    }
  }, [namespace]);

  useEffect(() => {
    fetchCards();
  }, [fetchCards]);

  const cardsByStage = (stage: KanbanCard["stage"]) => cards.filter((c) => c.stage === stage);

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-4 pt-4 pb-3 border-b flex-shrink-0">
        <KanbanSquare className="h-5 w-5" />
        <h1 className="text-lg font-semibold">Kanban</h1>
      </div>

      {loading ? (
        <div className="flex gap-4 p-4 overflow-x-auto flex-1">
          {KANBAN_STAGES.map((s) => (
            <div key={s.stage} className="flex flex-col w-64 flex-shrink-0 gap-2">
              <Skeleton className="h-10 w-full rounded-sm" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </div>
          ))}
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-3 text-muted-foreground">
          <p className="text-sm">{error}</p>
          <Button variant="outline" size="sm" onClick={fetchCards}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
        </div>
      ) : (
        <div
          className="flex gap-4 p-4 overflow-x-auto flex-1"
          role="region"
          aria-label="Kanban board"
        >
          {KANBAN_STAGES.map((s) => (
            <KanbanColumn
              key={s.stage}
              stage={s.stage}
              label={s.label}
              cards={cardsByStage(s.stage)}
              colorClass={s.colorClass}
            />
          ))}
        </div>
      )}
    </div>
  );
}
