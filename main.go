package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Port               string
	CloudflareAPIToken string
	CloudflareZoneID   string
	BasicAuthUsername  string
	BasicAuthPassword  string
}

// CloudflareListResponse represents the Cloudflare API response for list operations
type CloudflareListResponse struct {
	Success  bool                  `json:"success"`
	Errors   []CloudflareError     `json:"errors"`
	Messages []CloudflareMessage   `json:"messages"`
	Result   []CloudflareDNSRecord `json:"result"`
}

// CloudflareSingleResponse represents the Cloudflare API response for single record operations
type CloudflareSingleResponse struct {
	Success  bool                `json:"success"`
	Errors   []CloudflareError   `json:"errors"`
	Messages []CloudflareMessage `json:"messages"`
	Result   CloudflareDNSRecord `json:"result"`
}

// CloudflareError represents an error from Cloudflare API
type CloudflareError struct {
	Code             int                    `json:"code"`
	Message          string                 `json:"message"`
	DocumentationURL string                 `json:"documentation_url,omitempty"`
	Source           map[string]interface{} `json:"source,omitempty"`
}

// CloudflareMessage represents a message from Cloudflare API
type CloudflareMessage struct {
	Code             int                    `json:"code"`
	Message          string                 `json:"message"`
	DocumentationURL string                 `json:"documentation_url,omitempty"`
	Source           map[string]interface{} `json:"source,omitempty"`
}

// CloudflareDNSRecord represents a DNS record
type CloudflareDNSRecord struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	Name              string                 `json:"name"`
	Content           string                 `json:"content"`
	TTL               int                    `json:"ttl"`
	Proxied           bool                   `json:"proxied"`
	Proxiable         bool                   `json:"proxiable,omitempty"`
	Comment           string                 `json:"comment,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	CreatedOn         string                 `json:"created_on,omitempty"`
	ModifiedOn        string                 `json:"modified_on,omitempty"`
	CommentModifiedOn string                 `json:"comment_modified_on,omitempty"`
	TagsModifiedOn    string                 `json:"tags_modified_on,omitempty"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	Settings          map[string]interface{} `json:"settings,omitempty"`
}

// CloudflareUpdateRequest represents the request to update a DNS record
type CloudflareUpdateRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// CloudflareClient defines the interface for Cloudflare API interactions
type CloudflareClient interface {
	GetDNSRecord(hostname string) (string, string, error)
	UpdateDNSRecord(recordID, hostname, ip string) error
}

// RealCloudflareClient implements CloudflareClient using the real Cloudflare API
type RealCloudflareClient struct {
	APIToken string
	ZoneID   string
}

var (
	config   Config
	cfClient CloudflareClient
)

func main() {
	// Load configuration from environment variables
	config = Config{
		Port:               getEnv("PORT", "8080"),
		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),
		BasicAuthUsername:  getEnv("BASIC_AUTH_USERNAME", ""),
		BasicAuthPassword:  getEnv("BASIC_AUTH_PASSWORD", ""),
	}

	// Initialize Cloudflare client
	cfClient = &RealCloudflareClient{
		APIToken: config.CloudflareAPIToken,
		ZoneID:   config.CloudflareZoneID,
	}

	// Validate required configuration
	if config.CloudflareAPIToken == "" || config.CloudflareZoneID == "" {
		log.Fatal("Missing required environment variables: CLOUDFLARE_API_TOKEN, CLOUDFLARE_ZONE_ID")
	}

	http.HandleFunc("/nic/update", handleDynDNSUpdate)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Starting DynDNS Cloudflare Proxy on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleDynDNSUpdate(w http.ResponseWriter, r *http.Request) {
	if config.BasicAuthUsername != "" && config.BasicAuthPassword != "" {
		if !checkBasicAuth(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="DynDNS"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "badauth")
			return
		}
	}

	hostname := r.URL.Query().Get("hostname")
	myip := r.URL.Query().Get("myip")

	if hostname == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "notfqdn")
		return
	}

	// If myip is not provided, use the client's IP
	if myip == "" {
		myip = getClientIP(r)
	}

	// Validate IP address
	if !isValidIP(myip) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "badip")
		return
	}

	log.Printf("Updating DNS record: hostname=%s, ip=%s", hostname, myip)

	// Get existing DNS record from Cloudflare
	recordID, currentIP, err := cfClient.GetDNSRecord(hostname)
	if err != nil {
		log.Printf("Error getting DNS record: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "911")
		return
	}

	// Check if IP has changed
	if currentIP == myip {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "nochg %s", myip)
		return
	}

	// Update DNS record on Cloudflare
	if err := cfClient.UpdateDNSRecord(recordID, hostname, myip); err != nil {
		log.Printf("Error updating DNS record: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "911")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "good %s", myip)
}

func checkBasicAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return false
	}

	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == config.BasicAuthUsername && parts[1] == config.BasicAuthPassword
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func isValidIP(ip string) bool {
	// Basic IP validation (supports both IPv4 and IPv6)
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// IPv4 validation
		for _, part := range parts {
			if part == "" {
				return false
			}
		}
		return true
	}
	// Simple IPv6 check (contains colons)
	return strings.Contains(ip, ":")
}

func (c *RealCloudflareClient) GetDNSRecord(hostname string) (string, string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", c.ZoneID, hostname)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var cfResp CloudflareListResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return "", "", err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return "", "", fmt.Errorf("cloudflare API error: %s", cfResp.Errors[0].Message)
		}
		return "", "", fmt.Errorf("cloudflare API returned error")
	}

	if len(cfResp.Result) == 0 {
		return "", "", fmt.Errorf("DNS record not found: %s", hostname)
	}

	record := cfResp.Result[0]
	return record.ID, record.Content, nil
}

func (c *RealCloudflareClient) UpdateDNSRecord(recordID, hostname, ip string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.ZoneID, recordID)

	updateReq := CloudflareUpdateRequest{
		Type:    "A",
		Name:    hostname,
		Content: ip,
		TTL:     1, // Automatic TTL
		Proxied: false,
	}

	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfResp CloudflareSingleResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("cloudflare API error: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare API returned error")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
