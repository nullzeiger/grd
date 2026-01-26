// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nullzeiger/grd/internal/util"
)

func TestFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HOME", tmpDir)

	expected := filepath.Join(tmpDir, util.Filename)

	got := util.FilePath()

	if got != expected {
		t.Fatalf("FilePath() = %s; want %s", got, expected)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if !util.FileExists(filePath) {
		t.Fatalf("FileExists(%s) = false; want true", filePath)
	}

	nonExistent := filepath.Join(tmpDir, "does_not_exist.txt")
	if util.FileExists(nonExistent) {
		t.Fatalf("FileExists(%s) = true; want false", nonExistent)
	}
}
