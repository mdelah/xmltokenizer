package xmltokenizer_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/muktihari/xmltokenizer"
	"github.com/muktihari/xmltokenizer/internal/gpx"
	"github.com/muktihari/xmltokenizer/internal/xlsx"
	"github.com/muktihari/xmltokenizer/internal/xlsx/schema"
)

var tokenHeader = xmltokenizer.Token{
	Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	SelfClosing: true,
	Begin:       xmltokenizer.Pos{1, 1, 0},
	End:         xmltokenizer.Pos{1, 39, 38},
}

func TestTokenWithInmemXML(t *testing.T) {
	tt := []struct {
		name      string
		xml       string
		expecteds []xmltokenizer.Token
		err       error
	}{
		{
			name: "dtd without entity",
			xml: `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
	"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<body xmlns:foo="ns1" xmlns="ns2" xmlns:tag="ns3" ` +
				"\r\n\t" + `  >
	<hello lang="en">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔</hello>
	<query>&何; &is-it;</query>
	<goodbye />
	<outer foo:attr="value" xmlns:tag="ns4">
	<inner/>
	</outer>
	<tag:name>
	<![CDATA[Some text here.]]>
	</tag:name>
</body><!-- missing final newline -->`, // Note: retrieved from stdlib xml test.
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{2, 1, 1},
					End:         xmltokenizer.Pos{2, 39, 39},
				},
				{
					Data: []byte("<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"\n" +
						"	\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">"),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{3, 1, 40},
					End:         xmltokenizer.Pos{4, 60, 162},
				},
				{
					Name: xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("foo"), Full: []byte("xmlns:foo")}, Value: []byte("ns1")},
						{Name: xmltokenizer.Name{Local: []byte("xmlns"), Full: []byte("xmlns")}, Value: []byte("ns2")},
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("tag"), Full: []byte("xmlns:tag")}, Value: []byte("ns3")},
					},
					Begin: xmltokenizer.Pos{5, 1, 163},
					End:   xmltokenizer.Pos{6, 5, 219},
				},
				{
					Name: xmltokenizer.Name{Local: []byte("hello"), Full: []byte("hello")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Local: []byte("lang"), Full: []byte("lang")}, Value: []byte("en")},
					},
					Data:  []byte("World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔"),
					Begin: xmltokenizer.Pos{7, 2, 221},
					End:   xmltokenizer.Pos{7, 63, 284},
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("hello"), Full: []byte("hello")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{Line: 7, Column: 63, Offset: 284},
					End:          xmltokenizer.Pos{Line: 7, Column: 71, Offset: 292},
				},
				{
					Name:  xmltokenizer.Name{Local: []byte("query"), Full: []byte("query")},
					Data:  []byte("&何; &is-it;"),
					Begin: xmltokenizer.Pos{8, 2, 294},
					End:   xmltokenizer.Pos{8, 20, 314},
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("query"), Full: []byte("query")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{Line: 8, Column: 20, Offset: 314},
					End:          xmltokenizer.Pos{Line: 8, Column: 28, Offset: 322},
				},
				{
					Name:        xmltokenizer.Name{Local: []byte("goodbye"), Full: []byte("goodbye")},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{9, 2, 324},
					End:         xmltokenizer.Pos{9, 13, 335},
				},
				{
					Name: xmltokenizer.Name{Local: []byte("outer"), Full: []byte("outer")},
					Attrs: []xmltokenizer.Attr{
						{Name: xmltokenizer.Name{Prefix: []byte("foo"), Local: []byte("attr"), Full: []byte("foo:attr")}, Value: []byte("value")},
						{Name: xmltokenizer.Name{Prefix: []byte("xmlns"), Local: []byte("tag"), Full: []byte("xmlns:tag")}, Value: []byte("ns4")},
					},
					Begin: xmltokenizer.Pos{10, 2, 337},
					End:   xmltokenizer.Pos{10, 42, 377},
				},
				{
					Name:        xmltokenizer.Name{Local: []byte("inner"), Full: []byte("inner")},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{11, 2, 379},
					End:         xmltokenizer.Pos{11, 10, 387},
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("outer"), Full: []byte("outer")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{12, 2, 389},
					End:          xmltokenizer.Pos{12, 10, 397},
				},
				{
					Name:  xmltokenizer.Name{Prefix: []byte("tag"), Local: []byte("name"), Full: []byte("tag:name")},
					Data:  []byte("Some text here."),
					Begin: xmltokenizer.Pos{13, 2, 399},
					End:   xmltokenizer.Pos{14, 29, 438},
				},
				{
					Name:         xmltokenizer.Name{Prefix: []byte("tag"), Local: []byte("name"), Full: []byte("tag:name")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{15, 2, 440},
					End:          xmltokenizer.Pos{15, 13, 451},
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{16, 1, 452},
					End:          xmltokenizer.Pos{16, 8, 459},
				},
				{
					Data:        []byte("<!-- missing final newline -->"),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{16, 8, 459},
					End:         xmltokenizer.Pos{16, 38, 489},
				},
			},
		},
		{
			name: "unexpected EOF truncated XML after `<!`",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><!",
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 39, 38},
				},
			},
			err: io.ErrUnexpectedEOF,
		},
		{
			name: "missing attr name",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><a =\"ns2\"></a>",
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 39, 38},
				},
				{
					Name:  xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")},
					Attrs: []xmltokenizer.Attr{{xmltokenizer.Name{Local: []byte{}, Full: []byte{}}, []byte("ns2")}},
					Begin: xmltokenizer.Pos{1, 39, 38},
					End:   xmltokenizer.Pos{1, 49, 48},
				},
				{
					Name:         xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")},
					IsEndElement: true,
					Begin:        xmltokenizer.Pos{1, 49, 48},
					End:          xmltokenizer.Pos{1, 53, 52},
				},
			},
		},
		{
			name: "unexpected equals in attr name",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Image URL=\"https://test.com/my-url-ending-in-=\" URL2=\"https://ok.com\"/>",
			expecteds: []xmltokenizer.Token{
				{
					Data:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
					SelfClosing:  true,
					IsEndElement: false,
					Begin:        xmltokenizer.Pos{1, 1, 0},
					End:          xmltokenizer.Pos{1, 39, 38},
				},
				{Name: xmltokenizer.Name{Local: []byte("Image"), Full: []byte("Image")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("URL"), Full: []uint8("URL")},
							Value: []uint8("https://test.com/my-url-ending-in-="),
						},
						{
							Name:  xmltokenizer.Name{Local: []uint8("URL2"), Full: []uint8("URL2")},
							Value: []uint8("https://ok.com"),
						},
					},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 39, 38},
					End:         xmltokenizer.Pos{1, 111, 110},
				},
			},
		},
		{
			name: "tab after node name",
			xml:  `<sample	foo="bar"/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
					},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 20, 19},
				},
			},
		},
		{
			name: "tab after attribute value",
			xml:  `<sample foo="bar"	/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
					},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 21, 20},
				},
			},
		},
		{
			name: "tab between attributes",
			xml:  `<sample foo="bar"	baz="quux"/>`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{
						Local: []uint8("sample"),
						Full:  []uint8("sample"),
					},
					Attrs: []xmltokenizer.Attr{
						{
							Name: xmltokenizer.Name{
								Local: []uint8("foo"),
								Full:  []uint8("foo")},
							Value: []uint8("bar"),
						},
						{
							Name: xmltokenizer.Name{
								Local: []uint8("baz"),
								Full:  []uint8("baz")},
							Value: []uint8("quux"),
						},
					},
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 31, 30},
				},
			},
		},
		{
			name: "slash inside attribute value",
			xml:  `<sample path="foo/bar/baz">`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{Local: []byte("sample"), Full: []byte("sample")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("path"), Full: []uint8("path")},
							Value: []uint8("foo/bar/baz"),
						},
					},
					Begin: xmltokenizer.Pos{1, 1, 0},
					End:   xmltokenizer.Pos{1, 28, 27},
				},
			},
		},
		{
			name: "right angle bracket inside attribute value",
			xml:  `<sample path="foo>bar>baz">`,
			expecteds: []xmltokenizer.Token{
				{
					Name: xmltokenizer.Name{Local: []byte("sample"), Full: []byte("sample")},
					Attrs: []xmltokenizer.Attr{
						{
							Name:  xmltokenizer.Name{Local: []uint8("path"), Full: []uint8("path")},
							Value: []uint8("foo>bar>baz"),
						},
					},
					Begin: xmltokenizer.Pos{1, 1, 0},
					End:   xmltokenizer.Pos{1, 28, 27},
				},
			},
		},
		{
			name: "right angle bracket inside comment",
			xml:  `<!-->--><!-- foo>bar>baz -->`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<!-->-->`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 9, 8},
				},
				{
					Data:        []byte(`<!-- foo>bar>baz -->`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 9, 8},
					End:         xmltokenizer.Pos{1, 29, 28},
				},
			},
		},
		{
			name: "left angle bracket inside comment",
			xml:  `<!--<--><!-- foo<bar<baz -->`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []byte(`<!--<-->`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 9, 8},
				},
				{
					Data:        []byte(`<!-- foo<bar<baz -->`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 9, 8},
					End:         xmltokenizer.Pos{1, 29, 28},
				},
			},
		},
		{
			name: "angle brackets in processing instruction",
			xml:  `<?sample <foo> ?>`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8("<?sample <foo> ?>"),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 18, 17},
				},
			},
		},
		{
			name: "quote in processing instruction",
			xml:  `<?sample " ?>`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<?sample " ?>`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 14, 13},
				},
			},
		},
		{
			name: "quoted right angle bracket in doctype",
			xml:  `<!DOCTYPE ">" >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE ">" >`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 16, 15},
				},
			},
		},
		{
			name: "commented angle brackets in doctype",
			xml:  `<!DOCTYPE [ <!-- <foo> --> ] ><!DOCTYPE [ <!-- > --> ] ><!DOCTYPE [ <!-- < --> ] >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- <foo> --> ] >`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 31, 30},
				},
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- > --> ] >`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 31, 30},
					End:         xmltokenizer.Pos{1, 57, 56},
				},
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- < --> ] >`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 57, 56},
					End:         xmltokenizer.Pos{1, 83, 82},
				},
			},
		},
		{
			name: "commented quote in doctype",
			xml:  `<!DOCTYPE [ <!-- " --> ] >`,
			expecteds: []xmltokenizer.Token{
				{
					Data:        []uint8(`<!DOCTYPE [ <!-- " --> ] >`),
					SelfClosing: true,
					Begin:       xmltokenizer.Pos{1, 1, 0},
					End:         xmltokenizer.Pos{1, 27, 26},
				},
			},
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d]: %s", i, tc.name), func(t *testing.T) {
			checkTokens(t, bytes.NewReader([]byte(tc.xml)), tc.expecteds, tc.err)
		})
	}
}

func TestTokenWithSmallXMLFiles(t *testing.T) {
	tt := []struct {
		filename  string
		expecteds []xmltokenizer.Token
		err       error
	}{
		{filename: "cdata.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Name:  xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				Begin: xmltokenizer.Pos{Line: 2, Column: 1, Offset: 39},
				End:   xmltokenizer.Pos{Line: 2, Column: 10, Offset: 48},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("text"),
				Begin: xmltokenizer.Pos{Line: 3, Column: 3, Offset: 51},
				End:   xmltokenizer.Pos{Line: 4, Column: 23, Offset: 80},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 5, Column: 3, Offset: 83},
				End:          xmltokenizer.Pos{Line: 5, Column: 10, Offset: 90},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("<element>text</element>"),
				Begin: xmltokenizer.Pos{Line: 6, Column: 3, Offset: 93},
				End:   xmltokenizer.Pos{Line: 7, Column: 40, Offset: 139},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 8, Column: 3, Offset: 142},
				End:          xmltokenizer.Pos{Line: 8, Column: 10, Offset: 149},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("<element>text</element>"),
				Begin: xmltokenizer.Pos{Line: 9, Column: 3, Offset: 152},
				End:   xmltokenizer.Pos{Line: 12, Column: 8, Offset: 210},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 13, Column: 3, Offset: 213},
				End:          xmltokenizer.Pos{Line: 13, Column: 10, Offset: 220},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 14, Column: 1, Offset: 221},
				End:          xmltokenizer.Pos{Line: 14, Column: 11, Offset: 231},
			},
		}},
		{filename: "cdata_clrf.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Name:  xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				Begin: xmltokenizer.Pos{Line: 2, Column: 1, Offset: 39},
				End:   xmltokenizer.Pos{Line: 2, Column: 10, Offset: 48},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("text"),
				Begin: xmltokenizer.Pos{Line: 3, Column: 3, Offset: 51},
				End:   xmltokenizer.Pos{Line: 4, Column: 23, Offset: 80},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 5, Column: 3, Offset: 83},
				End:          xmltokenizer.Pos{Line: 5, Column: 10, Offset: 90},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("<element>text</element>"),
				Begin: xmltokenizer.Pos{Line: 6, Column: 3, Offset: 93},
				End:   xmltokenizer.Pos{Line: 7, Column: 40, Offset: 139},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 8, Column: 3, Offset: 142},
				End:          xmltokenizer.Pos{Line: 8, Column: 10, Offset: 149},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Data:  []byte("<element>text</element>"),
				Begin: xmltokenizer.Pos{Line: 9, Column: 3, Offset: 152},
				End:   xmltokenizer.Pos{Line: 12, Column: 8, Offset: 210},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 13, Column: 3, Offset: 213},
				End:          xmltokenizer.Pos{Line: 13, Column: 10, Offset: 220},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 14, Column: 1, Offset: 221},
				End:          xmltokenizer.Pos{Line: 14, Column: 11, Offset: 231},
			},
		}},
		{filename: filepath.Join("corrupted", "cdata_truncated.xml"), expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Name:  xmltokenizer.Name{Local: []byte("content"), Full: []byte("content")},
				Begin: xmltokenizer.Pos{Line: 2, Column: 1, Offset: 40},
				End:   xmltokenizer.Pos{Line: 2, Column: 10, Offset: 49},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("data"), Full: []byte("data")},
				Begin: xmltokenizer.Pos{Line: 3, Column: 3, Offset: 53},
				End:   xmltokenizer.Pos{Line: 3, Column: 9, Offset: 59},
			},
		},
			err: io.ErrUnexpectedEOF,
		},
		{filename: "self_closing.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Name:        xmltokenizer.Name{Local: []byte("a"), Full: []byte("a")},
				SelfClosing: true,
				Begin:       xmltokenizer.Pos{Line: 2, Column: 1, Offset: 39},
				End:         xmltokenizer.Pos{Line: 2, Column: 6, Offset: 44},
			},
			{
				Name:        xmltokenizer.Name{Local: []byte("b"), Full: []byte("b")},
				SelfClosing: true,
				Begin:       xmltokenizer.Pos{Line: 3, Column: 1, Offset: 45},
				End:         xmltokenizer.Pos{Line: 3, Column: 5, Offset: 49},
			},
		}},
		{filename: "copyright_header.xml", expecteds: []xmltokenizer.Token{
			{
				Data:        []byte("<!--\n  Copyright 2024 Example Licence Authors.\n-->"),
				SelfClosing: true,
				Begin:       xmltokenizer.Pos{Line: 1, Column: 1, Offset: 0},
				End:         xmltokenizer.Pos{Line: 3, Column: 4, Offset: 50},
			},
			{
				Data:        []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
				SelfClosing: true,
				Begin:       xmltokenizer.Pos{Line: 4, Column: 1, Offset: 51},
				End:         xmltokenizer.Pos{Line: 4, Column: 39, Offset: 89},
			},
		}},
		{filename: "dtd.xml", expecteds: []xmltokenizer.Token{
			tokenHeader,
			{
				Data: []byte("<!DOCTYPE note [\n" +
					"  <!ENTITY nbsp \"&#xA0;\">\n" +
					"  <!ENTITY writer \"Writer: Donald Duck.\">\n" +
					"  <!ENTITY copyright \"Copyright: W3Schools.\">\n" +
					"]>"),
				SelfClosing: true,
				Begin:       xmltokenizer.Pos{2, 1, 39},
				End:         xmltokenizer.Pos{6, 3, 172},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("note"), Full: []byte("note")},
				Begin: xmltokenizer.Pos{8, 1, 174},
				End:   xmltokenizer.Pos{8, 7, 180},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("to"), Full: []byte("to")},
				Data:  []byte("Tove"),
				Begin: xmltokenizer.Pos{9, 3, 183},
				End:   xmltokenizer.Pos{9, 11, 191},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("to"), Full: []byte("to")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{9, 11, 191},
				End:          xmltokenizer.Pos{9, 16, 196},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("from"), Full: []byte("from")},
				Data:  []byte("Jani"),
				Begin: xmltokenizer.Pos{10, 3, 199},
				End:   xmltokenizer.Pos{10, 13, 209},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("from"), Full: []byte("from")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 10, Column: 13, Offset: 209},
				End:          xmltokenizer.Pos{Line: 10, Column: 20, Offset: 216},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("heading"), Full: []byte("heading")},
				Data:  []byte("Reminder"),
				Begin: xmltokenizer.Pos{11, 3, 219},
				End:   xmltokenizer.Pos{11, 20, 236},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("heading"), Full: []byte("heading")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{11, 20, 236},
				End:          xmltokenizer.Pos{11, 30, 246},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
				Data:  []byte("Don't forget me this weekend!"),
				Begin: xmltokenizer.Pos{12, 3, 249},
				End:   xmltokenizer.Pos{12, 38, 284},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("body"), Full: []byte("body")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{12, 38, 284},
				End:          xmltokenizer.Pos{12, 45, 291},
			},
			{
				Name:  xmltokenizer.Name{Local: []byte("footer"), Full: []byte("footer")},
				Data:  []byte("&writer;&nbsp;&copyright;"),
				Begin: xmltokenizer.Pos{Line: 13, Column: 3, Offset: 294},
				End:   xmltokenizer.Pos{Line: 13, Column: 36, Offset: 327},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("footer"), Full: []byte("footer")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 13, Column: 36, Offset: 327},
				End:          xmltokenizer.Pos{Line: 13, Column: 45, Offset: 336},
			},
			{
				Name:         xmltokenizer.Name{Local: []byte("note"), Full: []byte("note")},
				IsEndElement: true,
				Begin:        xmltokenizer.Pos{Line: 14, Column: 1, Offset: 337},
				End:          xmltokenizer.Pos{Line: 14, Column: 8, Offset: 344},
			},
		}},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d], %s", i, tc.filename), func(t *testing.T) {
			path := filepath.Join("testdata", tc.filename)
			f, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			checkTokens(t, f, tc.expecteds, tc.err)
		})
	}
}

func checkTokens(t *testing.T, r io.Reader, expected []xmltokenizer.Token, expectedErr error) {
	t.Helper()
	tok := xmltokenizer.New(r, xmltokenizer.WithReadBufferSize(1))
	for j := 0; ; j++ {
		token, err := tok.Token()
		if err == io.EOF {
			if j != len(expected) {
				t.Errorf("too few tokens; wanted %d but got %d", len(expected), j)
			}
			break
		}
		if err != nil {
			if !errors.Is(err, expectedErr) {
				t.Fatalf("expected error: %v, got: %v", expectedErr, err)
			}
			return
		}

		if diff := cmp.Diff(token, expected[j]); diff != "" {
			t.Errorf("token #%d: got %#v; diff: %s", j+1, token, diff)
		}
	}
}

func TestTokenOnGPXFiles(t *testing.T) {
	filepath.Walk("testdata", func(path string, info fs.FileInfo, _ error) error {
		t.Run(path, func(t *testing.T) {
			if info.IsDir() {
				return
			}
			if strings.ToLower(filepath.Ext(path)) != ".gpx" {
				return
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Skip(err)
			}

			gpx1, err := gpx.UnmarshalWithXMLTokenizer(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("xmltokenizer: %v", err)
			}

			gpx2, err := gpx.UnmarshalWithStdlibXML(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("xml: %v", err)
			}

			if diff := cmp.Diff(gpx1, gpx2,
				cmp.Transformer("float64", func(x float64) uint64 {
					return math.Float64bits(x)
				}),
			); diff != "" {
				t.Fatal(diff)
			}
		})

		return nil
	})
}

func TestTokenOnXLSXFiles(t *testing.T) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	sheet1, err := xlsx.UnmarshalWithXMLTokenizer(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("xmltokenizer: %v", err)
	}
	sheet2, err := xlsx.UnmarshalWithStdlibXML(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("xml: %v", err)
	}

	if diff := cmp.Diff(sheet1, sheet2); diff != "" {
		t.Fatal(diff)
	}
}

func TestAutoGrowBufferCorrectness(t *testing.T) {
	path := filepath.Join("testdata", "xlsx_sheet1.xml")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tok := xmltokenizer.New(f,
		xmltokenizer.WithReadBufferSize(1),
	)

	var token xmltokenizer.Token
	var sheetData1 schema.SheetData
loop:
	for {
		token, err = tok.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		switch string(token.Name.Local) {
		case "sheetData":
			se := xmltokenizer.GetToken().Copy(token)
			err = sheetData1.UnmarshalToken(tok, se)
			xmltokenizer.PutToken(se)
			if err != nil {
				t.Fatal(err)
			}
			break loop
		}
	}

	f2, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	sheetData2, err := xlsx.UnmarshalWithStdlibXML(f2)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(sheetData1, sheetData2); diff != "" {
		t.Fatal(err)
	}
}

func TestRawTokenWithInmemXML(t *testing.T) {
	tt := []struct {
		name      string
		xml       string
		expecteds []string
		err       error
	}{
		{
			name: "simple xml happy flow",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
	"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<body xmlns:foo="ns1" xmlns="ns2" xmlns:tag="ns3" ` +
				"\r\n\t" + `  >
	<hello lang="en">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔</hello>
	<query>&何; &is-it;</query>
	<goodbye />
	<outer foo:attr="value" xmlns:tag="ns4">
	<inner/>
	</outer>
	<tag:name>
	<![CDATA[Some text here.]]>
	</tag:name>
</body><!-- missing final newline -->`, // Note: retrieved from stdlib xml test.
			expecteds: []string{
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
				"<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"\n" +
					"	\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">",
				"<body xmlns:foo=\"ns1\" xmlns=\"ns2\" xmlns:tag=\"ns3\" " +
					"\r\n\t" + "  >",
				"<hello lang=\"en\">World &lt;&gt;&apos;&quot; &#x767d;&#40300;翔",
				"</hello>",
				"<query>&何; &is-it;",
				"</query>",
				"<goodbye />",
				"<outer foo:attr=\"value\" xmlns:tag=\"ns4\">",
				"<inner/>",
				"</outer>",
				"<tag:name>\n	<![CDATA[Some text here.]]>",
				"</tag:name>",
				"</body>",
				"<!-- missing final newline -->",
			},
		},
		{
			name: "unexpected EOF truncated XML after `<!`",
			xml:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?><!",
			expecteds: []string{
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
				"<!",
			},
			err: io.ErrUnexpectedEOF,
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("[%d]: %s", i, tc.name), func(t *testing.T) {
			tok := xmltokenizer.New(
				bytes.NewReader([]byte(tc.xml)),
				xmltokenizer.WithReadBufferSize(1), // Read per char so we can cover more code paths
			)

			for i := 0; ; i++ {
				token, err := tok.RawToken()
				if err == io.EOF {
					break
				}
				if err != nil {
					if !errors.Is(err, tc.err) {
						t.Fatalf("expected error: %v, got: %v", tc.err, err)
					}
					return
				}
				if diff := cmp.Diff(string(token), tc.expecteds[i]); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}

	t.Run("with prior error", func(t *testing.T) {
		// Test in case RawToken() is reinvoked when there is prior error.
		tok := xmltokenizer.New(bytes.NewReader([]byte{}))
		token, err := tok.RawToken()
		if err != io.EOF {
			t.Fatalf("expected error: %v, got: %v", io.EOF, err)
		}
		_ = token
		token, err = tok.RawToken() // Reinvoke
		if err != io.EOF {
			t.Fatalf("expected error: %v, got: %v", io.EOF, err)
		}
		_ = token
	})
}
