package command

import (
	"context"

	"github.com/n1rna/ee-cli/internal/storage"
)

type storageKey struct{}

// WithStorage returns a new context with the UUID storage instance
func WithStorage(ctx context.Context, store *storage.UUIDStorage) context.Context {
	return context.WithValue(ctx, storageKey{}, store)
}

// GetStorage retrieves the UUID storage instance from the context
func GetStorage(ctx context.Context) *storage.UUIDStorage {
	if store, ok := ctx.Value(storageKey{}).(*storage.UUIDStorage); ok {
		return store
	}
	return nil
}
