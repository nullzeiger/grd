// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compare

import (
	"errors"
	"fmt"
	"regexp"
)

const versionPattern = `\d+\.\d+\.\d+`

type VersionComparator struct {
	re *regexp.Regexp
}

type CompareResult struct {
	IsLatest bool
}

func NewComparator() (*VersionComparator, error) {
	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return &VersionComparator{re: re}, nil
}

func (vc *VersionComparator) CompareVersions(release string, current string) (CompareResult, error) {
	releaseMatch := vc.re.FindString(release)
	if releaseMatch == "" {
		return CompareResult{}, errors.New("invalid release version: no X.Y.Z pattern found")
	}

	versionMatch := vc.re.FindString(current)
	if versionMatch == "" {
		return CompareResult{}, errors.New("invalid current version: no X.Y.Z pattern found")
	}

	isLatest := releaseMatch == versionMatch

	return CompareResult{IsLatest: isLatest}, nil
}
