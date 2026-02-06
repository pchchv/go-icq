package http

import "github.com/pchchv/go-icq/state"

// RegStatusParams is the list of parameters passed at
// the mock accountManager.RegStatus call site.
type RegStatusParams []struct {
	screenName state.IdentScreenName
	result     uint16
	err        error
}

// ConfirmStatusParams is the list of parameters passed at
// the mock accountManager.ConfirmStatus call site.
type ConfirmStatusParams []struct {
	screenName state.IdentScreenName
	result     bool
	err        error
}
