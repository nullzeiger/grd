// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import "os"

const (
	Filename = ".apps.json"
	Perm     = 0o644
)

func FilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return home + "/" + Filename
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
