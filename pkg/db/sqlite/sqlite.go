// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package sqlite

import (
	"context"
	"fmt"

	internalsqlite "github.com/volantvm/volant/internal/server/db/sqlite"
	volantdb "github.com/volantvm/volant/pkg/db"
)

// Store re-exports the SQLite-backed persistence layer.
type Store = internalsqlite.Store

// Open establishes a SQLite-backed store using Volant's internal migrations and pragmas.
func Open(ctx context.Context, path string) (*Store, error) {
	return internalsqlite.Open(ctx, path)
}

// WithTx delegates to the underlying store helper for transactional operations.
func WithTx(ctx context.Context, store *Store, fn func(volantdb.Queries) error) error {
	if store == nil {
		return fmt.Errorf("sqlite: store is nil")
	}
	return store.WithTx(ctx, fn)
}
