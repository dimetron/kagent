package handlers

import (
	"net/http"
	"time"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/internal/httpserver/errors"
	"github.com/kagent-dev/kagent/go/internal/utils"
	"github.com/kagent-dev/kagent/go/pkg/client/api"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// KanbanHandler handles kanban-related requests
type KanbanHandler struct {
	*Base
}

// NewKanbanHandler creates a new KanbanHandler
func NewKanbanHandler(base *Base) *KanbanHandler {
	return &KanbanHandler{Base: base}
}

// agentToKanbanStage derives a KanbanStage from an agent's status conditions.
//
// Stage mapping:
//
//	!accepted && !deploymentReady → Inbox
//	accepted  && !deploymentReady → Assigned
//	accepted  && deploymentReady  → InProgress
func agentToKanbanStage(accepted, deploymentReady bool) api.KanbanStage {
	if !accepted {
		return api.KanbanStageInbox
	}
	if !deploymentReady {
		return api.KanbanStageAssigned
	}
	return api.KanbanStageInProgress
}

// HandleListKanbanCards handles GET /api/kanban requests.
// It lists all agents and maps them to kanban cards.
func (h *KanbanHandler) HandleListKanbanCards(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("kanban-handler").WithValues("operation", "list")

	agentList := &v1alpha2.AgentList{}
	if err := h.KubeClient.List(r.Context(), agentList); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to list Agents from Kubernetes", err))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	cards := make([]api.KanbanCard, 0, len(agentList.Items))

	for _, agent := range agentList.Items {
		agentRef := utils.GetObjectRef(&agent)

		deploymentReady := false
		accepted := false
		for _, condition := range agent.Status.Conditions {
			if condition.Type == "Ready" && condition.Reason == "DeploymentReady" && condition.Status == "True" {
				deploymentReady = true
			}
			if condition.Type == "Accepted" && condition.Status == "True" {
				accepted = true
			}
		}

		cards = append(cards, api.KanbanCard{
			ID:        utils.ConvertToPythonIdentifier(agentRef),
			Title:     agent.Name,
			Stage:     agentToKanbanStage(accepted, deploymentReady),
			AgentName: agent.Name,
			Namespace: agent.Namespace,
			Priority:  "normal",
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	log.Info("Successfully listed kanban cards", "count", len(cards))
	data := api.NewResponse(cards, "Successfully listed kanban cards", false)
	RespondWithJSON(w, http.StatusOK, data)
}
