// Copyright 2026 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package remote

import (
	"context"
	"fmt"

	"github.com/nullzeiger/grd/internal/app"
	"github.com/nullzeiger/grd/internal/client"
)

func Read(ctx context.Context, rc *client.RequestClient, file string) ([]app.App, error) {
	var apps []app.App
	if err := rc.GetJSON(ctx, file, &apps); err != nil {
		return nil, fmt.Errorf("failed to read remote file: %w", err)
	}

	return apps, nil
}
