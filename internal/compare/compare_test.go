// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compare

import (
	"testing"
)

func TestNewComparator(t *testing.T) {
	comparator, err := NewComparator()
	if err != nil {
		t.Fatalf("NewComparator() returned an unexpected error: %v", err)
	}
	if comparator == nil {
		t.Fatal("NewComparator() returned a nil comparator")
	}
	if comparator.re == nil {
		t.Fatal("NewComparator() returned a comparator with a nil regexp")
	}
}

func TestCompareVersions(t *testing.T) {
	comparator, err := NewComparator()
	if err != nil {
		t.Fatalf("Failed to create comparator: %v", err)
	}

	testCases := []struct {
		name             string
		release          string
		current          string
		expectedIsLatest bool
		expectError      bool
		errorMsg         string
	}{
		{
			name:             "equal versions",
			release:          "v1.2.3",
			current:          "1.2.3",
			expectedIsLatest: true,
			expectError:      false,
		},
		{
			name:             "different versions",
			release:          "1.2.4",
			current:          "1.2.3",
			expectedIsLatest: false,
			expectError:      false,
		},
		{
			name:             "equal versions with extra text",
			release:          "release version 1.2.3",
			current:          "current version is 1.2.3",
			expectedIsLatest: true,
			expectError:      false,
		},
		{
			name:        "invalid release version",
			release:     "invalid",
			current:     "1.2.3",
			expectError: true,
			errorMsg:    "invalid release version: no X.Y.Z pattern found",
		},
		{
			name:        "invalid current version",
			release:     "1.2.3",
			current:     "invalid",
			expectError: true,
			errorMsg:    "invalid current version: no X.Y.Z pattern found",
		},
		{
			name:             "versions with more components",
			release:          "1.2.3.4",
			current:          "1.2.3",
			expectedIsLatest: true,
			expectError:      false,
		},
		{
			name:             "prefixed versions",
			release:          "grd-1.2.3",
			current:          "1.2.3",
			expectedIsLatest: true,
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := comparator.CompareVersions(tc.release, tc.current)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error but got none")
				}
				if err.Error() != tc.errorMsg {
					t.Fatalf("Expected error message '%s' but got '%s'", tc.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Did not expect an error but got: %v", err)
			}

			if result.IsLatest != tc.expectedIsLatest {
				t.Errorf("Expected IsLatest to be %v, but got %v", tc.expectedIsLatest, result.IsLatest)
			}
		})
	}
}
