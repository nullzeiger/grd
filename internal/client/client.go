// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultUserAgent = "grd/0.1.0"
	defaultTimeout   = 30 * time.Second
)

type RequestClient struct {
	client    *http.Client
	UserAgent string
}

func New() *RequestClient {
	return &RequestClient{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		UserAgent: defaultUserAgent,
	}
}

func GitHubLatestReleaseUrl(owner, repo string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
}

func (rc *RequestClient) GetJSON(ctx context.Context, url string, result any) error {
	req, err := rc.newRequest(ctx, http.MethodGet, url)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := rc.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http error %d: %s", resp.StatusCode, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}

	return nil
}

func (rc *RequestClient) DownloadStream(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := rc.newRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := rc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("http download error %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

func (rc *RequestClient) newRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", rc.UserAgent)

	return req, nil
}
