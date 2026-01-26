// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/nullzeiger/grd/internal/client"
)

func TestLatest(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		serverResponse Release
		serverStatus   int
		wantErr        bool
		errContains    string
	}{
		{
			name:  "successful fetch",
			owner: "testowner",
			repo:  "testrepo",
			serverResponse: Release{
				TagName: "v1.2.3",
				URL:     "https://api.github.com/repos/testowner/testrepo/releases/123",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "server returns 404",
			owner:        "testowner",
			repo:         "testrepo",
			serverStatus: http.StatusNotFound,
			wantErr:      true,
			errContains:  "http error 404",
		},
		{
			name:         "server returns 500",
			owner:        "testowner",
			repo:         "testrepo",
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
			errContains:  "http error 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverStatus != http.StatusOK {
					w.WriteHeader(tt.serverStatus)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			rc := client.New()
			ctx := context.Background()

			url := server.URL
			var release Release
			err := rc.GetJSON(ctx, url, &release)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Latest() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Latest() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Latest() unexpected error = %v", err)
				return
			}

			if release.TagName != tt.serverResponse.TagName {
				t.Errorf("Latest() TagName = %v, want %v", release.TagName, tt.serverResponse.TagName)
			}
			if release.URL != tt.serverResponse.URL {
				t.Errorf("Latest() URL = %v, want %v", release.URL, tt.serverResponse.URL)
			}
		})
	}
}

func TestLatest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	rc := client.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var release Release
	err := rc.GetJSON(ctx, server.URL, &release)
	if err == nil {
		t.Error("Latest() expected error due to context cancellation, got nil")
	}
}

func TestLocal(t *testing.T) {
	var testCmd, testFlag string
	if runtime.GOOS == "windows" {
		testCmd = "cmd"
		testFlag = "/C echo test-version"
	} else {
		testCmd = "echo"
		testFlag = "test-version"
	}

	tests := []struct {
		name        string
		app         string
		flag        string
		wantErr     bool
		errContains string
		checkOutput func(string) bool
	}{
		{
			name:    "successful command execution",
			app:     testCmd,
			flag:    testFlag,
			wantErr: false,
			checkOutput: func(output string) bool {
				return strings.Contains(output, "test-version")
			},
		},
		{
			name:        "nonexistent command",
			app:         "nonexistent-command-xyz",
			flag:        "--version",
			wantErr:     true,
			errContains: "failed to execute command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Local(tt.app, tt.flag)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Local() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Local() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Local() unexpected error = %v", err)
				return
			}

			if got == "" {
				t.Error("Local() returned empty string")
			}

			if tt.checkOutput != nil && !tt.checkOutput(got) {
				t.Errorf("Local() output = %v, failed output check", got)
			}
		})
	}
}

func TestLocal_EmptyOutput(t *testing.T) {
	var testCmd, testFlag string
	if runtime.GOOS == "windows" {
		testCmd = "cmd"
		testFlag = "/C echo."
	} else {
		testCmd = "true"
		testFlag = ""
	}

	if runtime.GOOS == "windows" {
		t.Skip("skipping empty output test on Windows")
	}

	got, err := Local(testCmd, testFlag)
	if err == nil {
		t.Error("Local() expected error for empty output, got nil")
	}
	if !strings.Contains(err.Error(), "returned empty output") {
		t.Errorf("Local() error = %v, want error containing 'returned empty output'", err)
	}
	if got != "" {
		t.Errorf("Local() = %v, want empty string", got)
	}
}

func TestLocal_WhitespaceHandling(t *testing.T) {
	var testCmd, testFlag string
	if runtime.GOOS == "windows" {
		testCmd = "cmd"
		testFlag = "/C echo   test-version  "
	} else {
		testCmd = "printf"
		testFlag = "  test-version  "

		if _, err := exec.LookPath("printf"); err != nil {
			t.Skip("printf command not available, skipping whitespace test")
		}
	}

	got, err := Local(testCmd, testFlag)
	if err != nil {
		t.Fatalf("Local() error = %v", err)
	}

	if strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") {
		t.Errorf("Local() = %q, expected whitespace to be trimmed", got)
	}
}
