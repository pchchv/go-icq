package foodgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// ODirService provides functionality for the ODir food group,
// which provides functionality for searching the user directory.
type ODirService struct {
	logger         *slog.Logger
	profileManager ProfileManager
}

// NewODirService creates a new instance of ODirService.
func NewODirService(logger *slog.Logger, profileManager ProfileManager) ODirService {
	return ODirService{
		logger:         logger,
		profileManager: profileManager,
	}
}

// KeywordListQuery returns a list of keywords that can be searched in the user directory.
func (s ODirService) KeywordListQuery(ctx context.Context, inFrame wire.SNACFrame) (wire.SNACMessage, error) {
	interests, err := s.profileManager.InterestList(ctx)
	if err != nil {
		return wire.SNACMessage{}, fmt.Errorf("InterestList: %w", err)
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ODir,
			SubGroup:  wire.ODirKeywordListReply,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x0F_0x04_KeywordListReply{
			Status:    0x01,
			Interests: interests,
		},
	}, nil
}

// newAIMNameAndAddrFromTLVList constructs an AIMNameAndAddr structure from the
// TLV list containing user directory fields like first name, last name, etc.
func newAIMNameAndAddrFromTLVList(tlvList wire.TLVList) state.AIMNameAndAddr {
	a := state.AIMNameAndAddr{}
	if firstName, hasFirstName := tlvList.String(wire.ODirTLVFirstName); hasFirstName {
		a.FirstName = firstName
	}

	if lastName, hasLastName := tlvList.String(wire.ODirTLVLastName); hasLastName {
		a.LastName = lastName
	}

	if middleName, hasMiddleName := tlvList.String(wire.ODirTLVMiddleName); hasMiddleName {
		a.MiddleName = middleName
	}

	if maidenName, hasMaidenName := tlvList.String(wire.ODirTLVMaidenName); hasMaidenName {
		a.MaidenName = maidenName
	}

	if country, hasCountry := tlvList.String(wire.ODirTLVCountry); hasCountry {
		a.Country = country
	}

	if st, hasState := tlvList.String(wire.ODirTLVState); hasState {
		a.State = st
	}

	if city, hasCity := tlvList.String(wire.ODirTLVCity); hasCity {
		a.City = city
	}

	if nickName, hasNickName := tlvList.String(wire.ODirTLVNickName); hasNickName {
		a.NickName = nickName
	}

	if zipCode, hasZIPCode := tlvList.String(wire.ODirTLVZIP); hasZIPCode {
		a.ZIPCode = zipCode
	}

	if address, hasAddress := tlvList.String(wire.ODirTLVAddress); hasAddress {
		a.Address = address
	}

	return a
}
