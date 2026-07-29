package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"browser-server/internal/bookmark"
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
		helpers.WriteError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Missing 'file' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Failed to read file")
		return
	}

	existingURLs, err := bookmark.ExistingURLs(r.Context(), userID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to load existing bookmarks: "+err.Error())
		return
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Failed to parse HTML file")
		return
	}

	var records []importRecord
	var skipped int
	walkTree(doc, "", &records, existingURLs, &skipped)

	imported := []models.BookmarkResponse{}
	var importErrors []string
	for _, rec := range records {
		id, inserted, err := bookmark.Create(r.Context(), bookmark.CreateInput{
			UserID: userID, Title: rec.Title, URL: rec.URL, FolderPath: rec.FolderPath,
		})
		if err != nil {
			importErrors = append(importErrors, rec.URL)
			continue
		}
		if !inserted {
			skipped++
			continue
		}
		now := time.Now()
		imported = append(imported, bookmark.Response(models.Bookmark{
			ID: int(id), UserID: userID, Title: rec.Title, URL: rec.URL,
			Tags: "[]", FolderPath: rec.FolderPath, CreatedAt: now, UpdatedAt: now,
		}))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.ImportResult{
		Imported:  len(imported),
		Skipped:   skipped,
		Bookmarks: imported,
		Errors:    importErrors,
	})
}

type importRecord struct {
	Title      string
	URL        string
	FolderPath string
}

func walkTree(n *html.Node, folderPath string, records *[]importRecord, existingURLs map[string]struct{}, skipped *int) {
	if n.Type == html.ElementNode && n.Data == "dl" {
		walkDL(n, folderPath, records, existingURLs, skipped)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkTree(c, folderPath, records, existingURLs, skipped)
	}
}

func walkDL(n *html.Node, folderPath string, records *[]importRecord, existingURLs map[string]struct{}, skipped *int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "dt" {
			continue
		}
		aTag := findChild(c, "a")
		if aTag != nil {
			url := getAttr(aTag, "href")
			title := strings.TrimSpace(getText(aTag))
			if url != "" && url != "javascript:void(0)" {
				if _, exists := existingURLs[url]; !exists {
					*records = append(*records, importRecord{
						Title:      title,
						URL:        url,
						FolderPath: folderPath,
					})
					existingURLs[url] = struct{}{}
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
