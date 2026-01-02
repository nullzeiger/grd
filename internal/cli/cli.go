// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	handling "github.com/nullzeiger/grd/internal/handling"
	"github.com/nullzeiger/grd/internal/storage"
)

func Run() {
	listFlag := flag.Bool("all", false, "List all app entries")
	addFlag := flag.Bool("add", false, "Add a new app entry")
	deleteFlag := flag.Int("delete", -1, "Delete an entry by index")
	searchFlag := flag.String("search", "", "Search entries by keyword")
	checkFlag := flag.Bool("check", false, "Check release applications")
	downloadFlag := flag.Bool("download", false, "Download latest version applications")

	name := flag.String("name", "", "Application name (required for -add)")
	owner := flag.String("owner", "", "GitHub repository owner (required for -add)")
	repo := flag.String("repo", "", "GitHub repository name (required for -add)")
	assetPattern := flag.String("asset", "", "Asset pattern to match in release files (required for -add)")
	versionFlag := flag.String("flag", "", "Local version flag (e.g., '--version') (required for -add)")

	flag.Parse()

	if err := storage.Create(); err != nil {
		fmt.Println("Error creating app file:", err)
		return
	}

	if *listFlag {
		entries, err := handling.All()
		if err != nil {
			fmt.Println("Error listing entries:", err)
			return
		}
		for _, e := range entries {
			fmt.Println(e)
		}
		return
	}

	if *addFlag {
		if *name == "" || *owner == "" || *repo == "" || *assetPattern == "" || *versionFlag == "" {
			fmt.Println("Missing required fields for -add: --name, --owner, --repo, --asset, --flag.")
			os.Exit(1)
		}

		newEntry := handling.App{
			Name:         *name,
			Owner:        *owner,
			Repo:         *repo,
			AssetPattern: *assetPattern,
			VersionFlag:  *versionFlag,
		}

		if err := handling.Create(newEntry); err != nil {
			fmt.Println("Error adding entry:", err)
			return
		}

		fmt.Println("Entry added successfully.")
		return
	}

	if *deleteFlag >= 0 {
		ok, err := handling.Delete(*deleteFlag)
		if err != nil {
			fmt.Println("Error deleting entry:", err)
			return
		}
		if ok {
			fmt.Printf("Entry [%d] deleted successfully.\n", *deleteFlag)
		}
		return
	}

	if *searchFlag != "" {
		matches, err := handling.Search(*searchFlag)
		if err != nil {
			fmt.Println("Error searching:", err)
			return
		}

		if len(matches) == 0 {
			fmt.Println("No results found.")
			return
		}

		for _, m := range matches {
			fmt.Printf(
				"[%d] Name: %s Repo: %s Owner: %s AssetPattern: %s VersionFlag: %s\n",
				m.Index, m.App.Name, m.App.Repo, m.App.Owner, m.App.AssetPattern, m.App.VersionFlag)
		}
		return
	}

	if *checkFlag {
		ctx := context.Background()

		err := handling.Check(ctx, *downloadFlag)
		if err != nil {
			fmt.Println("Error during check:", err)
		}
		return
	}

	if *downloadFlag {
		ctx := context.Background()

		err := handling.Download(ctx)
		if err != nil {
			fmt.Println("Error during download:", err)
		}
		return
	}

	flag.Usage()
}
