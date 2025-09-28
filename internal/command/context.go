package command

import (
	"context"
	"fmt"

	"github.com/n1rna/ee-cli/internal/manager"
	"github.com/n1rna/ee-cli/internal/util"
)

type (
	entityManagerKey  struct{}
	commandContextKey struct{}
)

// WithEntityManager returns a new context with the entity manager instance
func WithEntityManager(ctx context.Context, manager *manager.Manager) context.Context {
	return context.WithValue(ctx, entityManagerKey{}, manager)
}

// GetEntityManager retrieves the entity manager instance from the context
func GetEntityManager(ctx context.Context) *manager.Manager {
	if manager, ok := ctx.Value(entityManagerKey{}).(*manager.Manager); ok {
		return manager
	}
	return nil
}

// WithCommandContext returns a new context with the command context instance
func WithCommandContext(ctx context.Context, cmdCtx *util.CommandContext) context.Context {
	return context.WithValue(ctx, commandContextKey{}, cmdCtx)
}

// GetCommandContext retrieves the command context instance from the context
func GetCommandContext(ctx context.Context) *util.CommandContext {
	if cmdCtx, ok := ctx.Value(commandContextKey{}).(*util.CommandContext); ok {
		return cmdCtx
	}
	return nil
}

// RequireCommandContext retrieves the command context and returns an error if not found
func RequireCommandContext(ctx context.Context) (*util.CommandContext, error) {
	cmdCtx := GetCommandContext(ctx)
	if cmdCtx == nil {
		return nil, fmt.Errorf("command context not initialized")
	}
	return cmdCtx, nil
}

// RequireProjectContext retrieves the command context and ensures we're in a project
func RequireProjectContext(ctx context.Context) (*util.CommandContext, error) {
	cmdCtx, err := RequireCommandContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cmdCtx.RequireProjectContext(); err != nil {
		return nil, err
	}

	return cmdCtx, nil
}
