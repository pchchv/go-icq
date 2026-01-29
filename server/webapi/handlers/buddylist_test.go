package handlers

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/stretchr/testify/mock"
)

// MockWebAPISessionManager is a mock implementation of the WebAPISessionManager.
type MockWebAPISessionManager struct {
	mock.Mock
}

func (m *MockWebAPISessionManager) GetSession(ctx context.Context, aimsid string) (*state.WebAPISession, error) {
	args := m.Called(ctx, aimsid)
	if session := args.Get(0); session != nil {
		return session.(*state.WebAPISession), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWebAPISessionManager) TouchSession(ctx context.Context, aimsid string) error {
	args := m.Called(ctx, aimsid)
	return args.Error(0)
}

// MockFeedbagManager is a mock implementation of FeedbagManager.
type MockFeedbagManager struct {
	mock.Mock
}
