package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expected      string
	}{
		{
			name:          "X-Forwarded-For header",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1, 10.0.0.2",
			expected:      "10.0.0.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "10.0.0.3",
			expected:   "10.0.0.3",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("getClientIP() = %v, want %v", ip, tt.expected)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		valid bool
	}{
		{"Valid IPv4", "192.168.1.1", true},
		{"Valid IPv4 with zeros", "10.0.0.1", true},
		{"Valid IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"Invalid - empty", "", false},
		{"Invalid - incomplete IPv4", "192.168.1", false},
		{"Invalid - too many parts", "192.168.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIP(tt.ip)
			if result != tt.valid {
				t.Errorf("isValidIP(%s) = %v, want %v", tt.ip, result, tt.valid)
			}
		})
	}
}

func TestCheckBasicAuth(t *testing.T) {
	// Set up config for tests
	config = Config{
		BasicAuthUsername: "testuser",
		BasicAuthPassword: "testpass",
	}

	tests := []struct {
		name       string
		authHeader string
		expected   bool
	}{
		{
			name:       "Valid credentials",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
			expected:   true,
		},
		{
			name:       "Invalid credentials",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:wrongpass")),
			expected:   false,
		},
		{
			name:       "Invalid username",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:testpass")),
			expected:   false,
		},
		{
			name:       "No auth header",
			authHeader: "",
			expected:   false,
		},
		{
			name:       "Invalid format",
			authHeader: "Bearer token123",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			result := checkBasicAuth(req)
			if result != tt.expected {
				t.Errorf("checkBasicAuth() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable set",
			key:          "TEST_ENV_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "Environment variable not set",
			key:          "TEST_MISSING_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
