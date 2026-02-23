"use server";

import { BaseResponse } from "@/types";
import { KanbanCard } from "@/types/kanban";
import { fetchApi, createErrorResponse } from "./utils";

interface KanbanApiResponse {
  error: boolean;
  data: KanbanCard[];
  message: string;
}

export async function getKanbanCards(namespace: string): Promise<BaseResponse<KanbanCard[]>> {
  try {
    const response = await fetchApi<KanbanApiResponse>("/kanban");

    if (response.error || !response.data) {
      return { message: response.message || "Failed to fetch kanban cards", error: response.message };
    }

    const cards = namespace
      ? response.data.filter((c) => c.namespace === namespace)
      : response.data;

    return { message: "Kanban cards fetched successfully", data: cards };
  } catch (error) {
    return createErrorResponse<KanbanCard[]>(error, "Error fetching kanban cards");
  }
}
