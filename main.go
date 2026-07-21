package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/GoScouter/sdk"
)

var (
	nginxVersionRegex  = regexp.MustCompile(`(?i)nginx/([0-9]+\.[0-9]+\.[0-9]+)`)
	nginxBodySigRegex  = regexp.MustCompile(`(?i)<center>\s*nginx(?:/([0-9]+\.[0-9]+\.[0-9]+))?\s*</center>`)
)

type NginxModule struct{}

func (m *NginxModule) Name() string {
	return "nginx"
}

func (m *NginxModule) Description() string {
	return "Checks if a target server is using Nginx and detects its version using headers and advanced error-page fingerprinting"
}

func (m *NginxModule) Version() string {
	return "0.2.0"
}

func (m *NginxModule) Scout(target string, args []string) (sdk.Result, error) {
	result := &NginxResult{
		Target: target,
	}

	targetURL := target
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		result.Err = fmt.Errorf("invalid target URL: %w", err)
		return result, nil
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		result.Err = fmt.Errorf("failed to create request: %w", err)
		return result, nil
	}

	req.Header.Set("User-Agent", "GoScouter-NginxModule/0.2.0")

	resp, err := client.Do(req)
	if err != nil {
		result.Err = err
		return result, nil
	}

	result.StatusCode = resp.StatusCode
	result.ServerHeader = resp.Header.Get("Server")
	resp.Body.Close()

	serverLower := strings.ToLower(result.ServerHeader)
	if strings.Contains(serverLower, "nginx") {
		result.IsNginx = true
		result.DetectionMethod = "Server Header"
		matches := nginxVersionRegex.FindStringSubmatch(result.ServerHeader)
		if len(matches) > 1 {
			result.NginxVersion = matches[1]
		} else {
			result.NginxVersion = "Unknown (version hidden)"
		}
	}

	if !result.IsNginx || result.NginxVersion == "Unknown (version hidden)" {
		probeURL := *parsed
		probeURL.Path = "/_goscouter_non_existent_404_probe_" + fmt.Sprintf("%d", time.Now().UnixNano())

		probeReq, err := http.NewRequest(http.MethodGet, probeURL.String(), nil)
		if err == nil {
			probeReq.Header.Set("User-Agent", "GoScouter-NginxModule/0.2.0")
			probeResp, err := client.Do(probeReq)
			if err == nil {
				defer probeResp.Body.Close()

				probeServer := probeResp.Header.Get("Server")
				if probeServer != "" && result.ServerHeader == "" {
					result.ServerHeader = probeServer
				}

				probeServerLower := strings.ToLower(probeServer)
				if strings.Contains(probeServerLower, "nginx") {
					result.IsNginx = true
					if result.DetectionMethod == "" {
						result.DetectionMethod = "404 Probe Server Header"
					}
					matches := nginxVersionRegex.FindStringSubmatch(probeServer)
					if len(matches) > 1 && result.NginxVersion == "Unknown (version hidden)" {
						result.NginxVersion = matches[1]
					}
				}

				// Look for Nginx HTML signatures (<center>nginx/1.24.0</center>)
				bodyBytes, err := io.ReadAll(io.LimitReader(probeResp.Body, 4096))
				if err == nil && len(bodyBytes) > 0 {
					bodyStr := string(bodyBytes)
					matches := nginxBodySigRegex.FindStringSubmatch(bodyStr)
					if len(matches) > 0 {
						result.IsNginx = true
						if result.DetectionMethod == "" {
							result.DetectionMethod = "404 Error Page HTML Signature"
						}
						if len(matches) > 1 && matches[1] != "" {
							result.NginxVersion = matches[1]
						}
					}
				}
			}
		}
	}

	return result, nil
}

type NginxResult struct {
	Target          string
	IsNginx         bool
	NginxVersion    string
	ServerHeader    string
	DetectionMethod string
	StatusCode      int
	Err             error
}

func (r *NginxResult) Render() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Target:           %s\r\n", r.Target))

	if r.Err != nil {
		sb.WriteString(fmt.Sprintf("Status:           Error (%v)\r\n", r.Err))
		return sb.String()
	}

	if r.IsNginx {
		sb.WriteString("Nginx Detected:   Yes\r\n")
		if r.NginxVersion != "" {
			sb.WriteString(fmt.Sprintf("Nginx Version:    %s\r\n", r.NginxVersion))
		} else {
			sb.WriteString("Nginx Version:    Unknown\r\n")
		}
		if r.DetectionMethod != "" {
			sb.WriteString(fmt.Sprintf("Detection Method: %s\r\n", r.DetectionMethod))
		}
	} else {
		sb.WriteString("Nginx Detected:   No\r\n")
	}

	if r.ServerHeader != "" {
		sb.WriteString(fmt.Sprintf("Server Header:    %s\r\n", r.ServerHeader))
	} else {
		sb.WriteString("Server Header:    [None]\r\n")
	}

	sb.WriteString(fmt.Sprintf("HTTP Status:      %d\r\n", r.StatusCode))

	return sb.String()
}

func main() {
	if err := sdk.Serve(&NginxModule{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

