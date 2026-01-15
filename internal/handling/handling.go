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

func Check(ctx context.Context, download bool) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	sem := make(chan struct{}, concurrencyLevel())
	errChan := make(chan error, len(apps))
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, a := range apps {
		app := a
		wg.Go(func() {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			local, err := version.Local(app.Name, app.VersionFlag)
			if err != nil {
				errChan <- fmt.Errorf("local version check failed: %w", err)
				cancel()
				return
			}

			latest, err := version.Latest(ctx, rc, app.Owner, app.Repo)
			if err != nil {
				errChan <- fmt.Errorf("github check failed: %w", err)
				cancel()
				return
			}

			fmt.Printf("App: %s\nLocal: %s Remote: %s URL: %s\n",
				app.Name, local, latest.TagName, latest.URL)

			comparator, err := compare.NewComparator()
			if err != nil {
				errChan <- err
				cancel()
				return
			}

			result, err := comparator.CompareVersions(local, latest.TagName)
			if err != nil {
				errChan <- fmt.Errorf("comparison failed: %w", err)
				cancel()
				return
			}

			if result.IsLatest {
				fmt.Printf("Status: Up to date\n\n")
				return
			}

			fmt.Printf("Status: Update available!\n")
			if download {
				if err := downloadAsset(ctx, rc, app); err != nil {
					errChan <- err
					cancel()
					return
				}
			} else {
				fmt.Println()
			}
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func Download(ctx context.Context) error {
	rc := client.New()

	apps, err := storage.Read()
	if err != nil {
		return err
	}

	sem := make(chan struct{}, concurrencyLevel())
	errChan := make(chan error, len(apps))
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, a := range apps {
		app := a
		wg.Go(func() {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			if err := downloadAsset(ctx, rc, app); err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", app.Name, err)
				errChan <- err
				cancel()
				return
			}
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadAsset(ctx context.Context, rc *client.RequestClient, app App) error {
	fmt.Printf("Downloading latest release for %s... ", app.Name)
	destPath, err := release.Latest(ctx, rc, app.Owner, app.Repo, app.AssetPattern)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Printf("Success! Saved to: %s\n\n", destPath)
	return nil
}

func Remote(ctx context.Context, remoteFile string) error {
	rc := client.New()

	apps, err := remote.Read(ctx, rc, remoteFile)
	if err != nil {
		return err
	}

	sem := make(chan struct{}, concurrencyLevel())
	errChan := make(chan error, len(apps))
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, a := range apps {
		app := a
		wg.Go(func() {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			if err := downloadAsset(ctx, rc, app); err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", app.Name, err)
				errChan <- err
				cancel()
				return
			}
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
