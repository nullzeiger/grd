// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package release

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nullzeiger/grd/internal/client"
)

type Release struct {
	Assets []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func Latest(
	ctx context.Context,
	rc *client.RequestClient,
	owner, repo, assetPattern string,
) (string, error) {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	url := client.GitHubLatestReleaseUrl(owner, repo)

	var releaseData Release
	if err := rc.GetJSON(ctx, url, &releaseData); err != nil {
		return "", fmt.Errorf("failed to get latest release metadata: %w", err)
	}

	var chosen Asset
	found := false
	for _, a := range releaseData.Assets {
		if strings.Contains(a.Name, assetPattern) {
			chosen = a
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("asset not found matching pattern %q in repo %s/%s", assetPattern, owner, repo)
	}

	stream, err := rc.DownloadStream(ctx, chosen.BrowserDownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to start download: %w", err)
	}
	defer stream.Close()

	destPath := filepath.Join(homeDir, chosen.Name)
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, stream); err != nil {
		return "", fmt.Errorf("failed to write data to file: %w", err)
	}

	return destPath, nil
}
