package command

import (
	"context"

	"github.com/n1rna/menv/internal/storage"
)

type storageKey struct{}

// WithStorage returns a new context with the storage instance
func WithStorage(ctx context.Context, store *storage.Storage) context.Context {
	return context.WithValue(ctx, storageKey{}, store)
}

// GetStorage retrieves the storage instance from the context
func GetStorage(ctx context.Context) *storage.Storage {
	if store, ok := ctx.Value(storageKey{}).(*storage.Storage); ok {
		return store
	}
	return nil
}
