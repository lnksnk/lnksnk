package excelize

import "encoding/xml"

// vmlDrawing directly maps the root element in the file
// xl/drawings/vmlDrawing%d.vml.
type vmlDrawing struct {
	XMLName     xml.Name         `xml:"xml"`
	XMLNSv      string           `xml:"xmlns:v,attr"`
	XMLNSo      string           `xml:"xmlns:o,attr"`
	XMLNSx      string           `xml:"xmlns:x,attr"`
	XMLNSmv     string           `xml:"xmlns:mv,attr,omitempty"`
	ShapeLayout *xlsxShapeLayout `xml:"o:shapelayout"`
	ShapeType   *xlsxShapeType   `xml:"v:shapetype"`
	Shape       []xlsxShape      `xml:"v:shape"`
}

// xlsxShapeLayout directly maps the shapelayout element. This element contains
// child elements that store information used in the editing and layout of
// shapes.
type xlsxShapeLayout struct {
	Ext   string     `xml:"v:ext,attr"`
	IDmap *xlsxIDmap `xml:"o:idmap"`
}

// xlsxIDmap directly maps the idmap element.
type xlsxIDmap struct {
	Ext  string `xml:"v:ext,attr"`
	Data int    `xml:"data,attr"`
}

// xlsxShape directly maps the shape element.
type xlsxShape struct {
	XMLName     xml.Name `xml:"v:shape"`
	ID          string   `xml:"id,attr"`
	SpID        string   `xml:"o:spid,attr,omitempty"`
	Type        string   `xml:"type,attr"`
	Style       string   `xml:"style,attr"`
	Button      string   `xml:"o:button,attr,omitempty"`
	Filled      string   `xml:"filled,attr,omitempty"`
	FillColor   string   `xml:"fillcolor,attr,omitempty"`
	InsetMode   string   `xml:"urn:schemas-microsoft-com:office:office insetmode,attr,omitempty"`
	Stroked     string   `xml:"stroked,attr,omitempty"`
	StrokeColor string   `xml:"strokecolor,attr,omitempty"`
	Val         string   `xml:",innerxml"`
}

// xlsxShapeType directly maps the shapetype element.
type xlsxShapeType struct {
	ID             string      `xml:"id,attr"`
	CoordSize      string      `xml:"coordsize,attr"`
	Spt            int         `xml:"o:spt,attr"`
	PreferRelative string      `xml:"o:preferrelative,attr,omitempty"`
	Path           string      `xml:"path,attr"`
	Filled         string      `xml:"filled,attr,omitempty"`
	Stroked        string      `xml:"stroked,attr,omitempty"`
	Stroke         *xlsxStroke `xml:"v:stroke"`
	VFormulas      *vFormulas  `xml:"v:formulas"`
	VPath          *vPath      `xml:"v:path"`
	Lock           *oLock      `xml:"o:lock"`
}

// xlsxStroke directly maps the stroke element.
type xlsxStroke struct {
	JoinStyle string `xml:"joinstyle,attr"`
}

// vPath directly maps the v:path element.
type vPath struct {
	ExtrusionOK     string `xml:"o:extrusionok,attr,omitempty"`
	GradientShapeOK string `xml:"gradientshapeok,attr,omitempty"`
	ConnectType     string `xml:"o:connecttype,attr"`
}

// oLock directly maps the o:lock element.
type oLock struct {
	Ext         string `xml:"v:ext,attr"`
	Rotation    string `xml:"rotation,attr,omitempty"`
	AspectRatio string `xml:"aspectratio,attr,omitempty"`
}

// vFormulas directly maps to the v:formulas element
type vFormulas struct {
	Formulas []vFormula `xml:"v:f"`
}

// vFormula directly maps to the v:f element
type vFormula struct {
	Equation string `xml:"eqn,attr"`
}

// vFill directly maps the v:fill element. This element must be defined within a
// Shape element.
type vFill struct {
	Angle  int    `xml:"angle,attr,omitempty"`
	Color2 string `xml:"color2,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Fill   *oFill `xml:"o:fill"`
}

// oFill directly maps the o:fill element.
type oFill struct {
	Ext  string `xml:"v:ext,attr"`
	Type string `xml:"type,attr,omitempty"`
}

// vShadow directly maps the v:shadow element. This element must be defined
// within a Shape element. In addition, the On attribute must be set to True.
type vShadow struct {
	On       string `xml:"on,attr"`
	Color    string `xml:"color,attr,omitempty"`
	Obscured string `xml:"obscured,attr"`
}

// vTextBox directly maps the v:textbox element. This element must be defined
// within a Shape element.
type vTextBox struct {
	Style string   `xml:"style,attr"`
	Div   *xlsxDiv `xml:"div"`
}

// vImageData directly maps the v:imagedata element. This element must be
// defined within a Shape element.
type vImageData struct {
	RelID string `xml:"o:relid,attr"`
	Title string `xml:"o:title,attr,omitempty"`
}

// xlsxDiv directly maps the div element.
type xlsxDiv struct {
	Style string    `xml:"style,attr"`
	Font  []vmlFont `xml:"font"`
}

type vmlFont struct {
	Face    string `xml:"face,attr,omitempty"`
	Size    uint   `xml:"size,attr,omitempty"`
	Color   string `xml:"color,attr,omitempty"`
	Content string `xml:",innerxml"`
}

// xClientData (Attached Object Data) directly maps the x:ClientData element.
// This element specifies data associated with objects attached to a
// spreadsheet. While this element might contain any of the child elements
// below, only certain combinations are meaningful. The ObjectType attribute
// determines the kind of object the element represents and which subset of
// child elements is appropriate. Relevant groups are identified for each child
// element.
type xClientData struct {
	ObjectType    string  `xml:"ObjectType,attr"`
	MoveWithCells *string `xml:"x:MoveWithCells"`
	SizeWithCells *string `xml:"x:SizeWithCells"`
	Anchor        string  `xml:"x:Anchor"`
	Locked        string  `xml:"x:Locked,omitempty"`
	PrintObject   string  `xml:"x:PrintObject,omitempty"`
	AutoFill      string  `xml:"x:AutoFill,omitempty"`
	FmlaMacro     string  `xml:"x:FmlaMacro,omitempty"`
	TextHAlign    string  `xml:"x:TextHAlign,omitempty"`
	TextVAlign    string  `xml:"x:TextVAlign,omitempty"`
	Row           *int    `xml:"x:Row"`
	Column        *int    `xml:"x:Column"`
	Checked       int     `xml:"x:Checked,omitempty"`
	FmlaLink      string  `xml:"x:FmlaLink,omitempty"`
	NoThreeD      *string `xml:"x:NoThreeD"`
	FirstButton   *string `xml:"x:FirstButton"`
	Val           uint    `xml:"x:Val,omitempty"`
	Min           uint    `xml:"x:Min,omitempty"`
	Max           uint    `xml:"x:Max,omitempty"`
	Inc           uint    `xml:"x:Inc,omitempty"`
	Page          uint    `xml:"x:Page,omitempty"`
	Horiz         *string `xml:"x:Horiz"`
	Dx            uint    `xml:"x:Dx,omitempty"`
}

// decodeVmlDrawing defines the structure used to parse the file
// xl/drawings/vmlDrawing%d.vml.
type decodeVmlDrawing struct {
	ShapeType decodeShapeType `xml:"urn:schemas-microsoft-com:vml shapetype"`
	Shape     []decodeShape   `xml:"urn:schemas-microsoft-com:vml shape"`
}

// decodeShapeType defines the structure used to parse the shapetype element in
// the file xl/drawings/vmlDrawing%d.vml.
type decodeShapeType struct {
	ID             string `xml:"id,attr"`
	CoordSize      string `xml:"coordsize,attr"`
	Spt            int    `xml:"spt,attr"`
	PreferRelative string `xml:"preferrelative,attr,omitempty"`
	Path           string `xml:"path,attr"`
	Filled         string `xml:"filled,attr,omitempty"`
	Stroked        string `xml:"stroked,attr,omitempty"`
}

// decodeShape defines the structure used to parse the particular shape element.
type decodeShape struct {
	ID          string `xml:"id,attr"`
	SpID        string `xml:"spid,attr,omitempty"`
	Type        string `xml:"type,attr"`
	Style       string `xml:"style,attr"`
	Button      string `xml:"button,attr,omitempty"`
	Filled      string `xml:"filled,attr,omitempty"`
	FillColor   string `xml:"fillcolor,attr,omitempty"`
	InsetMode   string `xml:"urn:schemas-microsoft-com:office:office insetmode,attr,omitempty"`
	Stroked     string `xml:"stroked,attr,omitempty"`
	StrokeColor string `xml:"strokecolor,attr,omitempty"`
	Val         string `xml:",innerxml"`
}

// decodeShapeVal defines the structure used to parse the sub-element of the
// shape in the file xl/drawings/vmlDrawing%d.vml.
type decodeShapeVal struct {
	TextBox    decodeVMLTextBox    `xml:"textbox"`
	ClientData decodeVMLClientData `xml:"ClientData"`
}

// decodeVMLFontU defines the structure used to parse the u element in the VML.
type decodeVMLFontU struct {
	Class string `xml:"class,attr"`
	Val   string `xml:",chardata"`
}

// decodeVMLFontI defines the structure used to parse the i element in the VML.
type decodeVMLFontI struct {
	U   *decodeVMLFontU `xml:"u"`
	Val string          `xml:",chardata"`
}

// decodeVMLFontB defines the structure used to parse the b element in the VML.
type decodeVMLFontB struct {
	I   *decodeVMLFontI `xml:"i"`
	U   *decodeVMLFontU `xml:"u"`
	Val string          `xml:",chardata"`
}

// decodeVMLFont defines the structure used to parse the font element in the VML.
type decodeVMLFont struct {
	Face  string          `xml:"face,attr,omitempty"`
	Size  uint            `xml:"size,attr,omitempty"`
	Color string          `xml:"color,attr,omitempty"`
	B     *decodeVMLFontB `xml:"b"`
	I     *decodeVMLFontI `xml:"i"`
	U     *decodeVMLFontU `xml:"u"`
	Val   string          `xml:",chardata"`
}

// decodeVMLDiv defines the structure used to parse the div element in the VML.
type decodeVMLDiv struct {
	Font []decodeVMLFont `xml:"font"`
}

// decodeVMLTextBox defines the structure used to parse the v:textbox element in
// the file xl/drawings/vmlDrawing%d.vml.
type decodeVMLTextBox struct {
	Div decodeVMLDiv `xml:"div"`
}

// decodeVMLClientData defines the structure used to parse the x:ClientData
// element in the file xl/drawings/vmlDrawing%d.vml.
type decodeVMLClientData struct {
	ObjectType string `xml:"ObjectType,attr"`
	Anchor     string
	FmlaMacro  string
	Column     *int
	Row        *int
	Checked    int
	FmlaLink   string
	Val        uint
	Min        uint
	Max        uint
	Inc        uint
	Page       uint
	Horiz      *string
}

// encodeShape defines the structure used to re-serialization shape element.
type encodeShape struct {
	Fill       *vFill       `xml:"v:fill"`
	Shadow     *vShadow     `xml:"v:shadow"`
	Path       *vPath       `xml:"v:path"`
	TextBox    *vTextBox    `xml:"v:textbox"`
	ImageData  *vImageData  `xml:"v:imagedata"`
	ClientData *xClientData `xml:"x:ClientData"`
	Lock       *oLock       `xml:"o:lock"`
}

// formCtrlPreset defines the structure used to form control presets.
type formCtrlPreset struct {
	autoFill     string
	fill         *vFill
	fillColor    string
	filled       string
	firstButton  *string
	noThreeD     *string
	objectType   string
	shadow       *vShadow
	strokeButton string
	strokeColor  string
	stroked      string
	textHAlign   string
	textVAlign   string
}

// vmlOptions defines the structure used to internal comments and form controls.
type vmlOptions struct {
	formCtrl bool
	sheet    string
	Comment
	FormControl
}

// FormControl directly maps the form controls information.
type FormControl struct {
	Cell         string
	Macro        string
	Width        uint
	Height       uint
	Checked      bool
	CurrentVal   uint
	MinVal       uint
	MaxVal       uint
	IncChange    uint
	PageChange   uint
	Horizontally bool
	CellLink     string
	Text         string
	Paragraph    []RichTextRun
	Type         FormControlType
	Format       GraphicOptions
}

// HeaderFooterImageOptions defines the settings for an image to be accessible
// from the worksheet header and footer options.
type HeaderFooterImageOptions struct {
	Position  HeaderFooterImagePositionType
	File      []byte
	IsFooter  bool
	FirstPage bool
	Extension string
	Width     string
	Height    string
}
