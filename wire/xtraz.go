package wire

import "html"

const (
	XtrazFuncData       uint16 = 0x0002 // greeting cards, custom data
	XtrazFuncNotify     uint16 = 0x0008 // XStatus notifications
	XtrazFuncInvitation uint16 = 0x0001 // chat invitation
	XtrazFuncUserRemove uint16 = 0x0004 // user removal notification
)

// MangleXtrazXML encodes XML for Xtraz transport using HTML entities.
func MangleXtrazXML(plain string) string {
	return html.EscapeString(plain)
}

// UnmangleXtrazXML decodes the HTML entity encoded XML used in Xtraz messages.
// Xtraz uses HTML entity encoding for transport: &lt; &gt; &amp; &quot;
func UnmangleXtrazXML(mangled string) string {
	return html.UnescapeString(mangled)
}
