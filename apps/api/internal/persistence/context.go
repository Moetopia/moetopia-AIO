package persistence

import (
	"context"
	"moetopia.local/aio/api/internal/persistence/ent"
)

type clientKey struct{}

func WithClient(ctx context.Context, client *ent.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, client)
}
func Client(ctx context.Context, fallback *ent.Client) *ent.Client {
	if client, ok := ctx.Value(clientKey{}).(*ent.Client); ok {
		return client
	}
	return fallback
}
