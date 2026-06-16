package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

var StartedAt time.Time

type healthResponse struct {
	Status string  `json:"status"`
	Uptime float64 `json:"uptime_seconds"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{
		Status: "ok",
		Uptime: time.Since(StartedAt).Seconds(),
	})
}
