package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"browser-server/internal/helpers"
	"browser-server/internal/history"
	"browser-server/internal/models"
)

const defaultGroupedHistoryLimit = 100

// GetGroupedHistory returns history aggregated by URL, searched and paginated
// entirely on the server so clients never have to load every row at once.
func GetGroupedHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := history.ListGrouped(r.Context(), history.GroupedOptions{
		UserID: helpers.GetUserIDFromQuery(r),
		Search: r.URL.Query().Get("q"),
		Column: r.URL.Query().Get("column"),
		Domain: r.URL.Query().Get("domain"),
		Limit:  helpers.GetLimitFromQuery(r, defaultGroupedHistoryLimit),
		Offset: helpers.GetOffsetFromQuery(r),
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(resp)
}

// GetHistoryDomains returns every hostname represented in history, ordered by
// visit count so the most-used domain appears first.
func GetHistoryDomains(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	domains, err := history.ListDomains(r.Context(),
		helpers.GetUserIDFromQuery(r), r.URL.Query().Get("q"))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(domains)
}

func GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entries, err := history.List(r.Context(), history.ListOptions{
		UserID: helpers.GetUserIDFromQuery(r),
		URL:    r.URL.Query().Get("url"),
		Limit:  helpers.GetLimitFromQuery(r, 0),
		Offset: helpers.GetOffsetFromQuery(r),
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(entries)
}

func CreateHistory(w http.ResponseWriter, r *http.Request) {
	var entry models.History
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", entry.UserID)
	v.URL("url", entry.URL)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	// Don't record browsing history for the server's own web UI.
	if helpers.IsSelfOrigin(entry.URL, ServerPort) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if entry.VisitedAt.IsZero() {
		entry.VisitedAt = time.Now()
	}

	id, err := history.Create(r.Context(), entry)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	entry.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func GetHistoryByID(w http.ResponseWriter, r *http.Request) {
	entry, err := history.GetByID(r.Context(), helpers.GetIDFromPath(r))
	if err == history.ErrNotFound {
		helpers.WriteError(w, http.StatusNotFound, "History entry not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func DeleteHistory(w http.ResponseWriter, r *http.Request) {
	deleted, err := history.Delete(r.Context(), helpers.GetIDFromPath(r))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if !deleted {
		helpers.WriteError(w, http.StatusNotFound, "History entry not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
