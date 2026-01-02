// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/json"
	"os"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/util"
)

func Create() error {
	path := util.FilePath()

	if util.FileExists(path) {
		return nil
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte("[]"))
	return err
}

func Read() ([]app.App, error) {
	path := util.FilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var apps []app.App
	err = json.Unmarshal(data, &apps)
	return apps, err
}

func Write(apps []app.App) error {
	path := util.FilePath()

	jsonData, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, util.Perm)
}

func Append(app app.App) error {
	apps, err := Read()
	if err != nil {
		return err
	}

	apps = append(apps, app)

	return Write(apps)
}
