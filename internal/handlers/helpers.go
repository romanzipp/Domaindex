package handlers

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/romanzipp/domain-manager/internal/services"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return "—"
			}
			return t.Format("2006-01-02")
		},
		"formatDate2": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
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
		"currencySymbol": services.Symbol,
		"formatAmount":   services.FormatAmount,
		"add":            func(a, b int) int { return a + b },
		"derefUint": func(p *uint) uint {
			if p == nil {
				return 0
			}
			return *p
		},
		"derefFloat": func(p *float64) float64 {
			if p == nil {
				return 0
			}
			return *p
		},
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
		"daysInt": func(t *time.Time) int {
			if t == nil {
				return 9999
			}
			return int(time.Until(*t).Hours() / 24)
		},
	}
}
