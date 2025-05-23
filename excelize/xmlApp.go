package excelize

import "encoding/xml"

// AppProperties directly maps the document application properties.
type AppProperties struct {
	Application       string
	ScaleCrop         bool
	DocSecurity       int
	Company           string
	LinksUpToDate     bool
	HyperlinksChanged bool
	AppVersion        string
}

// xlsxProperties specifies to an OOXML document properties such as the
// template used, the number of pages and words, and the application name and
// version.
type xlsxProperties struct {
	XMLName              xml.Name `xml:"http://schemas.openxmlformats.org/officeDocument/2006/extended-properties Properties"`
	Vt                   string   `xml:"xmlns:vt,attr"`
	Template             string   `xml:",omitempty"`
	Manager              string   `xml:",omitempty"`
	Company              string   `xml:",omitempty"`
	Pages                int      `xml:",omitempty"`
	Words                int      `xml:",omitempty"`
	Characters           int      `xml:",omitempty"`
	PresentationFormat   string   `xml:",omitempty"`
	Lines                int      `xml:",omitempty"`
	Paragraphs           int      `xml:",omitempty"`
	Slides               int      `xml:",omitempty"`
	Notes                int      `xml:",omitempty"`
	TotalTime            int      `xml:",omitempty"`
	HiddenSlides         int      `xml:",omitempty"`
	MMClips              int      `xml:",omitempty"`
	ScaleCrop            bool     `xml:",omitempty"`
	HeadingPairs         *xlsxVectorVariant
	TitlesOfParts        *xlsxVectorLpstr
	LinksUpToDate        bool   `xml:",omitempty"`
	CharactersWithSpaces int    `xml:",omitempty"`
	SharedDoc            bool   `xml:",omitempty"`
	HyperlinkBase        string `xml:",omitempty"`
	HLinks               *xlsxVectorVariant
	HyperlinksChanged    bool `xml:",omitempty"`
	DigSig               *xlsxDigSig
	Application          string `xml:",omitempty"`
	AppVersion           string `xml:",omitempty"`
	DocSecurity          int    `xml:",omitempty"`
}

// xlsxVectorVariant specifies the set of hyperlinks that were in this
// document when last saved.
type xlsxVectorVariant struct {
	Content string `xml:",innerxml"`
}

type xlsxVectorLpstr struct {
	Content string `xml:",innerxml"`
}

// xlsxDigSig contains the signature of a digitally signed document.
type xlsxDigSig struct {
	Content string `xml:",innerxml"`
}
