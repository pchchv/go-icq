package wire

import (
	"html"
	"strconv"
)

const (
	XtrazFuncData       uint16 = 0x0002 // greeting cards, custom data
	XtrazFuncNotify     uint16 = 0x0008 // XStatus notifications
	XtrazFuncInvitation uint16 = 0x0001 // chat invitation
	XtrazFuncUserRemove uint16 = 0x0004 // user removal notification
)

// XtrazNotifyResponse represents a parsed Xtraz notification response (<NR> type).
type XtrazNotifyResponse struct {
	UIN     string
	Index   uint8
	Title   string
	Message string
}

// XtrazNotifyRequest represents a parsed Xtraz notification request (<N> type).
type XtrazNotifyRequest struct {
	PluginID  string
	ServiceID string
	RequestID string
	TransID   string // transaction ID
	SenderID  string // sender's UIN
}

// xmlNotifyResponseRoot is the internal XML structure for the <Root> element in responses.
type xmlNotifyResponseRoot struct {
	UIN   string `xml:"uin"`
	Index uint8  `xml:"index"`
	Title string `xml:"title"`
	Desc  string `xml:"desc"`
}

// MangleXtrazXML encodes XML for Xtraz transport using HTML entities.
func MangleXtrazXML(plain string) string {
	return html.EscapeString(plain)
}

// UnmangleXtrazXML decodes the HTML entity encoded XML used in Xtraz messages.
// Xtraz uses HTML entity encoding for transport: &lt; &gt; &amp; &quot;
func UnmangleXtrazXML(mangled string) string {
	return html.UnescapeString(mangled)
}

// BuildXtrazNotifyResponse builds an Xtraz notification response XML string.
func BuildXtrazNotifyResponse(uin string, index uint8, title, message string) string {
	xmlStr := `<NR><RES><ret event="OnRemoteNotification"><srv><id></id>` +
		`<val srv_id="cAwaySrv"><Root><CASXtraSetAwayMessage></CASXtraSetAwayMessage>` +
		`<uin>` + uin + `</uin>` +
		`<index>` + strconv.Itoa(int(index)) + `</index>` +
		`<title>` + MangleXtrazXML(title) + `</title>` +
		`<desc>` + MangleXtrazXML(message) + `</desc>` +
		`</Root></val></srv></ret></RES></NR>`
	return MangleXtrazXML(xmlStr)
}

// BuildXtrazNotifyRequest builds an Xtraz notification request XML string.
func BuildXtrazNotifyRequest(senderUIN string) string {
	xml := `<N><QUERY><PluginID>srvMng</PluginID></QUERY>` +
		`<NOTIFY><srv><id>cAwaySrv</id>` +
		`<req><id>AwayStat</id><trans>1</trans>` +
		`<senderId>` + senderUIN + `</senderId></req></srv></NOTIFY></N>`
	return MangleXtrazXML(xml)
}
