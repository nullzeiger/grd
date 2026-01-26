// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/client"
)

func TestRead_Success(t *testing.T) {
	expectedApps := []app.App{
		{Name: "App1", Owner: "Owner1", Repo: "Repo1", AssetPattern: "tar.gz", VersionFlag: "-v"},
		{Name: "App2", Owner: "Owner2", Repo: "Repo2", AssetPattern: "zip", VersionFlag: "--version"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedApps); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	rc := client.New()

	apps, err := Read(ctx, rc, server.URL)

	if err != nil {
		t.Fatalf("Read() returned unexpected error: %v", err)
	}

	if len(apps) != len(expectedApps) {
		t.Errorf("Expected %d apps, got %d", len(expectedApps), len(apps))
	}

	if apps[0].Name != expectedApps[0].Name {
		t.Errorf("Expected first app name %s, got %s", expectedApps[0].Name, apps[0].Name)
	}
}

func TestRead_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ "name": "Incomplete JSON...`))
	}))
	defer server.Close()

	ctx := context.Background()
	rc := client.New()

	apps, err := Read(ctx, rc, server.URL)

	if err == nil {
		t.Fatal("Read() expected error for malformed JSON, got nil")
	}
	if apps != nil {
		t.Error("Read() expected nil apps slice on error, got data")
	}
}

func TestRead_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx := context.Background()
	rc := client.New()

	_, err := Read(ctx, rc, server.URL)

	if err == nil {
		t.Fatal("Read() expected error for HTTP 404, got nil")
	}
}
