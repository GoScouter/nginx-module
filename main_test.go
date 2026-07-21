package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNginxModule_Scout(t *testing.T) {
	tests := []struct {
		name            string
		serverHeader    string
		bodyContent     string
		statusCode      int
		expectedIsNginx bool
		expectedVersion string
		expectedMethod  string
	}{
		{
			name:            "Nginx with explicit version header",
			serverHeader:    "nginx/1.24.0",
			expectedIsNginx: true,
			expectedVersion: "1.24.0",
			expectedMethod:  "Server Header",
		},
		{
			name:            "Nginx with version hidden in header",
			serverHeader:    "nginx",
			expectedIsNginx: true,
			expectedVersion: "Unknown (version hidden)",
			expectedMethod:  "Server Header",
		},
		{
			name:            "Server header suppressed, 404 page contains Nginx signature and version",
			serverHeader:    "",
			bodyContent:     "<html><head><title>404 Not Found</title></head><body><center><h1>404 Not Found</h1></center><hr><center>nginx/1.22.1</center></body></html>",
			statusCode:      http.StatusNotFound,
			expectedIsNginx: true,
			expectedVersion: "1.22.1",
			expectedMethod:  "404 Error Page HTML Signature",
		},
		{
			name:            "Non-Nginx server (Apache)",
			serverHeader:    "Apache/2.4.41 (Ubuntu)",
			expectedIsNginx: false,
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverHeader != "" {
					w.Header().Set("Server", tt.serverHeader)
				}
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				if tt.bodyContent != "" {
					w.Write([]byte(tt.bodyContent))
				}
			}))
			defer ts.Close()

			module := &NginxModule{}
			rawRes, err := module.Scout(ts.URL, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			res, ok := rawRes.(*NginxResult)
			if !ok {
				t.Fatalf("expected *NginxResult, got %T", rawRes)
			}

			if res.Err != nil {
				t.Fatalf("result error: %v", res.Err)
			}

			if res.IsNginx != tt.expectedIsNginx {
				t.Errorf("IsNginx = %v; want %v", res.IsNginx, tt.expectedIsNginx)
			}

			if res.NginxVersion != tt.expectedVersion {
				t.Errorf("NginxVersion = %q; want %q", res.NginxVersion, tt.expectedVersion)
			}

			if tt.expectedMethod != "" && res.DetectionMethod != tt.expectedMethod {
				t.Errorf("DetectionMethod = %q; want %q", res.DetectionMethod, tt.expectedMethod)
			}

			rendered := res.Render()
			if !strings.Contains(rendered, "\r\n") {
				t.Errorf("Render output should contain CRLF line endings")
			}
		})
	}
}
