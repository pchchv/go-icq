package state

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pchchv/go-icq/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFile string = "aim_test.db"

func TestSQLiteUserStore_AllRelationships(t *testing.T) {
	// buddyList represents the contents of a client-side or server-side buddy list
	type buddyList struct {
		// privacyMode is your current privacy mode.
		privacyMode wire.FeedbagPDMode
		// buddyList is the list of users on the buddy list. only active for wire.FeedbagPDModePermitAll and wire.FeedbagPDModePermitOnList
		buddyList []IdentScreenName
		// buddyList is the list of users on the permit list. only active when wire.FeedbagPDModePermitSome is set.
		permitList []IdentScreenName
		// buddyList is the list of users on the deny list. only active when wire.FeedbagPDModeDenySome is set.
		denyList []IdentScreenName
	}

	tests := []struct {
		name            string
		me              IdentScreenName
		clientSideLists map[IdentScreenName]buddyList
		serverSideLists map[IdentScreenName]buddyList
		tempBuddyList   map[IdentScreenName][]IdentScreenName
		expect          []Relationship
		filter          []IdentScreenName
	}{
		{
			name: "[me, client-side]: Allow all users to contact me [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, server-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, server-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, server-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, server-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow all users to contact me [them, server-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow all users to contact me [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow all users to contact me [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow all users to contact me [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow all users to contact me [them, server-side]: Allow only users on my Buddy List",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow all users to contact me [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow all users to contact me [them, server-side]: Allow only the users below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow all users to contact me [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow all users to contact me [them, server-side]: Block all users",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow all users to contact me [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow all users to contact me [them, server-side]: Block the users Below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, server-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, server-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, server-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, server-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only users on my Buddy List [them, server-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only users on my Buddy List [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only users on my Buddy List [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only users on my Buddy List [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only users on my Buddy List [them, server-side]: Allow only users on my Buddy List",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only users on my Buddy List [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only users on my Buddy List [them, server-side]: Allow only the users below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only users on my Buddy List [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only users on my Buddy List [them, server-side]: Block all users",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only users on my Buddy List [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only users on my Buddy List [them, server-side]: Block the users Below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, server-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, server-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, server-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, server-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Allow only the users below [them, server-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only the users below [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only the users below [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only the users below [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only the users below [them, server-side]: Allow only users on my Buddy List",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only the users below [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only the users below [them, server-side]: Allow only the users below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only the users below [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only the users below [them, server-side]: Block all users",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Allow only the users below [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Allow only the users below [them, server-side]: Block the users Below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{NewIdentScreenName("them")},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, server-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, server-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, server-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, server-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block all users [them, server-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block all users [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block all users [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block all users [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block all users [them, server-side]: Allow only users on my Buddy List",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block all users [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block all users [them, server-side]: Allow only the users below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block all users [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block all users [them, server-side]: Block all users",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block all users [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block all users [them, server-side]: Block the users Below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, server-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, server-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, server-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, server-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, client-side]: Block the users Below [them, server-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block the users Below [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block the users Below [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block the users Below [them, client-side]: Allow only users on my Buddy List",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block the users Below [them, server-side]: Allow only users on my Buddy List",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitOnList,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block the users Below [them, client-side]: Allow only the users below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block the users Below [them, server-side]: Allow only the users below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModePermitSome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{NewIdentScreenName("me")},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block the users Below [them, client-side]: Block all users",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block the users Below [them, server-side]: Block all users",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenyAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "[me, server-side]: Block the users Below [them, client-side]: Block the users Below",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name:            "[me, server-side]: Block the users Below [them, server-side]: Block the users Below",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("them")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("them")},
				},
				NewIdentScreenName("them"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("me")},
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them"),
					BlocksYou:     true,
					YouBlock:      true,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "(with filter) [me, client-side]: Allow all users to contact me [them, client-side]: Allow all users to contact me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList: []IdentScreenName{
						NewIdentScreenName("them-1"),
						NewIdentScreenName("them-2"),
					},
					permitList: []IdentScreenName{},
					denyList:   []IdentScreenName{},
				},
				NewIdentScreenName("them-1"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them-2"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them-3"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{},
			filter: []IdentScreenName{
				NewIdentScreenName("them-3"),
				NewIdentScreenName("them-1"),
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them-1"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: false,
					IsOnYourList:  true,
				},
				{
					User:          NewIdentScreenName("them-3"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  false,
				},
			},
		},
		{
			name:            "(filtered) [me, server-side]: Allow all users to contact me [them, server-side]: Allow all users to contact me",
			me:              NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList: []IdentScreenName{
						NewIdentScreenName("them-1"),
						NewIdentScreenName("them-2"),
					},
					permitList: []IdentScreenName{},
					denyList:   []IdentScreenName{},
				},
				NewIdentScreenName("them-1"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them-2"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("them-3"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			filter: []IdentScreenName{
				NewIdentScreenName("them-1"),
				NewIdentScreenName("them-3"),
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("them-1"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: false,
					IsOnYourList:  true,
				},
				{
					User:          NewIdentScreenName("them-3"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  false,
				},
			},
		},
		{
			name: "i have a temp buddy, they don't have me",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("friend1"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("friend1")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("friend2"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			tempBuddyList: map[IdentScreenName][]IdentScreenName{
				NewIdentScreenName("me"): {
					NewIdentScreenName("friend2"),
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("friend1"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
				{
					User:          NewIdentScreenName("friend2"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: false,
					IsOnYourList:  true,
				},
			},
		},
		{
			name: "someone has me as a temp buddy, I don't have them",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("friend1"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("friend1")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
				NewIdentScreenName("friend2"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			tempBuddyList: map[IdentScreenName][]IdentScreenName{
				NewIdentScreenName("friend2"): {
					NewIdentScreenName("me"),
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("friend1"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
				{
					User:          NewIdentScreenName("friend2"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  false,
				},
			},
		},
		{
			name: "I have a temp buddy that I've blocked",
			me:   NewIdentScreenName("me"),
			clientSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("friend1"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{NewIdentScreenName("me")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			serverSideLists: map[IdentScreenName]buddyList{
				NewIdentScreenName("me"): {
					privacyMode: wire.FeedbagPDModeDenySome,
					buddyList:   []IdentScreenName{NewIdentScreenName("friend1")},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{NewIdentScreenName("friend2")},
				},
				NewIdentScreenName("friend2"): {
					privacyMode: wire.FeedbagPDModePermitAll,
					buddyList:   []IdentScreenName{},
					permitList:  []IdentScreenName{},
					denyList:    []IdentScreenName{},
				},
			},
			tempBuddyList: map[IdentScreenName][]IdentScreenName{
				NewIdentScreenName("me"): {
					NewIdentScreenName("friend2"),
				},
			},
			expect: []Relationship{
				{
					User:          NewIdentScreenName("friend1"),
					BlocksYou:     false,
					YouBlock:      false,
					IsOnTheirList: true,
					IsOnYourList:  true,
				},
				{
					User:          NewIdentScreenName("friend2"),
					BlocksYou:     false,
					YouBlock:      true,
					IsOnTheirList: false,
					IsOnYourList:  true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				_ = os.Remove(testFile)
			}()

			feedbagStore, err := NewSQLiteUserStore(testFile)
			assert.NoError(t, err)

			for sn, list := range tt.clientSideLists {
				assert.NoError(t, feedbagStore.SetPDMode(context.Background(), sn, list.privacyMode))
				for _, buddy := range list.buddyList {
					assert.NoError(t, feedbagStore.AddBuddy(context.Background(), sn, buddy))
				}
				for _, buddy := range list.permitList {
					assert.NoError(t, feedbagStore.PermitBuddy(context.Background(), sn, buddy))
				}
				for _, buddy := range list.denyList {
					assert.NoError(t, feedbagStore.DenyBuddy(context.Background(), sn, buddy))
				}
			}

			for sn, list := range tt.serverSideLists {
				assert.NoError(t, feedbagStore.UseFeedbag(context.Background(), sn))
				itemID := uint16(1)
				items := []wire.FeedbagItem{
					pdInfoItem(itemID, list.privacyMode),
				}
				itemID++
				for _, buddy := range list.buddyList {
					items = append(items, newFeedbagItem(wire.FeedbagClassIdBuddy, itemID, buddy.String()))
					itemID++
				}
				for _, buddy := range list.permitList {
					items = append(items, newFeedbagItem(wire.FeedbagClassIDPermit, itemID, buddy.String()))
					itemID++
				}
				for _, buddy := range list.denyList {
					items = append(items, newFeedbagItem(wire.FeedbagClassIDDeny, itemID, buddy.String()))
					itemID++
				}
				assert.NoError(t, feedbagStore.FeedbagUpsert(context.Background(), sn, items))
			}

			for sn, list := range tt.tempBuddyList {
				for _, buddy := range list {
					assert.NoError(t, feedbagStore.AddBuddy(context.Background(), sn, buddy))
				}
			}

			have, err := feedbagStore.AllRelationships(context.Background(), tt.me, tt.filter)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expect, have)
		})
	}
}

func TestSQLiteUserStore_FeedbagUpsert(t *testing.T) {
	t.Run("buddy screen name is converted to ident screen name", func(t *testing.T) {
		defer func() {
			assert.NoError(t, os.Remove(testFile))
		}()

		f, err := NewSQLiteUserStore(testFile)
		assert.NoError(t, err)

		given := []wire.FeedbagItem{
			{
				GroupID:   0,
				ItemID:    1805,
				ClassID:   wire.FeedbagClassIdBuddy,
				Name:      "my Friend Bill",
				TLVLBlock: wire.TLVLBlock{},
			},
			{
				GroupID: 0x0A,
				ItemID:  0,
				ClassID: wire.FeedbagClassIdGroup,
				Name:    "Friends",
			},
		}
		expect := []wire.FeedbagItem{
			{
				GroupID:   0,
				ItemID:    1805,
				ClassID:   wire.FeedbagClassIdBuddy,
				Name:      "myfriendbill",
				TLVLBlock: wire.TLVLBlock{},
			},
			{
				GroupID: 0x0A,
				ItemID:  0,
				ClassID: wire.FeedbagClassIdGroup,
				Name:    "Friends",
			},
		}

		me := NewIdentScreenName("me")
		if err := f.FeedbagUpsert(context.Background(), me, given); err != nil {
			t.Fatalf("failed to upsert: %s", err.Error())
		}

		have, err := f.Feedbag(context.Background(), me)
		assert.NoError(t, err)
		assert.ElementsMatch(t, expect, have)
	})

	t.Run("upsert PD info with mode", func(t *testing.T) {
		defer func() {
			assert.NoError(t, os.Remove(testFile))
		}()

		f, err := NewSQLiteUserStore(testFile)
		assert.NoError(t, err)

		given := []wire.FeedbagItem{
			{
				GroupID: 0x0A,
				ItemID:  0,
				ClassID: wire.FeedbagClassIdPdinfo,
				TLVLBlock: wire.TLVLBlock{
					TLVList: wire.TLVList{
						wire.NewTLVBE(wire.FeedbagAttributesPdMode, wire.FeedbagPDModePermitSome),
					},
				},
			},
		}

		me := NewIdentScreenName("me")
		err = f.FeedbagUpsert(context.Background(), me, given)
		assert.NoError(t, err)

		var pdMode uint8
		err = f.db.QueryRow(`SELECT pdMode from feedbag WHERE screenName = ?`, me.String()).Scan(&pdMode)
		assert.NoError(t, err)
		assert.Equal(t, wire.FeedbagPDMode(pdMode), wire.FeedbagPDModePermitSome)
	})

	t.Run("upsert PD info without mode (QIP behavior)", func(t *testing.T) {
		defer func() {
			assert.NoError(t, os.Remove(testFile))
		}()

		f, err := NewSQLiteUserStore(testFile)
		assert.NoError(t, err)

		given := []wire.FeedbagItem{
			{
				GroupID:   0x0A,
				ItemID:    0,
				ClassID:   wire.FeedbagClassIdPdinfo,
				TLVLBlock: wire.TLVLBlock{},
			},
		}

		me := NewIdentScreenName("me")
		err = f.FeedbagUpsert(context.Background(), me, given)
		assert.NoError(t, err)

		var pdMode uint8
		err = f.db.QueryRow(`SELECT pdMode from feedbag WHERE screenName = ?`, me.String()).Scan(&pdMode)
		assert.NoError(t, err)
		assert.Equal(t, wire.FeedbagPDMode(pdMode), wire.FeedbagPDModePermitAll)
	})
}

func TestFeedbagDelete(t *testing.T) {
	screenName := NewIdentScreenName("sn2day")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	itemsIn := []wire.FeedbagItem{
		{
			GroupID: 0,
			ItemID:  1805,
			ClassID: 3,
			Name:    "spimmer1234",
			TLVLBlock: wire.TLVLBlock{
				TLVList: wire.TLVList{
					wire.NewTLVBE(0x01, uint16(1000)),
				},
			},
		},
		{
			GroupID: 0x0A,
			ItemID:  0,
			ClassID: 1,
			Name:    "Friends",
		},
		{
			GroupID: 0x0B,
			ItemID:  100,
			ClassID: 1,
			Name:    "co-workers",
		},
	}

	if err := f.FeedbagUpsert(context.Background(), screenName, itemsIn); err != nil {
		t.Fatalf("failed to upsert: %s", err.Error())
	}

	if err := f.FeedbagDelete(context.Background(), screenName, []wire.FeedbagItem{itemsIn[0]}); err != nil {
		t.Fatalf("failed to delete: %s", err.Error())
	}

	itemsOut, err := f.Feedbag(context.Background(), screenName)
	if err != nil {
		t.Fatalf("failed to retrieve: %s", err.Error())
	}

	expect := itemsIn[1:]
	if !reflect.DeepEqual(expect, itemsOut) {
		t.Fatalf("items did not match:\n in: %v\n out: %v", expect, itemsOut)
	}
}

func TestLastModifiedEmpty(t *testing.T) {
	screenName := NewIdentScreenName("sn2day")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	_, err = f.FeedbagLastModified(context.Background(), screenName)
	if err != nil {
		t.Fatalf("get error from last modified: %s", err.Error())
	}
}

func TestLastModifiedNotEmpty(t *testing.T) {

	screenName := NewIdentScreenName("sn2day")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	itemsIn := []wire.FeedbagItem{
		{
			GroupID: 0x0A,
			ItemID:  0,
			ClassID: 1,
			Name:    "Friends",
		},
	}
	if err := f.FeedbagUpsert(context.Background(), screenName, itemsIn); err != nil {
		t.Fatalf("failed to upsert: %s", err.Error())
	}

	_, err = f.FeedbagLastModified(context.Background(), screenName)
	if err != nil {
		t.Fatalf("get error from last modified: %s", err.Error())
	}
}

func TestProfile(t *testing.T) {
	screenName := NewIdentScreenName("sn2day")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	u := User{
		IdentScreenName: screenName,
	}
	if err := f.InsertUser(context.Background(), u); err != nil {
		t.Fatalf("failed to upsert new user: %s", err.Error())
	}

	profile, err := f.Profile(context.Background(), screenName)
	if err != nil {
		t.Fatalf("failed to retrieve profile: %s", err.Error())
	}

	if !profile.Empty() {
		t.Fatalf("expected empty profile for %s", screenName)
	}

	newProfile := UserProfile{
		ProfileText: "here is my profile",
		MIMEType:    "text/plain",
		UpdateTime:  time.Now().UTC().Truncate(time.Second),
	}
	if err := f.SetProfile(context.Background(), screenName, newProfile); err != nil {
		t.Fatalf("failed to create new profile: %s", err.Error())
	}

	profile, err = f.Profile(context.Background(), screenName)
	if err != nil {
		t.Fatalf("failed to retrieve profile: %s", err.Error())
	}

	if profile.ProfileText != newProfile.ProfileText {
		t.Fatalf("profiles did not match:\n expected: %v\n actual: %v", newProfile.ProfileText, profile.ProfileText)
	}
	if profile.MIMEType != newProfile.MIMEType {
		t.Fatalf("mime types did not match:\n expected: %v\n actual: %v", newProfile.MIMEType, profile.MIMEType)
	}
	if !profile.UpdateTime.Equal(newProfile.UpdateTime) {
		t.Fatalf("update times did not match:\n expected: %v\n actual: %v", newProfile.UpdateTime, profile.UpdateTime)
	}

	updatedProfile := UserProfile{
		ProfileText: "here is my profile [updated]",
		MIMEType:    "text/html",
		UpdateTime:  time.Now().UTC().Truncate(time.Second),
	}
	if err := f.SetProfile(context.Background(), screenName, updatedProfile); err != nil {
		t.Fatalf("failed to create new profile: %s", err.Error())
	}

	profile, err = f.Profile(context.Background(), screenName)
	if err != nil {
		t.Fatalf("failed to retrieve profile: %s", err.Error())
	}

	if profile.ProfileText != updatedProfile.ProfileText {
		t.Fatalf("updated profiles did not match:\n expected: %v\n actual: %v", updatedProfile.ProfileText, profile.ProfileText)
	}
	if profile.MIMEType != updatedProfile.MIMEType {
		t.Fatalf("updated mime types did not match:\n expected: %v\n actual: %v", updatedProfile.MIMEType, profile.MIMEType)
	}
	if !profile.UpdateTime.Equal(updatedProfile.UpdateTime) {
		t.Fatalf("updated update times did not match:\n expected: %v\n actual: %v", updatedProfile.UpdateTime, profile.UpdateTime)
	}
}

func TestProfileNonExistent(t *testing.T) {

	screenName := NewIdentScreenName("sn2day")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	prof, err := f.Profile(context.Background(), screenName)
	assert.NoError(t, err)
	assert.True(t, prof.Empty())
	assert.Equal(t, "", prof.MIMEType)
	assert.True(t, prof.UpdateTime.IsZero())
}

func TestProfile_MimeTypeAndUpdateTime(t *testing.T) {
	screenName := NewIdentScreenName("testuser")

	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	require.NoError(t, err)

	u := User{
		IdentScreenName: screenName,
	}
	require.NoError(t, f.InsertUser(context.Background(), u))

	t.Run("profile with empty mimeType and zero updateTime", func(t *testing.T) {
		profile := UserProfile{
			ProfileText: "test profile",
			MIMEType:    "",
			UpdateTime:  time.Time{},
		}
		require.NoError(t, f.SetProfile(context.Background(), screenName, profile))

		retrieved, err := f.Profile(context.Background(), screenName)
		require.NoError(t, err)
		assert.Equal(t, profile.ProfileText, retrieved.ProfileText)
		assert.Equal(t, "", retrieved.MIMEType)
		assert.True(t, retrieved.UpdateTime.IsZero())
	})

	t.Run("profile with mimeType and updateTime", func(t *testing.T) {
		updateTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		profile := UserProfile{
			ProfileText: "test profile with mime",
			MIMEType:    "text/html",
			UpdateTime:  updateTime,
		}
		require.NoError(t, f.SetProfile(context.Background(), screenName, profile))

		retrieved, err := f.Profile(context.Background(), screenName)
		require.NoError(t, err)
		assert.Equal(t, profile.ProfileText, retrieved.ProfileText)
		assert.Equal(t, "text/html", retrieved.MIMEType)
		assert.True(t, retrieved.UpdateTime.Equal(updateTime), "expected %v, got %v", updateTime, retrieved.UpdateTime)
	})

	t.Run("update profile with different mimeType and updateTime", func(t *testing.T) {
		updateTime := time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC)
		profile := UserProfile{
			ProfileText: "updated profile",
			MIMEType:    "text/plain",
			UpdateTime:  updateTime,
		}
		require.NoError(t, f.SetProfile(context.Background(), screenName, profile))

		retrieved, err := f.Profile(context.Background(), screenName)
		require.NoError(t, err)
		assert.Equal(t, profile.ProfileText, retrieved.ProfileText)
		assert.Equal(t, "text/plain", retrieved.MIMEType)
		assert.True(t, retrieved.UpdateTime.Equal(updateTime), "expected %v, got %v", updateTime, retrieved.UpdateTime)
	})
}

func TestGetUser(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	screenName := NewIdentScreenName("testscreenname")

	insertedUser := &User{
		IdentScreenName:   screenName,
		DisplayScreenName: DisplayScreenName("testscreenname"),
		AuthKey:           "theauthkey",
		StrongMD5Pass:     []byte("thepasshash"),
		RegStatus:         3,
		LastWarnUpdate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), // Database default value
	}
	err = f.InsertUser(context.Background(), *insertedUser)
	assert.NoError(t, err)

	actualUser, err := f.User(context.Background(), screenName)
	if err != nil {
		t.Fatalf("failed to get user: %s", err.Error())
	}

	if !reflect.DeepEqual(insertedUser, actualUser) {
		t.Fatalf("users are not equal. expect: %v actual: %v", insertedUser, actualUser)
	}
}

func TestSQLiteUserStore_User_OfflineMsgCount(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	screenName := NewIdentScreenName("testuser")

	// Insert user first
	err = f.InsertUser(context.Background(), User{
		IdentScreenName:   screenName,
		DisplayScreenName: DisplayScreenName("testuser"),
	})
	assert.NoError(t, err)

	// Set offlineMsgCount using the store method
	err = f.SetOfflineMsgCount(context.Background(), screenName, 5)
	assert.NoError(t, err)

	// Retrieve user and verify OfflineMsgCount is loaded
	user, err := f.User(context.Background(), screenName)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, 5, user.OfflineMsgCount)
}

func TestGetUserNotFound(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	actualUser, err := f.User(context.Background(), NewIdentScreenName("testscreenname"))
	if err != nil {
		t.Fatalf("failed to get user: %s", err.Error())
	}

	if actualUser != nil {
		t.Fatal("expected user to not be found")
	}
}

func TestSQLiteUserStore_Users(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	want := []User{
		{
			IdentScreenName:   NewIdentScreenName("userA"),
			DisplayScreenName: "userA",
		},
		{
			IdentScreenName:   NewIdentScreenName("userB"),
			DisplayScreenName: "userB",
		},
		{
			IdentScreenName:   NewIdentScreenName("userC"),
			DisplayScreenName: "userC",
			IsBot:             true,
		},
		{
			IdentScreenName:   NewIdentScreenName("100003"),
			DisplayScreenName: "100003",
			IsICQ:             true,
		},
	}

	for _, u := range want {
		err := f.InsertUser(context.Background(), u)
		assert.NoError(t, err)
	}

	have, err := f.AllUsers(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, want, have)
}

func TestSQLiteUserStore_InsertUser_UINButNotIsICQ(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	user := User{
		IdentScreenName:   NewIdentScreenName("100003"),
		DisplayScreenName: "100003",
	}

	err = f.InsertUser(context.Background(), user)
	assert.ErrorContains(t, err, "inserting user with UIN and isICQ=false")
}

func TestSQLiteUserStore_DeleteUser_DeleteExistentUser(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	err = f.InsertUser(context.Background(), User{
		IdentScreenName:   NewIdentScreenName("userA"),
		DisplayScreenName: "userA",
	})
	assert.NoError(t, err)
	err = f.InsertUser(context.Background(), User{
		IdentScreenName:   NewIdentScreenName("userB"),
		DisplayScreenName: "userB",
	})
	assert.NoError(t, err)

	err = f.DeleteUser(context.Background(), NewIdentScreenName("userA"))
	assert.NoError(t, err)

	have, err := f.AllUsers(context.Background())
	assert.NoError(t, err)

	want := []User{{
		IdentScreenName:   NewIdentScreenName("userB"),
		DisplayScreenName: "userB",
	}}
	assert.Equal(t, want, have)
}

func TestSQLiteUserStore_DeleteUser_DeleteNonExistentUser(t *testing.T) {
	defer func() {
		assert.NoError(t, os.Remove(testFile))
	}()

	f, err := NewSQLiteUserStore(testFile)
	assert.NoError(t, err)

	err = f.DeleteUser(context.Background(), NewIdentScreenName("userA"))
	assert.ErrorIs(t, ErrNoUser, err)
}

func TestNewStubUser(t *testing.T) {
	have, err := NewStubUser("userA")
	assert.NoError(t, err)

	want := User{
		IdentScreenName:   NewIdentScreenName("userA"),
		DisplayScreenName: "userA",
		AuthKey:           have.AuthKey,
	}
	assert.NoError(t, want.HashPassword("welcome1"))

	assert.Equal(t, want, have)
}

func newFeedbagItem(classID uint16, itemID uint16, name string) wire.FeedbagItem {
	return wire.FeedbagItem{
		ClassID: classID,
		ItemID:  itemID,
		Name:    name,
	}
}

func pdInfoItem(itemID uint16, pdMode wire.FeedbagPDMode) wire.FeedbagItem {
	return wire.FeedbagItem{
		ClassID: wire.FeedbagClassIdPdinfo,
		ItemID:  itemID,
		TLVLBlock: wire.TLVLBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.FeedbagAttributesPdMode, uint8(pdMode)),
			},
		},
	}
}
