package excelize

import "encoding/xml"

// xlsxMetadata directly maps the metadata element. A cell in a spreadsheet
// application can have metadata associated with it. Metadata is just a set of
// additional properties about the particular cell, and this metadata is stored
// in the metadata xml part. There are two types of metadata: cell metadata and
// value metadata. Cell metadata contains information about the cell itself,
// and this metadata can be carried along with the cell as it moves
// (insert, shift, copy/paste, merge, unmerge, etc). Value metadata is
// information about the value of a particular cell. Value metadata properties
// can be propagated along with the value as it is referenced in formulas.
type xlsxMetadata struct {
	XMLName         xml.Name             `xml:"metadata"`
	MetadataTypes   *xlsxInnerXML        `xml:"metadataTypes"`
	MetadataStrings *xlsxInnerXML        `xml:"metadataStrings"`
	MdxMetadata     *xlsxInnerXML        `xml:"mdxMetadata"`
	FutureMetadata  []xlsxFutureMetadata `xml:"futureMetadata"`
	CellMetadata    *xlsxMetadataBlocks  `xml:"cellMetadata"`
	ValueMetadata   *xlsxMetadataBlocks  `xml:"valueMetadata"`
	ExtLst          *xlsxInnerXML        `xml:"extLst"`
}

// xlsxFutureMetadata directly maps the futureMetadata element. This element
// represents future metadata information.
type xlsxFutureMetadata struct {
	Bk     []xlsxFutureMetadataBlock `xml:"bk"`
	ExtLst *xlsxInnerXML             `xml:"extLst"`
}

// xlsxFutureMetadataBlock directly maps the kb element. This element represents
// a block of future metadata information. This is a location for storing
// feature extension information.
type xlsxFutureMetadataBlock struct {
	ExtLst *xlsxInnerXML `xml:"extLst"`
}

// xlsxMetadataBlocks directly maps the metadata element. This element
// represents cell metadata information. Cell metadata is information metadata
// about a specific cell, and it stays tied to that cell position.
type xlsxMetadataBlocks struct {
	Count int                 `xml:"count,attr,omitempty"`
	Bk    []xlsxMetadataBlock `xml:"bk"`
}

// xlsxMetadataBlock directly maps the bk element. This element represents a
// block of metadata records.
type xlsxMetadataBlock struct {
	Rc []xlsxMetadataRecord `xml:"rc"`
}

// xlsxMetadataRecord directly maps the rc element. This element represents a
// reference to a specific metadata record.
type xlsxMetadataRecord struct {
	T int `xml:"t,attr"`
	V int `xml:"v,attr"`
}

// xlsxRichValueData directly maps the rvData element that specifies rich value
// data.
type xlsxRichValueData struct {
	XMLName xml.Name        `xml:"rvData"`
	Count   int             `xml:"count,attr,omitempty"`
	Rv      []xlsxRichValue `xml:"rv"`
	ExtLst  *xlsxInnerXML   `xml:"extLst"`
}

// xlsxRichValue directly maps the rv element that specifies rich value data
// information for a single rich value
type xlsxRichValue struct {
	S  int           `xml:"s,attr"`
	V  []string      `xml:"v"`
	Fb *xlsxInnerXML `xml:"fb"`
}

// xlsxRichValueRels directly maps the richValueRels element. This element that
// specifies a list of rich value relationships.
type xlsxRichValueRels struct {
	XMLName xml.Name                       `xml:"richValueRels"`
	Rels    []xlsxRichValueRelRelationship `xml:"rel"`
	ExtLst  *xlsxInnerXML                  `xml:"extLst"`
}

// xlsxRichValueRelRelationship directly maps the rel element. This element
// specifies a relationship for a rich value property.
type xlsxRichValueRelRelationship struct {
	ID string `xml:"id,attr"`
}

// xlsxWebImagesSupportingRichData directly maps the webImagesSrd element. This
// element specifies a list of sets of properties associated with web image rich
// values.
type xlsxWebImagesSupportingRichData struct {
	XMLName     xml.Name                         `xml:"webImagesSrd"`
	WebImageSrd []xlsxWebImageSupportingRichData `xml:"webImageSrd"`
	ExtLst      *xlsxInnerXML                    `xml:"extLst"`
}

// xlsxWebImageSupportingRichData directly maps the webImageSrd element. This
// element specifies a set of properties for a web image rich value.
type xlsxWebImageSupportingRichData struct {
	Address           xlsxExternalReference `xml:"address"`
	MoreImagesAddress xlsxExternalReference `xml:"moreImagesAddress"`
	Blip              xlsxExternalReference `xml:"blip"`
}
