package seeds

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/romanzipp/domaindex/internal/models"
	"gorm.io/gorm"
)

//go:embed data/prices/*.csv
var priceFiles embed.FS

func seedRegistrarPrices(db *gorm.DB, userID uint) error {
	return fs.WalkDir(priceFiles, "data/prices", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".csv" {
			return err
		}

		ianaID := strings.TrimSuffix(filepath.Base(path), ".csv")

		var registrar models.Registrar
		if err := db.Where("user_id = ? AND iana_id = ?", userID, ianaID).First(&registrar).Error; err != nil {
			return nil // registrar not seeded for this user, skip
		}

		key := fmt.Sprintf("prices_%s_%d", ianaID, userID)
		if hasRun(db, key) {
			return nil
		}

		prices, err := parsePriceCSV(priceFiles, path, registrar.ID)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		if err := db.CreateInBatches(prices, 100).Error; err != nil {
			return fmt.Errorf("insert prices %s: %w", path, err)
		}

		markRun(db, key)
		return nil
	})
}

func parsePriceCSV(fsys embed.FS, path string, registrarID uint) ([]models.Price, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	prices := make([]models.Price, 0, len(records)-1)
	for _, row := range records[1:] { // skip header
		if len(row) < 5 {
			continue
		}
		p := models.Price{
			RegistrarID:    &registrarID,
			TLD:            row[0],
			InitialPerYear: parseFloat(row[1]),
			RenewPerYear:   parseFloat(row[2]),
			Transfer:       parseFloat(row[3]),
			PrivacyPerYear: parseFloat(row[4]),
		}
		prices = append(prices, p)
	}
	return prices, nil
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
