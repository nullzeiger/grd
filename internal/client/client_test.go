// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	rc := New()

	if rc == nil {
		t.Fatal("New() returned a nil RequestClient")
	}
	if rc.client == nil {
		t.Error("New() returned a RequestClient with a nil http.Client")
	}
	if rc.client.Timeout != defaultTimeout {
		t.Errorf("New() client timeout expected %v, got %v", defaultTimeout, rc.client.Timeout)
	}
	if rc.UserAgent != defaultUserAgent {
		t.Errorf("New() UserAgent expected %s, got %s", defaultUserAgent, rc.UserAgent)
	}
}

func TestGitHubLatestReleaseUrl(t *testing.T) {
	owner := "testowner"
	repo := "testrepo"
	expected := "https://api.github.com/repos/testowner/testrepo/releases/latest"
	actual := GitHubLatestReleaseUrl(owner, repo)

	if actual != expected {
		t.Errorf("GitHubLatestReleaseUrl(%s, %s) expected %s, got %s", owner, repo, expected, actual)
	}
}

func TestGetJSON(t *testing.T) {
	type testResult struct {
		Message string `json:"message"`
	}

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectError    bool
		expectedResult testResult
		errorContains  string
	}{
		{
			name:         "successful JSON response",
			statusCode:   http.StatusOK,
			responseBody: `{"message": "success"}`,
			expectError:  false,
			expectedResult: testResult{
				Message: "success",
			},
		},
		{
			name:          "HTTP error response",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"message": "not found"}`,
			expectError:   true,
			errorContains: "http error 404: 404 Not Found",
		},
		{
			name:          "invalid JSON response",
			statusCode:    http.StatusOK,
			responseBody:  `invalid json`,
			expectError:   true,
			errorContains: "json decode failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			rc := New()
			var result testResult
			err := rc.GetJSON(context.Background(), server.URL, &result)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error but got none")
				}
				if tc.errorContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tc.errorContains)) {
					t.Errorf("Error message '%v' did not contain expected substring '%s'", err, tc.errorContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error but got: %v", err)
				}
				if !reflect.DeepEqual(result, tc.expectedResult) {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, result)
				}
			}
		})
	}

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate a slow response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "slow but eventually successful"}`))
		}))
		defer server.Close()

		rc := New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		var result testResult
		err := rc.GetJSON(ctx, server.URL, &result)

		if err == nil {
			t.Fatal("Expected an error due to context cancellation, but got none")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("context deadline exceeded")) && !bytes.Contains([]byte(err.Error()), []byte("operation was canceled")) {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
	})
}

func TestDownloadStream(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "successful download",
			statusCode:   http.StatusOK,
			responseBody: "download content",
			expectError:  false,
		},
		{
			name:          "HTTP error response",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "server error",
			expectError:   true,
			errorContains: "http download error 500: 500 Internal Server Error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			rc := New()
			readCloser, err := rc.DownloadStream(context.Background(), server.URL)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error but got none")
				}
				if tc.errorContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tc.errorContains)) {
					t.Errorf("Error message '%v' did not contain expected substring '%s'", err, tc.errorContains)
				}
				if readCloser != nil {
					readCloser.Close() // Ensure the body is closed even on error if it was returned
					t.Error("Expected nil ReadCloser on error, but got one")
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error but got: %v", err)
				}
				if readCloser == nil {
					t.Fatal("Expected a ReadCloser, but got nil")
				}
				defer readCloser.Close()

				content, readErr := io.ReadAll(readCloser)
				if readErr != nil {
					t.Fatalf("Failed to read downloaded content: %v", readErr)
				}
				if string(content) != tc.responseBody {
					t.Errorf("Downloaded content expected '%s', got '%s'", tc.responseBody, string(content))
				}
			}
		})
	}

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate a slow response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`slow download`))
		}))
		defer server.Close()

		rc := New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		readCloser, err := rc.DownloadStream(ctx, server.URL)

		if err == nil {
			t.Fatal("Expected an error due to context cancellation, but got none")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("context deadline exceeded")) && !bytes.Contains([]byte(err.Error()), []byte("operation was canceled")) {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
		if readCloser != nil {
			readCloser.Close()
		}
	})
}

// Helper to mock http.Client.Do method for network errors
type mockFailingRoundTripper struct{}

func (mfrt *mockFailingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF // Simulate a network error
}

func TestGetJSON_NetworkError(t *testing.T) {
	rc := New()
	rc.client.Transport = &mockFailingRoundTripper{}

	var result interface{}
	err := rc.GetJSON(context.Background(), "http://example.com", &result)
	if err == nil {
		t.Fatal("Expected an error due to network failure, but got none")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("request failed")) {
		t.Errorf("Expected network error, got: %v", err)
	}
}

func TestDownloadStream_NetworkError(t *testing.T) {
	rc := New()
	rc.client.Transport = &mockFailingRoundTripper{}

	readCloser, err := rc.DownloadStream(context.Background(), "http://example.com")
	if err == nil {
		t.Fatal("Expected an error due to network failure, but got none")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("download request failed")) {
		t.Errorf("Expected network error, got: %v", err)
	}
	if readCloser != nil {
		readCloser.Close()
	}
}

func TestNewRequest(t *testing.T) {
	rc := New()
	ctx := context.Background()
	method := http.MethodGet
	url := "http://example.com"

	req, err := rc.newRequest(ctx, method, url)
	if err != nil {
		t.Fatalf("newRequest() returned an unexpected error: %v", err)
	}

	if req.Method != method {
		t.Errorf("Request method expected %s, got %s", method, req.Method)
	}
	if req.URL.String() != url {
		t.Errorf("Request URL expected %s, got %s", url, req.URL.String())
	}
	if req.Header.Get("User-Agent") != defaultUserAgent {
		t.Errorf("Request User-Agent expected %s, got %s", defaultUserAgent, rc.UserAgent)
	}
	if req.Context() != ctx {
		t.Error("Request context was not set correctly")
	}

}
