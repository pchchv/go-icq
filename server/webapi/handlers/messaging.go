package handlers

import (
	"context"

	"github.com/pchchv/go-icq/state"
)

// RelationshipFetcher defines methods for fetching user relationships
type RelationshipFetcher interface {
	Relationship(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) (state.Relationship, error)
}
