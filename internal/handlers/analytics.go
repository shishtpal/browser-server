package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func BatchUpsertUsage(w http.ResponseWriter, r *http.Request) {
	var req models.UsageBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	merged := make(map[string]int)
	for _, e := range req.Entries {
		if e.Domain == "" || e.Date == "" {
			continue
		}
		key := fmt.Sprintf("%s|%s", e.Domain, e.Date)
		merged[key] += e.Seconds
	}

	upserted := 0
	for key, seconds := range merged {
		parts := splitKey(key)
		if len(parts) != 2 {
			continue
		}
		_, err := db.UsageDB.Exec(
			`INSERT INTO domain_usage (user_id, domain, date, total_seconds)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(user_id, domain, date)
			 DO UPDATE SET total_seconds = total_seconds + excluded.total_seconds, updated_at = CURRENT_TIMESTAMP`,
			req.UserID, parts[0], parts[1], seconds,
		)
		if err == nil {
			upserted++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.UsageBatchResponse{Upserted: upserted})
}

func GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	groupBy := r.URL.Query().Get("group_by")
	limit := helpers.GetLimitFromQuery(r, 10)

	if userID <= 0 || startDate == "" || endDate == "" {
		http.Error(w, "user_id, start_date, and end_date are required", http.StatusBadRequest)
		return
	}

	if groupBy == "" {
		groupBy = "day"
	}

	var totalSeconds int
	err := db.UsageDB.QueryRow(
		`SELECT COALESCE(SUM(total_seconds), 0) FROM domain_usage
		 WHERE user_id = ? AND date >= ? AND date <= ?`,
		userID, startDate, endDate,
	).Scan(&totalSeconds)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	domains := []models.DomainUsage{}
	rows, err := db.UsageDB.Query(
		`SELECT domain, SUM(total_seconds) as total_seconds
		 FROM domain_usage
		 WHERE user_id = ? AND date >= ? AND date <= ?
		 GROUP BY domain
		 ORDER BY total_seconds DESC
		 LIMIT ?`,
		userID, startDate, endDate, limit,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d models.DomainUsage
			if err := rows.Scan(&d.Domain, &d.TotalSeconds); err != nil {
				continue
			}
			if totalSeconds > 0 {
				d.Percentage = math.Round(float64(d.TotalSeconds)/float64(totalSeconds)*1000) / 10
			}
			domains = append(domains, d)
		}
	}

	timeline := []models.TimelinePoint{}
	var groupExpr string
	switch groupBy {
	case "week":
		groupExpr = "strftime('%Y-%W', date)"
	case "month":
		groupExpr = "substr(date, 1, 7)"
	default:
		groupExpr = "date"
	}
	query := fmt.Sprintf(
		`SELECT %s as period, SUM(total_seconds) as total_seconds
		 FROM domain_usage
		 WHERE user_id = ? AND date >= ? AND date <= ?
		 GROUP BY period
		 ORDER BY period`,
		groupExpr,
	)
	timelineRows, err := db.UsageDB.Query(query, userID, startDate, endDate)
	if err == nil {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var tp models.TimelinePoint
			if err := timelineRows.Scan(&tp.Period, &tp.TotalSeconds); err != nil {
				continue
			}
			timeline = append(timeline, tp)
		}
	}

	summary := models.AnalyticsSummary{
		TotalSeconds: totalSeconds,
		Domains:      domains,
		Timeline:     timeline,
	}

	json.NewEncoder(w).Encode(summary)
}

func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return nil
}
