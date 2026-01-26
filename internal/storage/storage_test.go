// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storage_test

import (
	"os"
	"testing"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/storage"
	"github.com/nullzeiger/grd/internal/util"
)

func setupTempStorage(t *testing.T) string {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	path := util.FilePath()

	if err := storage.Create(); err != nil {
		t.Fatalf("storage.Create() failed: %v", err)
	}

	return path
}

func TestCreate(t *testing.T) {
	path := setupTempStorage(t)

	if !util.FileExists(path) {
		t.Fatalf("File %s should exist after Create()", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != "[]" {
		t.Fatalf("File content = %s; want []", string(data))
	}
}

func TestWriteAndRead(t *testing.T) {
	setupTempStorage(t)

	apps := []app.App{
		{Name: "example", Owner: "user", Repo: "example-repo", AssetPattern: "Linux", VersionFlag: "--version"},
	}

	if err := storage.Write(apps); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	readApps, err := storage.Read()
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if len(readApps) != 1 {
		t.Fatalf("Read returned %d accounts; want 1", len(readApps))
	}

	if readApps[0].Name != "example" {
		t.Fatalf("App Name = %s; want example", readApps[0].Name)
	}
}

func TestAppend(t *testing.T) {
	setupTempStorage(t)

	app1 := app.App{Name: "example1", Owner: "user1", Repo: "example-repo1", AssetPattern: "Linux", VersionFlag: "--version"}
	app2 := app.App{Name: "example2", Owner: "user2", Repo: "example-repo2", AssetPattern: "Linux", VersionFlag: "--version"}

	if err := storage.Append(app1); err != nil {
		t.Fatalf("Append() failed: %v", err)
	}

	if err := storage.Append(app2); err != nil {
		t.Fatalf("Append() failed: %v", err)
	}

	apps, err := storage.Read()
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if len(apps) != 2 {
		t.Fatalf("Read returned %d apps; want 2", len(apps))
	}

	if apps[0].Name != "example1" || apps[1].Name != "example2" {
		t.Fatalf("Apps data mismatch: %v", apps)
	}
}

func TestAppendWithoutCreate(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	app := app.App{Name: "example", Owner: "user", Repo: "example-repo", AssetPattern: "Linux", VersionFlag: "--version"}

	err := storage.Append(app)
	if err == nil {
		t.Fatalf("Append() should fail if storage file does not exist")
	}
}
