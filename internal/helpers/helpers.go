package helpers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetUserIDFromQuery(r *http.Request) int {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		return 0
	}
	userID, _ := strconv.Atoi(userIDStr)
	return userID
}

func GetIDFromPath(r *http.Request) int {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	return id
}

func GetIDFromQuery(r *http.Request) int {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return 0
	}
	id, _ := strconv.Atoi(idStr)
	return id
}

func ParseTagsFromJSON(tagsJSON string) []string {
	if tagsJSON == "" {
		return []string{}
	}
	var tags []string
	json.Unmarshal([]byte(tagsJSON), &tags)
	return tags
}

func TagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	jsonBytes, _ := json.Marshal(tags)
	return string(jsonBytes)
}
