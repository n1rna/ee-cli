package command

import (
	"context"

	"github.com/n1rna/ee-cli/internal/entities"
)

type entityManagerKey struct{}

// WithEntityManager returns a new context with the entity manager instance
func WithEntityManager(ctx context.Context, manager *entities.Manager) context.Context {
	return context.WithValue(ctx, entityManagerKey{}, manager)
}

// GetEntityManager retrieves the entity manager instance from the context
func GetEntityManager(ctx context.Context) *entities.Manager {
	if manager, ok := ctx.Value(entityManagerKey{}).(*entities.Manager); ok {
		return manager
	}
	return nil
}
