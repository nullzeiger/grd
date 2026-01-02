// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

type App struct {
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	Repo         string `json:"repo"`
	AssetPattern string `json:"asset_pattern"`
	VersionFlag  string `json:"version_flag"`
}
