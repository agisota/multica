package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

type SubmitContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Message string `json:"message"`
}

func (h *Handler) SubmitContactMessage(w http.ResponseWriter, r *http.Request) {
	var req SubmitContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Company = strings.TrimSpace(req.Company)
	req.Message = strings.TrimSpace(req.Message)

	switch {
	case req.Name == "":
		writeError(w, http.StatusBadRequest, "name is required")
		return
	case req.Email == "" || !strings.Contains(req.Email, "@"):
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	case req.Message == "":
		writeError(w, http.StatusBadRequest, "message is required")
		return
	case len(req.Name) > 120:
		writeError(w, http.StatusBadRequest, "name is too long")
		return
	case len(req.Company) > 160:
		writeError(w, http.StatusBadRequest, "company is too long")
		return
	case len(req.Message) > 4000:
		writeError(w, http.StatusBadRequest, "message is too long")
		return
	}

	if err := h.EmailService.SendLandingContactMessage(req.Name, req.Email, req.Company, req.Message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
