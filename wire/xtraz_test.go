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
