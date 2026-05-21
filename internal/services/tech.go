package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

	d.TechDNSProvider = s.fetchDNSProvider(d.Name)

	d.TechFetchedAt = &now
	return nil
}

func (s *TechService) fetchDNSProvider(domain string) string {
	servers, err := net.LookupNS(domain)
	if err != nil || len(servers) == 0 {
		return ""
	}
	hosts := make([]string, 0, len(servers))
	for _, ns := range servers {
		hosts = append(hosts, strings.ToLower(strings.TrimSuffix(ns.Host, ".")))
	}
	return detectDNSProvider(hosts)
}

// detectDNSProvider maps nameserver hostnames to a human-readable provider name.
// Returns the base domain of the first nameserver as fallback.
func detectDNSProvider(nameservers []string) string {
	if len(nameservers) == 0 {
		return ""
	}
	for _, ns := range nameservers {
		for _, p := range dnsProviderPatterns {
			if strings.Contains(ns, p.match) {
				return p.name
			}
		}
	}
	return baseDomain(nameservers[0])
}

var dnsProviderPatterns = []struct {
	match string
	name  string
}{
	{"cloudflare.com", "Cloudflare"},
	{"awsdns", "AWS Route 53"},
	{"azure-dns", "Azure DNS"},
	{"googledomains.com", "Google Cloud DNS"},
	{"google.com", "Google Cloud DNS"},
	{"domaincontrol.com", "GoDaddy"},
	{"registrar-servers.com", "Namecheap"},
	{"namecheaphosting.com", "Namecheap"},
	{"hetzner.com", "Hetzner"},
	{"hetzner.de", "Hetzner"},
	{"first-ns.de", "Hetzner"},
	{"digitalocean.com", "DigitalOcean"},
	{"dnsimple.com", "DNSimple"},
	{"nsone.net", "NS1"},
	{"vercel-dns.com", "Vercel"},
	{"netlify.com", "Netlify"},
	{"gandi.net", "Gandi"},
	{"ovh.net", "OVH"},
	{"porkbun.com", "Porkbun"},
	{"he.net", "Hurricane Electric"},
	{"name.com", "Name.com"},
	{"dynadot.com", "Dynadot"},
	{"namesilo.com", "NameSilo"},
	{"spaceship.com", "Spaceship"},
	{"sav.com", "Sav"},
	{"linode.com", "Linode"},
	{"akamai", "Akamai"},
	{"fastly.net", "Fastly"},
	{"easydns.com", "easyDNS"},
	{"hover.com", "Hover"},
	{"rackspace.com", "Rackspace"},
	{"alibabadns.com", "Alibaba Cloud DNS"},
	{"dnspod.net", "DNSPod"},
	{"yandex.net", "Yandex"},
}

func baseDomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
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
