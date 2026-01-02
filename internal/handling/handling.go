// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handling

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/client"
	"github.com/nullzeiger/grd/internal/compare"
	"github.com/nullzeiger/grd/internal/release"
	"github.com/nullzeiger/grd/internal/storage"
	"github.com/nullzeiger/grd/internal/version"
)

type App = app.App

type SearchResult struct {
	Index int
	App   App
}

func All() ([]string, error) {
	apps, err := storage.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read storage: %w", err)
	}

	entries := make([]string, 0, len(apps))
	for i, app := range apps {
		entries = append(entries,
			fmt.Sprintf("[%d] Name: %-15s Owner: %-10s Repo: %-15s Asset: %-10s Flag: %s",
				i, app.Name, app.Owner, app.Repo, app.AssetPattern, app.VersionFlag))
	}
	return entries, nil
}

func Create(newApp App) error {
	if newApp.Name == "" || newApp.Owner == "" || newApp.Repo == "" {
		return errors.New("invalid app data: missing required fields")
	}
	return storage.Append(newApp)
}

func Delete(index int) (bool, error) {
	apps, err := storage.Read()
	if err != nil {
		return false, err
	}

	if index < 0 || index >= len(apps) {
		return false, fmt.Errorf("index %d out of range", index)
	}

	copy(apps[index:], apps[index+1:])
	apps[len(apps)-1] = App{}
	apps = apps[:len(apps)-1]

	return true, storage.Write(apps)
}

func Search(key string) ([]SearchResult, error) {
	apps, err := storage.Read()
	if err != nil {
		return nil, err
	}

	key = strings.ToLower(key)
	results := make([]SearchResult, 0)

	for i, app := range apps {
		if strings.Contains(strings.ToLower(app.Name), key) ||
			strings.Contains(strings.ToLower(app.Owner), key) ||
			strings.Contains(strings.ToLower(app.Repo), key) {

			results = append(results, SearchResult{Index: i, App: app})
		}
	}

	return results, nil
}

func Check(ctx context.Context, download bool) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	var errList []error

	for _, app := range apps {
		err := func() error {
			local, err := version.Local(app.Name, app.VersionFlag)
			if err != nil {
				return fmt.Errorf("local version check failed: %w", err)
			}

			latest, err := version.Latest(ctx, rc, app.Owner, app.Repo)
			if err != nil {
				return fmt.Errorf("github check failed: %w", err)
			}

			fmt.Printf("App: %s\n  Local: %s\n  Remote: %s\n  URL: %s\n",
				app.Name, local, latest.TagName, latest.URL)

			comparator, err := compare.NewComparator()
			if err != nil {
				return err
			}

			result, err := comparator.CompareVersions(local, latest.TagName)
			if err != nil {
				return fmt.Errorf("comparison failed: %w", err)
			}

			if result.IsLatest {
				fmt.Printf("  Status: Up to date\n\n")
				return nil
			}

			fmt.Printf("  Status: Update available!\n")
			if download {
				if err := downloadAsset(ctx, rc, app); err != nil {
					return err
				}
			} else {
				fmt.Println()
			}
			return nil
		}()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n\n", app.Name, err)
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("encountered %d errors during check", len(errList))
	}

	return nil
}

func Download(ctx context.Context) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	var errList []error

	for _, app := range apps {
		if err := downloadAsset(ctx, rc, app); err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", app.Name, err)
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("encountered %d errors during download", len(errList))
	}
	return nil
}

func downloadAsset(ctx context.Context, rc *client.RequestClient, app App) error {
	fmt.Printf("Downloading latest release for %s...\n", app.Name)
	destPath, err := release.Latest(ctx, rc, app.Owner, app.Repo, app.AssetPattern)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Printf("Success! Saved to: %s\n\n", destPath)
	return nil
}
