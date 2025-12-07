package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockCloudflareClient is a mock implementation of CloudflareClient
type MockCloudflareClient struct {
	Records      map[string]string // hostname -> ip
	RecordIDs    map[string]string // hostname -> id
	GetError     error
	UpdateError  error
	UpdateCalled bool
}

func (m *MockCloudflareClient) GetDNSRecord(hostname string) (string, string, error) {
	if m.GetError != nil {
		return "", "", m.GetError
	}
	if ip, ok := m.Records[hostname]; ok {
		return m.RecordIDs[hostname], ip, nil
	}
	return "", "", errors.New("record not found")
}

func (m *MockCloudflareClient) UpdateDNSRecord(recordID, hostname, ip string) error {
	m.UpdateCalled = true
	if m.UpdateError != nil {
		return m.UpdateError
	}
	m.Records[hostname] = ip
	return nil
}

func TestHandleDynDNSUpdate(t *testing.T) {
	// Save original config and restore after tests
	originalConfig := config
	defer func() { config = originalConfig }()

	// Reset config for these tests (disable auth)
	config = Config{
		BasicAuthUsername: "",
		BasicAuthPassword: "",
		CloudflareZoneID:  "test-zone",
	}

	tests := []struct {
		name           string
		queryParams    string
		mockClient     *MockCloudflareClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "Successful update",
			queryParams: "?hostname=test.example.com&myip=1.2.3.4",
			mockClient: &MockCloudflareClient{
				Records:   map[string]string{"test.example.com": "1.1.1.1"},
				RecordIDs: map[string]string{"test.example.com": "rec123"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "good 1.2.3.4",
		},
		{
			name:        "No change",
			queryParams: "?hostname=test.example.com&myip=1.1.1.1",
			mockClient: &MockCloudflareClient{
				Records:   map[string]string{"test.example.com": "1.1.1.1"},
				RecordIDs: map[string]string{"test.example.com": "rec123"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "nochg 1.1.1.1",
		},
		{
			name:        "Cloudflare API error on get",
			queryParams: "?hostname=test.example.com&myip=1.2.3.4",
			mockClient: &MockCloudflareClient{
				GetError: errors.New("api error"),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "911",
		},
		{
			name:        "Cloudflare API error on update",
			queryParams: "?hostname=test.example.com&myip=1.2.3.4",
			mockClient: &MockCloudflareClient{
				Records:     map[string]string{"test.example.com": "1.1.1.1"},
				RecordIDs:   map[string]string{"test.example.com": "rec123"},
				UpdateError: errors.New("update error"),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "911",
		},
		{
			name:           "Missing hostname",
			queryParams:    "?myip=1.2.3.4",
			mockClient:     &MockCloudflareClient{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "notfqdn",
		},
		{
			name:           "Invalid IP",
			queryParams:    "?hostname=test.example.com&myip=invalid",
			mockClient:     &MockCloudflareClient{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "badip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the global cfClient to our mock
			cfClient = tt.mockClient

			req := httptest.NewRequest("GET", "/nic/update"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handleDynDNSUpdate(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("status code = %v, want %v", resp.StatusCode, tt.expectedStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("body = %v, want to contain %v", body, tt.expectedBody)
			}
		})
	}
}
