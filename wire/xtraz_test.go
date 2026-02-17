package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMangleXtrazXML(t *testing.T) {
	tests := []struct {
		name  string
		plain string
		want  string
	}{
		{
			name:  "mangle basic XML",
			plain: "<N><QUERY></QUERY></N>",
			want:  "&lt;N&gt;&lt;QUERY&gt;&lt;/QUERY&gt;&lt;/N&gt;",
		},
		{
			name:  "mangle special chars",
			plain: `<tag attr="value">text & more</tag>`,
			want:  "&lt;tag attr=&#34;value&#34;&gt;text &amp; more&lt;/tag&gt;",
		},
		{
			name:  "plain text unchanged",
			plain: "plain text",
			want:  "plain text",
		},
		{
			name:  "empty string",
			plain: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MangleXtrazXML(tt.plain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnmangleXtrazXML(t *testing.T) {
	tests := []struct {
		name    string
		mangled string
		want    string
	}{
		{
			name:    "unmangle basic entities",
			mangled: "&lt;N&gt;&lt;QUERY&gt;&lt;/QUERY&gt;&lt;/N&gt;",
			want:    "<N><QUERY></QUERY></N>",
		},
		{
			name:    "unmangle all entities",
			mangled: "&lt;tag attr=&quot;value&quot;&gt;text &amp; more&lt;/tag&gt;",
			want:    `<tag attr="value">text & more</tag>`,
		},
		{
			name:    "plain text unchanged",
			mangled: "plain text",
			want:    "plain text",
		},
		{
			name:    "empty string",
			mangled: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnmangleXtrazXML(tt.mangled)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestXtrazCapabilityGUID(t *testing.T) {
	// verify the GUID matches the expected GUID
	expected := "3b60b3ef-d82a-6c45-a4e0-9c5a5e67e865"
	assert.Equal(t, expected, CapXtrazScript.String())
}

func TestXStatusConstants(t *testing.T) {
	assert.Equal(t, uint8(1), XStatusAngry)
	assert.Equal(t, uint8(32), XStatusCoffee2)
}
func TestParseXtrazNotifyResponse(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    *XtrazNotifyResponse
		wantErr bool
	}{
		{
			name: "parse valid XStatus response",
			xml: `<NR><RES><ret event="OnRemoteNotification"><srv><id></id>` +
				`<val srv_id="cAwaySrv"><Root><CASXtraSetAwayMessage></CASXtraSetAwayMessage>` +
				`<uin>123456</uin><index>5</index><title>Having a beer</title>` +
				`<desc>Cheers!</desc></Root></val></srv></ret></RES></NR>`,
			want: &XtrazNotifyResponse{
				UIN:     "123456",
				Index:   5,
				Title:   "Having a beer",
				Message: "Cheers!",
			},
			wantErr: false,
		},
		{
			name:    "parse XML without Root element",
			xml:     "<NR><RES></RES></NR>",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "parse empty XML",
			xml:     "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseXtrazNotifyResponse([]byte(tt.xml))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildXtrazNotifyResponse(t *testing.T) {
	uin := "123456"
	index := uint8(5)
	title := "Having a beer"
	message := "Cheers!"
	result := BuildXtrazNotifyResponse(uin, index, title, message)

	// unmangle and verify the structure
	unmangled := UnmangleXtrazXML(result)
	assert.Contains(t, unmangled, "<NR>")
	assert.Contains(t, unmangled, "<uin>123456</uin>")
	assert.Contains(t, unmangled, "<index>5</index>")

	// verify it can be parsed back
	parsed, err := ParseXtrazNotifyResponse([]byte(unmangled))
	assert.NoError(t, err)
	assert.Equal(t, uin, parsed.UIN)
	assert.Equal(t, index, parsed.Index)
}
