// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nullzeiger/grd/internal/client"
)

type Release struct {
	TagName string `json:"tag_name"`
	URL     string `json:"url"`
}

func Latest(ctx context.Context, rc *client.RequestClient, owner, repo string) (*Release, error) {
	url := client.GitHubLatestReleaseUrl(owner, repo)

	var releaseData Release
	if err := rc.GetJSON(ctx, url, &releaseData); err != nil {
		return nil, fmt.Errorf("fetching latest version for %s/%s: %w", owner, repo, err)
	}

	return &releaseData, nil
}

func Local(app, flag string) (string, error) {

	output, err := exec.Command(app, flag).Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s %s': %w", app, flag, err)
	}

	version := strings.TrimSpace(string(output))

	if version == "" {
		return "", fmt.Errorf("command '%s %s' returned empty output", app, flag)
	}

	return version, nil
}
