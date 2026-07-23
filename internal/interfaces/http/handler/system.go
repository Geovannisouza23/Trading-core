package handler

import (
	"encoding/json"
	"net/http"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	httpmw "trading-core/internal/interfaces/http/middleware"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
	"trading-core/internal/interfaces/http/response"
)

// SystemHandler serves the /v1/system/* and /v1/paper/reset control
// endpoints. Every action here is audited with the caller's identity (from
// the validated JWT), never a client-supplied field.
type SystemHandler struct {
	changeMode input.ChangeOperationalModeUseCase
	killSwitch input.ActivateKillSwitchUseCase
	resetPaper input.ResetPaperStateUseCase
	homeMode   string
}

func NewSystemHandler(
	changeMode input.ChangeOperationalModeUseCase,
	killSwitch input.ActivateKillSwitchUseCase,
	resetPaper input.ResetPaperStateUseCase,
	homeMode string,
) *SystemHandler {
	return &SystemHandler{changeMode: changeMode, killSwitch: killSwitch, resetPaper: resetPaper, homeMode: homeMode}
}

func actorFrom(r *http.Request) string {
	claims, ok := httpmw.ClaimsFromContext(r.Context())
	if !ok || claims.Subject == "" {
		return "unknown"
	}
	return claims.Subject
}

func decodeReason(r *http.Request) string {
	var body request.ReasonRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		return "not specified"
	}
	return body.Reason
}

func (h *SystemHandler) Pause(w http.ResponseWriter, r *http.Request) {
	err := h.changeMode.Execute(r.Context(), command.ChangeOperationalModeCommand{
		TargetMode: "PAUSED", ActedBy: actorFrom(r), Origin: "http", Reason: decodeReason(r),
	})
	respondSystemAction(w, r, err)
}

// Resume always targets the deployment's configured "home" mode (e.g.
// PAPER or TESTNET) rather than an arbitrary caller-chosen mode: spec
// section 15 exposes no endpoint to pick an arbitrary target, and REAL can
// never be reached this way (see internal/application/usecase/
// change_operational_mode.go).
func (h *SystemHandler) Resume(w http.ResponseWriter, r *http.Request) {
	err := h.changeMode.Execute(r.Context(), command.ChangeOperationalModeCommand{
		TargetMode: h.homeMode, ActedBy: actorFrom(r), Origin: "http", Reason: decodeReason(r),
	})
	respondSystemAction(w, r, err)
}

func (h *SystemHandler) CloseOnly(w http.ResponseWriter, r *http.Request) {
	err := h.changeMode.Execute(r.Context(), command.ChangeOperationalModeCommand{
		TargetMode: "CLOSE_ONLY", ActedBy: actorFrom(r), Origin: "http", Reason: decodeReason(r),
	})
	respondSystemAction(w, r, err)
}

func (h *SystemHandler) KillSwitch(w http.ResponseWriter, r *http.Request) {
	var body request.KillSwitchRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		body.Reason = "manual activation via API"
	}
	err := h.killSwitch.Execute(r.Context(), command.ActivateKillSwitchCommand{
		ActivatedBy: actorFrom(r), Origin: "http", Reason: body.Reason, AllowClose: body.AllowClose,
	})
	respondSystemAction(w, r, err)
}

func (h *SystemHandler) PaperReset(w http.ResponseWriter, r *http.Request) {
	respondSystemAction(w, r, h.resetPaper.Execute(r.Context()))
}

func respondSystemAction(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, response.StatusResponse{Status: "ok"})
}
