package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func TestHistoryDomainsAndExactDomainFilter(t *testing.T) {
	db.InitHistoryDB(t.TempDir())
	t.Cleanup(func() {
		db.HistoryDB.Close()
		db.HistoryDB = nil
	})

	visitedAt := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	rows := []struct {
		userID   int
		url      string
		title    string
		duration int
	}{
		{1, "https://user:pass@example.com:8443/first", "First", 10},
		{1, "https://user:pass@example.com:8443/first", "First again", 20},
		{1, "https://example.com?section=second", "Second", 30},
		{1, "https://not-example.com/page", "Other", 40},
		{2, "https://example.com/private", "Another user", 50},
	}
	for _, row := range rows {
		if _, err := db.HistoryDB.Exec(
			"INSERT INTO history (user_id, url, domain, title, visited_at, duration) VALUES (?, ?, ?, ?, ?, ?)",
			row.userID, row.url, helpers.URLHostname(row.url), row.title, visitedAt, row.duration,
		); err != nil {
			t.Fatalf("insert history: %v", err)
		}
	}

	domainRequest := httptest.NewRequest(http.MethodGet, "/api/history/domains?user_id=1", nil)
	domainResponse := httptest.NewRecorder()
	GetHistoryDomains(domainResponse, domainRequest)
	if domainResponse.Code != http.StatusOK {
		t.Fatalf("domain status = %d, want %d: %s", domainResponse.Code, http.StatusOK, domainResponse.Body.String())
	}
	var domains []models.HistoryDomainSummary
	if err := json.NewDecoder(domainResponse.Body).Decode(&domains); err != nil {
		t.Fatalf("decode domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("domain count = %d, want 2", len(domains))
	}
	if domains[0].Domain != "example.com" || domains[0].VisitCount != 3 || domains[0].URLCount != 2 || domains[0].TotalDuration != 60 {
		t.Fatalf("top domain = %+v, want example.com with 3 visits, 2 URLs, and 60 seconds", domains[0])
	}

	groupedRequest := httptest.NewRequest(http.MethodGet, "/api/history/grouped?user_id=1&domain=example.com", nil)
	groupedResponse := httptest.NewRecorder()
	GetGroupedHistory(groupedResponse, groupedRequest)
	if groupedResponse.Code != http.StatusOK {
		t.Fatalf("grouped status = %d, want %d: %s", groupedResponse.Code, http.StatusOK, groupedResponse.Body.String())
	}
	var grouped models.GroupedHistoryResponse
	if err := json.NewDecoder(groupedResponse.Body).Decode(&grouped); err != nil {
		t.Fatalf("decode grouped history: %v", err)
	}
	if grouped.Total != 2 || len(grouped.Entries) != 2 {
		t.Fatalf("exact-domain grouped result = total %d, entries %d; want 2 and 2", grouped.Total, len(grouped.Entries))
	}
}
