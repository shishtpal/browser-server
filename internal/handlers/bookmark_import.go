package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func ImportBookmarks(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserIDFromQuery(r)
	if userID == 0 {
		userID = 1
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}

	existingURLs := make(map[string]bool)
	rows, err := db.BookmarkDB.Query("SELECT url FROM bookmarks WHERE user_id = ?", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var url string
			rows.Scan(&url)
			existingURLs[url] = true
		}
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		http.Error(w, "Failed to parse HTML file", http.StatusBadRequest)
		return
	}

	var records []importRecord
	var skipped int
	walkTree(doc, "", &records, existingURLs, &skipped)

	imported := []models.BookmarkResponse{}
	for _, rec := range records {
		result, err := db.BookmarkDB.Exec(
			"INSERT INTO bookmarks (user_id, title, url, description, tags, folder_path) VALUES (?, ?, ?, ?, ?, ?)",
			userID, rec.Title, rec.URL, "", "[]", rec.FolderPath,
		)
		if err != nil {
			continue
		}
		id, _ := result.LastInsertId()
		imported = append(imported, models.BookmarkResponse{
			ID:         int(id),
			UserID:     userID,
			Title:      rec.Title,
			URL:        rec.URL,
			Tags:       []string{},
			FolderPath: rec.FolderPath,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.ImportResult{
		Imported:  len(imported),
		Skipped:   skipped,
		Bookmarks: imported,
	})
}

type importRecord struct {
	Title      string
	URL        string
	FolderPath string
}

func walkTree(n *html.Node, folderPath string, records *[]importRecord, existingURLs map[string]bool, skipped *int) {
	if n.Type == html.ElementNode && n.Data == "dl" {
		walkDL(n, folderPath, records, existingURLs, skipped)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkTree(c, folderPath, records, existingURLs, skipped)
	}
}

func walkDL(n *html.Node, folderPath string, records *[]importRecord, existingURLs map[string]bool, skipped *int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "dt" {
			continue
		}
		aTag := findChild(c, "a")
		if aTag != nil {
			url := getAttr(aTag, "href")
			title := strings.TrimSpace(getText(aTag))
			if url != "" && url != "javascript:void(0)" {
				if !existingURLs[url] {
					*records = append(*records, importRecord{
						Title:      title,
						URL:        url,
						FolderPath: folderPath,
					})
					existingURLs[url] = true
				} else {
					*skipped++
				}
			}
			continue
		}
		h3Tag := findChild(c, "h3")
		if h3Tag != nil {
			folderName := strings.TrimSpace(getText(h3Tag))
			newPath := folderPath
			if folderName != "" {
				if newPath != "" {
					newPath += "/" + folderName
				} else {
					newPath = folderName
				}
			}
			nestedDL := findChild(c, "dl")
			if nestedDL == nil {
				nestedDL = findNextSiblingDL(c)
			}
			if nestedDL != nil {
				walkDL(nestedDL, newPath, records, existingURLs, skipped)
			}
		}
	}
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func findNextSiblingDL(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && s.Data == "dl" {
			return s
		}
		if s.Type == html.ElementNode && s.Data == "dt" {
			if dl := findChild(s, "dl"); dl != nil {
				return dl
			}
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func getText(n *html.Node) string {
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return buf.String()
}
