// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package release

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nullzeiger/grd/internal/client"
)

func TestLatest(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		assetPattern   string
		releaseData    Release
		downloadData   string
		metadataStatus int
		downloadStatus int
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful download",
			owner:        "testowner",
			repo:         "testrepo",
			assetPattern: "linux",
			releaseData: Release{
				Assets: []Asset{
					{
						Name:               "app-linux-amd64.tar.gz",
						BrowserDownloadURL: "http://example.com/download",
					},
					{
						Name:               "app-darwin-amd64.tar.gz",
						BrowserDownloadURL: "http://example.com/other",
					},
				},
			},
			downloadData:   "test file content",
			metadataStatus: http.StatusOK,
			downloadStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:         "asset not found",
			owner:        "testowner",
			repo:         "testrepo",
			assetPattern: "windows",
			releaseData: Release{
				Assets: []Asset{
					{
						Name:               "app-linux-amd64.tar.gz",
						BrowserDownloadURL: "http://example.com/download",
					},
				},
			},
			metadataStatus: http.StatusOK,
			wantErr:        true,
			errContains:    "asset not found matching pattern",
		},
		{
			name:         "empty assets list",
			owner:        "testowner",
			repo:         "testrepo",
			assetPattern: "linux",
			releaseData: Release{
				Assets: []Asset{},
			},
			metadataStatus: http.StatusOK,
			wantErr:        true,
			errContains:    "asset not found matching pattern",
		},
		{
			name:           "metadata fetch error",
			owner:          "testowner",
			repo:           "testrepo",
			assetPattern:   "linux",
			metadataStatus: http.StatusNotFound,
			wantErr:        true,
			errContains:    "failed to get latest release metadata",
		},
		{
			name:         "download error",
			owner:        "testowner",
			repo:         "testrepo",
			assetPattern: "linux",
			releaseData: Release{
				Assets: []Asset{
					{
						Name:               "app-linux-amd64.tar.gz",
						BrowserDownloadURL: "http://example.com/download",
					},
				},
			},
			metadataStatus: http.StatusOK,
			downloadStatus: http.StatusInternalServerError,
			wantErr:        true,
			errContains:    "failed to start download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			originalHome := os.Getenv("HOME")
			if originalHome == "" {
				originalHome = os.Getenv("USERPROFILE")
			}
			defer func() {
				if originalHome != "" {
					if runtime := os.Getenv("GOOS"); runtime == "windows" {
						os.Setenv("USERPROFILE", originalHome)
					} else {
						os.Setenv("HOME", originalHome)
					}
				}
			}()

			os.Setenv("HOME", tempDir)
			os.Setenv("USERPROFILE", tempDir)

			downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.downloadStatus != http.StatusOK {
					w.WriteHeader(tt.downloadStatus)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.downloadData))
			}))
			defer downloadServer.Close()

			for i := range tt.releaseData.Assets {
				tt.releaseData.Assets[i].BrowserDownloadURL = downloadServer.URL + "/download"
			}

			metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.metadataStatus != http.StatusOK {
					w.WriteHeader(tt.metadataStatus)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.releaseData)
			}))
			defer metadataServer.Close()

			rc := client.New()
			ctx := context.Background()

			var releaseData Release
			err := rc.GetJSON(ctx, metadataServer.URL, &releaseData)

			if tt.metadataStatus != http.StatusOK {
				if err == nil {
					t.Errorf("Expected error for metadata fetch, got nil")
					return
				}
				if !strings.Contains(err.Error(), "http error") {
					t.Errorf("Error = %v, want error containing 'http error'", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error fetching metadata: %v", err)
			}

			var chosen Asset
			found := false
			for _, a := range releaseData.Assets {
				if strings.Contains(a.Name, tt.assetPattern) {
					chosen = a
					found = true
					break
				}
			}

			if !found {
				if !tt.wantErr {
					t.Errorf("Asset not found but expected success")
					return
				}
				if !strings.Contains("asset not found matching pattern", tt.errContains) {
					return
				}
				return
			}

			stream, err := rc.DownloadStream(ctx, chosen.BrowserDownloadURL)
			if tt.downloadStatus != http.StatusOK {
				if err == nil {
					t.Errorf("Expected download error, got nil")
					return
				}
				if !strings.Contains(err.Error(), "http") {
					t.Errorf("Error = %v, want error containing 'http'", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected download error: %v", err)
			}
			defer stream.Close()

			destPath := filepath.Join(tempDir, chosen.Name)
			outFile, err := os.Create(destPath)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, stream); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			if !tt.wantErr {
				content, err := os.ReadFile(destPath)
				if err != nil {
					t.Errorf("Failed to read downloaded file: %v", err)
				}
				if string(content) != tt.downloadData {
					t.Errorf("Downloaded content = %q, want %q", string(content), tt.downloadData)
				}
			}
		})
	}
}

func TestLatest_PatternMatching(t *testing.T) {
	tests := []struct {
		name      string
		assets    []Asset
		pattern   string
		wantAsset string
		wantFound bool
	}{
		{
			name: "exact substring match",
			assets: []Asset{
				{Name: "app-linux-amd64.tar.gz", BrowserDownloadURL: "url1"},
				{Name: "app-darwin-amd64.tar.gz", BrowserDownloadURL: "url2"},
			},
			pattern:   "linux",
			wantAsset: "app-linux-amd64.tar.gz",
			wantFound: true,
		},
		{
			name: "partial match",
			assets: []Asset{
				{Name: "myapp-v1.0.0-linux-x86.zip", BrowserDownloadURL: "url1"},
			},
			pattern:   "linux",
			wantAsset: "myapp-v1.0.0-linux-x86.zip",
			wantFound: true,
		},
		{
			name: "no match",
			assets: []Asset{
				{Name: "app-darwin-amd64.tar.gz", BrowserDownloadURL: "url1"},
				{Name: "app-freebsd-amd64.tar.gz", BrowserDownloadURL: "url2"},
			},
			pattern:   "windows",
			wantAsset: "",
			wantFound: false,
		},
		{
			name: "first match wins",
			assets: []Asset{
				{Name: "app-linux-arm64.tar.gz", BrowserDownloadURL: "url1"},
				{Name: "app-linux-amd64.tar.gz", BrowserDownloadURL: "url2"},
			},
			pattern:   "linux",
			wantAsset: "app-linux-arm64.tar.gz",
			wantFound: true,
		},
		{
			name:      "empty assets",
			assets:    []Asset{},
			pattern:   "linux",
			wantAsset: "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chosen Asset
			found := false
			for _, a := range tt.assets {
				if strings.Contains(a.Name, tt.pattern) {
					chosen = a
					found = true
					break
				}
			}

			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}

			if found && chosen.Name != tt.wantAsset {
				t.Errorf("chosen asset = %q, want %q", chosen.Name, tt.wantAsset)
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
		t.Error("Expected error due to context cancellation, got nil")
	}
}

func TestLatest_FileCreationError(t *testing.T) {
	tempDir := t.TempDir()

	testDir := filepath.Join(tempDir, "test")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if err := os.Chmod(testDir, 0444); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	defer os.Chmod(testDir, 0755)

	destPath := filepath.Join(testDir, "test-file.txt")
	_, err := os.Create(destPath)
	if err == nil {
		t.Error("Expected error creating file in read-only directory, got nil")
	}
}

func TestAsset_JSONSerialization(t *testing.T) {
	asset := Asset{
		Name:               "test-asset.tar.gz",
		BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/test-asset.tar.gz",
	}

	data, err := json.Marshal(asset)
	if err != nil {
		t.Fatalf("Failed to marshal asset: %v", err)
	}

	var decoded Asset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal asset: %v", err)
	}

	if decoded.Name != asset.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, asset.Name)
	}
	if decoded.BrowserDownloadURL != asset.BrowserDownloadURL {
		t.Errorf("BrowserDownloadURL = %q, want %q", decoded.BrowserDownloadURL, asset.BrowserDownloadURL)
	}
}

func TestRelease_JSONSerialization(t *testing.T) {
	release := Release{
		Assets: []Asset{
			{
				Name:               "asset1.tar.gz",
				BrowserDownloadURL: "https://example.com/asset1",
			},
			{
				Name:               "asset2.zip",
				BrowserDownloadURL: "https://example.com/asset2",
			},
		},
	}

	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("Failed to marshal release: %v", err)
	}

	var decoded Release
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal release: %v", err)
	}

	if len(decoded.Assets) != len(release.Assets) {
		t.Errorf("Assets length = %d, want %d", len(decoded.Assets), len(release.Assets))
	}

	for i, asset := range decoded.Assets {
		if asset.Name != release.Assets[i].Name {
			t.Errorf("Asset[%d].Name = %q, want %q", i, asset.Name, release.Assets[i].Name)
		}
		if asset.BrowserDownloadURL != release.Assets[i].BrowserDownloadURL {
			t.Errorf("Asset[%d].BrowserDownloadURL = %q, want %q", i, asset.BrowserDownloadURL, release.Assets[i].BrowserDownloadURL)
		}
	}
}
