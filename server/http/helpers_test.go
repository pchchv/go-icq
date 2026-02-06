package http

import (
	"net/mail"

	"github.com/pchchv/go-icq/state"
)

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

// EmailAddressParams is the list of parameters passed at
// the mock accountManager.EmailAddress call site
type EmailAddressParams []struct {
	screenName state.IdentScreenName
	result     *mail.Address
	err        error
}

// updateSuspendedStatus is the list of parameters passed at
// the mock accountManager.updateSuspendedStatus call site.
type updateSuspendedStatusParams []struct {
	suspendedStatus uint16
	screenName      state.IdentScreenName
	err             error
}

// setBotStatusParams is the list of parameters passed at
// the mock accountManager.SetBotStatus call site
type setBotStatusParams []struct {
	isBot      bool
	screenName state.IdentScreenName
	err        error
}
