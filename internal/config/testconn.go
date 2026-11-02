package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TestResult describes the outcome of a best-effort connection check.
type TestResult struct {
	OK       bool
	Message  string
	Models   int    // number of models advertised by an OpenAI-compatible /models
	Endpoint string // endpoint used
}

// TestConnection performs a best-effort check against the provider endpoint:
// it issues GET {baseURL}/models with the resolved credential. It never
// blocks longer than `timeout`.
func TestConnection(p *ProviderConfig, timeout time.Duration) TestResult {
	info := KeyInfoOf(p)
	res := TestResult{Endpoint: "unknown"}
	if p == nil {
		res.Message = "provider not found"
		return res
	}
	baseURL, _ := p.Settings["baseURL"].(string)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		res.Message = "no baseURL configured"
		return res
	}
	res.Endpoint = baseURL
	if info.Status != KeySet {
		res.Message = "no API key set for this provider"
		return res
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		res.Message = "invalid request: " + err.Error()
		return res
	}
	req.Header.Set("Authorization", "Bearer "+info.Resolved)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	httpRes, err := client.Do(req)
	if err != nil {
		res.Message = fmt.Sprintf("request failed: %s", friendlyErr(err))
		return res
	}
	defer httpRes.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(httpRes.Body, 1<<20))

	count := countModels(body)
	switch {
	case httpRes.StatusCode >= 200 && httpRes.StatusCode < 300:
		res.OK = true
		if count > 0 {
			res.Message = fmt.Sprintf("connected · %d model(s) listed", count)
		} else {
			res.Message = "connected · endpoint responded"
		}
		res.Models = count
	case httpRes.StatusCode == 401 || httpRes.StatusCode == 403:
		res.Message = fmt.Sprintf("authentication failed (%d) · check the API key", httpRes.StatusCode)
	default:
		res.Message = fmt.Sprintf("endpoint responded with HTTP %d", httpRes.StatusCode)
	}
	return res
}

func countModels(body []byte) int {
	var payload struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &payload) == nil {
		return len(payload.Data)
	}
	return 0
}

func friendlyErr(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "connection refused") {
		return "connection refused (is the endpoint reachable?)"
	}
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "lookup") {
		return "host could not be resolved"
	}
	if strings.Contains(msg, "timeout") {
		return "timed out"
	}
	if len(msg) > 120 {
		return msg[:120] + "…"
	}
	return msg
}
