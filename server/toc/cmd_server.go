package toc

import (
	"fmt"
	"strings"

	"github.com/pchchv/go-icq/wire"
)

var (
	rateLimitExceededErr = "ERROR:903"
	cmdInternalSvcErr    = "ERROR:989:internal server error"
)

// userInfoToUpdateBuddy creates an UPDATE_BUDDY server reply from a User Info TLV.
func userInfoToUpdateBuddy(snac wire.TLVUserInfo) string {
	online, _ := snac.Uint32BE(wire.OServiceUserInfoSignonTOD)
	idle, _ := snac.Uint16BE(wire.OServiceUserInfoIdleTime)
	uc := [3]string{" ", "O", " "}
	if snac.IsAway() {
		uc[2] = "U"
	}

	class := strings.Join(uc[:], "")
	return fmt.Sprintf("UPDATE_BUDDY:%s:%s:%s:%d:%d:%s", snac.ScreenName, "T", fmt.Sprintf("%d", snac.WarningLevel/10), online, idle, class)
}
