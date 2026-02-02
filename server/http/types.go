package http

import (
	"context"

	"github.com/pchchv/go-icq/state"
)

// BARTAssetManager defines methods for managing BART (Buddy ART) assets.
type BARTAssetManager interface {
	// BARTItem retrieves a BART asset by its hash.
	BARTItem(ctx context.Context, hash []byte) ([]byte, error)
	// InsertBARTItem inserts a BART asset.
	InsertBARTItem(ctx context.Context, hash []byte, blob []byte, itemType uint16) error
	// ListBARTItems returns BART assets filtered by type.
	ListBARTItems(ctx context.Context, itemType uint16) ([]state.BARTItem, error)
	// DeleteBARTItem deletes a BART asset by hash.
	DeleteBARTItem(ctx context.Context, hash []byte) error
}
