package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type MihomoClient struct {
	baseURL    *url.URL
	secret     string
	httpClient *http.Client
}

type ProxiesResponse struct {
	Proxies map[string]json.RawMessage `json:"proxies"`
}

type ConnectionsResponse struct {
	DownloadTotal int64             `json:"downloadTotal"`
	UploadTotal   int64             `json:"uploadTotal"`
	Connections   []json.RawMessage `json:"connections"`
}

type DelayResponse struct {
	Delay int `json:"delay"`
}

func NewMihomoClient(controller string, secret string, httpClient *http.Client) (*MihomoClient, error) {
	baseURL, err := normalizeControllerURL(controller)
	if err != nil {
		return nil, err
	}
	if !isLocalhost(baseURL.Hostname()) {
		return nil, fmt.Errorf("external controller must be localhost, got %q", baseURL.Host)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &MihomoClient{
		baseURL:    baseURL,
		secret:     secret,
		httpClient: httpClient,
	}, nil
}

func (c *MihomoClient) GetProxies(ctx context.Context) (*ProxiesResponse, error) {
	var out ProxiesResponse
	if err := c.doJSON(ctx, http.MethodGet, "/proxies", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *MihomoClient) SelectProxy(ctx context.Context, groupName string, proxyName string) error {
	if groupName == "" {
		return errors.New("group name is required")
	}
	if proxyName == "" {
		return errors.New("proxy name is required")
	}

	body := map[string]string{"name": proxyName}
	path := "/proxies/" + url.PathEscape(groupName)
	return c.doJSON(ctx, http.MethodPut, path, body, nil)
}

func (c *MihomoClient) TestProxyDelay(ctx context.Context, proxyName string, testURL string, timeoutMS int) (*DelayResponse, error) {
	if proxyName == "" {
		return nil, errors.New("proxy name is required")
	}
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	path := "/proxies/" + url.PathEscape(proxyName) + "/delay"
	endpoint, err := c.endpoint(path)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("url", testURL)
	query.Set("timeout", fmt.Sprint(timeoutMS))
	endpoint.RawQuery = query.Encode()

	var out DelayResponse
	if err := c.doJSONURL(ctx, http.MethodGet, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *MihomoClient) GetConnections(ctx context.Context) (*ConnectionsResponse, error) {
	var out ConnectionsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/connections", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *MihomoClient) CloseConnections(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodDelete, "/connections", nil, nil)
}

func (c *MihomoClient) doJSON(ctx context.Context, method string, path string, in any, out any) error {
	endpoint, err := c.endpoint(path)
	if err != nil {
		return err
	}
	return c.doJSONURL(ctx, method, endpoint, in, out)
}

func (c *MihomoClient) doJSONURL(ctx context.Context, method string, endpoint *url.URL, in any, out any) error {
	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mihomo %s %s: %w", method, endpoint.EscapedPath(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mihomo %s %s: status %d: %s", method, endpoint.EscapedPath(), resp.StatusCode, strings.TrimSpace(string(message)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *MihomoClient) endpoint(escapedPath string) (*url.URL, error) {
	rawPath := strings.TrimRight(c.baseURL.EscapedPath(), "/") + escapedPath
	decodedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		return nil, err
	}

	endpoint := *c.baseURL
	endpoint.Path = decodedPath
	endpoint.RawPath = rawPath
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return &endpoint, nil
}

func normalizeControllerURL(controller string) (*url.URL, error) {
	if controller == "" {
		return nil, errors.New("external controller address is required")
	}
	if !strings.Contains(controller, "://") {
		controller = "http://" + controller
	}

	baseURL, err := url.Parse(controller)
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme != "http" {
		return nil, fmt.Errorf("external controller must use http, got %q", baseURL.Scheme)
	}
	if baseURL.Host == "" {
		return nil, errors.New("external controller host is required")
	}
	return baseURL, nil
}

func isLocalhost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
