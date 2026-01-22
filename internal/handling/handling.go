// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handling

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/client"
	"github.com/nullzeiger/grd/internal/compare"
	"github.com/nullzeiger/grd/internal/release"
	"github.com/nullzeiger/grd/internal/remote"
	"github.com/nullzeiger/grd/internal/storage"
	"github.com/nullzeiger/grd/internal/version"
)

type App = app.App

type SearchResult struct {
	Index int
	App   App
}

var outputMu sync.Mutex

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
	if newApp.Name == "" || newApp.Owner == "" || newApp.Repo == "" || newApp.AssetPattern == "" || newApp.VersionFlag == "" {
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

func concurrencyLevel() int {
	n := runtime.NumCPU()
	if n < 1 {
		return 1
	}
	return n
}

func runParallel(ctx context.Context, apps []App, task func(context.Context, App) error) error {
	sem := make(chan struct{}, concurrencyLevel())
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, a := range apps {
		app := a

		select {
		case <-ctx.Done():
			break
		default:
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break
		}

		wg.Go(func() {
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			if err := task(ctx, app); err != nil {
				select {
				case errChan <- err:
					cancel()
				default:
				}
			}
		})
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return err
	}

	return nil
}

func Check(ctx context.Context, download bool) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	return runParallel(ctx, apps, func(ctx context.Context, app App) error {
		local, err := version.Local(app.Name, app.VersionFlag)
		if err != nil {
			return fmt.Errorf("local version check failed for %s: %w", app.Name, err)
		}

		latest, err := version.Latest(ctx, rc, app.Owner, app.Repo)
		if err != nil {
			return fmt.Errorf("github check failed for %s: %w", app.Name, err)
		}

		comparator, err := compare.NewComparator()
		if err != nil {
			return err
		}

		result, err := comparator.CompareVersions(local, latest.TagName)
		if err != nil {
			return fmt.Errorf("comparison failed for %s: %w", app.Name, err)
		}

		outputMu.Lock()
		fmt.Printf("App: %s\nLocal: %s Remote: %s URL: %s\n",
			app.Name, local, latest.TagName, latest.URL)

		if result.IsLatest {
			fmt.Printf("Status: Up to date\n\n")
			outputMu.Unlock()
			return nil
		}

		fmt.Printf("Status: Update available!\n")
		if !download {
			fmt.Println()
			outputMu.Unlock()
			return nil
		}
		outputMu.Unlock()

		return downloadAssetSafe(ctx, rc, app)
	})
}

func Download(ctx context.Context) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	return runParallel(ctx, apps, func(ctx context.Context, app App) error {
		if err := downloadAssetSafe(ctx, rc, app); err != nil {
			outputMu.Lock()
			fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", app.Name, err)
			outputMu.Unlock()
			return err
		}
		return nil
	})
}

func Remote(ctx context.Context, remoteFile string) error {
	rc := client.New()

	apps, err := remote.Read(ctx, rc, remoteFile)
	if err != nil {
		return err
	}

	return runParallel(ctx, apps, func(ctx context.Context, app App) error {
		if err := downloadAssetSafe(ctx, rc, app); err != nil {
			outputMu.Lock()
			fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", app.Name, err)
			outputMu.Unlock()
			return err
		}
		return nil
	})
}

func downloadAssetSafe(ctx context.Context, rc *client.RequestClient, app App) error {
	outputMu.Lock()
	fmt.Printf("Downloading latest release for %s... ", app.Name)
	outputMu.Unlock()

	destPath, err := release.Latest(ctx, rc, app.Owner, app.Repo, app.AssetPattern)

	outputMu.Lock()
	defer outputMu.Unlock()

	if err != nil {
		fmt.Printf("Failed!\n")
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Printf("Success! Saved to: %s\n\n", destPath)
	return nil
}
