package handlers

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return "—"
			}
			return t.Format("2006-01-02")
		},
		"sortLink": func(currentSort, currentDir, col, baseURL string) string {
			dir := "asc"
			if currentSort == col && currentDir == "asc" {
				dir = "desc"
			}
			sep := "?"
			if strings.Contains(baseURL, "?") {
				sep = "&"
			}
			return baseURL + sep + "sort=" + col + "&dir=" + dir
		},
		"sortIndicator": func(currentSort, currentDir, col string) string {
			if currentSort != col {
				return ""
			}
			if currentDir == "asc" {
				return " ↑"
			}
			return " ↓"
		},
		"add": func(a, b int) int { return a + b },
		"daysUntil": func(t *time.Time) string {
			if t == nil {
				return "—"
			}
			days := int(time.Until(*t).Hours() / 24)
			if days < 0 {
				return "expired"
			}
			return fmt.Sprintf("%dd", days)
		},
	}
}
