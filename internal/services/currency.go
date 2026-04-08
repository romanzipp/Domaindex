package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CurrencyService struct {
	mu        sync.RWMutex
	rates     map[string]float64 // all relative to EUR
	fetchedAt time.Time
	ttl       time.Duration
}

func NewCurrencyService() *CurrencyService {
	s := &CurrencyService{
		rates: map[string]float64{"EUR": 1.0},
		ttl:   24 * time.Hour,
	}
	go s.refresh()
	return s
}

// Convert converts amount from one currency to another.
// Returns the original amount unchanged if either currency is unknown.
func (s *CurrencyService) Convert(amount float64, from, to string) float64 {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if from == to || amount == 0 {
		return amount
	}

	s.mu.RLock()
	stale := time.Since(s.fetchedAt) > s.ttl
	fromRate, fromOK := s.rates[from]
	toRate, toOK := s.rates[to]
	s.mu.RUnlock()

	if stale {
		go s.refresh()
	}

	if !fromOK || !toOK || fromRate == 0 {
		return amount
	}

	// Convert via EUR as base: amount_in_EUR = amount / fromRate, then * toRate
	return amount / fromRate * toRate
}

func (s *CurrencyService) refresh() {
	resp, err := http.Get("https://api.frankfurter.app/latest?base=EUR")
	if err != nil {
		log.Printf("currency: fetch rates: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("currency: decode rates: %v", err)
		return
	}

	result.Rates["EUR"] = 1.0

	s.mu.Lock()
	s.rates = result.Rates
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	log.Printf("currency: updated %d exchange rates", len(result.Rates))
}

// Symbol returns a short symbol for common currencies, falls back to the code.
func Symbol(currency string) string {
	symbols := map[string]string{
		"USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥",
		"CHF": "Fr", "CAD": "C$", "AUD": "A$", "NZD": "NZ$",
		"SEK": "kr", "NOK": "kr", "DKK": "kr", "PLN": "zł",
		"CZK": "Kč", "HUF": "Ft", "RON": "lei", "BGN": "лв",
	}
	if s, ok := symbols[strings.ToUpper(currency)]; ok {
		return s
	}
	return currency
}

// FormatAmount formats a float with 2 decimal places and the currency symbol.
func FormatAmount(amount float64, currency string) string {
	return fmt.Sprintf("%s %.2f", Symbol(currency), amount)
}
