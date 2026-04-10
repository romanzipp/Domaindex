package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/romanzipp/domaindex/internal/models"
	"gorm.io/gorm"
)

type TechService struct {
	db     *gorm.DB
	client *http.Client
}

func NewTechService(db *gorm.DB) *TechService {
	return &TechService{
		db:     db,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *TechService) UpdateDomain(d *models.Domain) error {
	now := time.Now()

	aRecords, aaaaRecords, _ := s.fetchDNS(d.Name)

	aJSON, _ := json.Marshal(aRecords)
	aaaaJSON, _ := json.Marshal(aaaaRecords)
	d.TechARecords = string(aJSON)
	d.TechAAAARecords = string(aaaaJSON)

	if len(aRecords) > 0 {
		asn, asnOrg, country, _ := s.fetchASN(aRecords[0])
		d.TechASN = asn
		d.TechASNOrg = asnOrg
		d.TechCountry = country
	}

	sslEnabled, sslExpiry, sslIssuer, _ := s.fetchSSL(d.Name)
	d.TechSSLEnabled = sslEnabled
	d.TechSSLExpiry = sslExpiry
	d.TechSSLIssuer = sslIssuer

	d.TechFetchedAt = &now
	return nil
}

func (s *TechService) fetchDNS(domain string) (aRecords, aaaaRecords []string, err error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, nil, err
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			aRecords = append(aRecords, v4.String())
		} else {
			aaaaRecords = append(aaaaRecords, ip.String())
		}
	}
	return aRecords, aaaaRecords, nil
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	AS          string `json:"as"`
	Org         string `json:"org"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

func (s *TechService) fetchASN(ip string) (asn, org, country string, err error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,as,org,country,countryCode", ip)
	resp, err := s.client.Get(url)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var data ipAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", "", err
	}
	if data.Status != "success" {
		return "", "", "", fmt.Errorf("ip-api: status %s", data.Status)
	}

	return data.AS, data.Org, data.CountryCode, nil
}

func (s *TechService) fetchSSL(domain string) (enabled bool, expiry *time.Time, issuer string, err error) {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		domain+":443",
		&tls.Config{InsecureSkipVerify: true}, //nolint:gosec // intentional: we want cert details even for invalid/expired certs
	)
	if err != nil {
		return false, nil, "", err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return true, nil, "", nil
	}

	leaf := certs[0]
	t := leaf.NotAfter

	org := ""
	if len(leaf.Issuer.Organization) > 0 {
		org = leaf.Issuer.Organization[0]
	} else {
		org = leaf.Issuer.CommonName
	}

	return true, &t, org, nil
}
