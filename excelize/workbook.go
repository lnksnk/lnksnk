package excelize

import (
	"bytes"
	"encoding/xml"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// SetWorkbookProps provides a function to sets workbook properties.
func (f *File) SetWorkbookProps(opts *WorkbookPropsOptions) error {
	wb, err := f.workbookReader()
	if err != nil {
		return err
	}
	if wb.WorkbookPr == nil {
		wb.WorkbookPr = new(xlsxWorkbookPr)
	}
	if opts == nil {
		return nil
	}
	if opts.Date1904 != nil {
		wb.WorkbookPr.Date1904 = *opts.Date1904
	}
	if opts.FilterPrivacy != nil {
		wb.WorkbookPr.FilterPrivacy = *opts.FilterPrivacy
	}
	if opts.CodeName != nil {
		wb.WorkbookPr.CodeName = *opts.CodeName
	}
	return nil
}

// GetWorkbookProps provides a function to gets workbook properties.
func (f *File) GetWorkbookProps() (WorkbookPropsOptions, error) {
	var opts WorkbookPropsOptions
	wb, err := f.workbookReader()
	if err != nil {
		return opts, err
	}
	if wb.WorkbookPr != nil {
		opts.Date1904 = boolPtr(wb.WorkbookPr.Date1904)
		opts.FilterPrivacy = boolPtr(wb.WorkbookPr.FilterPrivacy)
		opts.CodeName = stringPtr(wb.WorkbookPr.CodeName)
	}
	return opts, err
}

// ProtectWorkbook provides a function to prevent other users from viewing
// hidden worksheets, adding, moving, deleting, or hiding worksheets, and
// renaming worksheets in a workbook. The optional field AlgorithmName
// specified hash algorithm, support XOR, MD4, MD5, SHA-1, SHA2-56, SHA-384,
// and SHA-512 currently, if no hash algorithm specified, will be using the XOR
// algorithm as default. The generated workbook only works on Microsoft Office
// 2007 and later. For example, protect workbook with protection settings:
//
//	err := f.ProtectWorkbook(&excelize.WorkbookProtectionOptions{
//	    Password:      "password",
//	    LockStructure: true,
//	})
func (f *File) ProtectWorkbook(opts *WorkbookProtectionOptions) error {
	wb, err := f.workbookReader()
	if err != nil {
		return err
	}
	if wb.WorkbookProtection == nil {
		wb.WorkbookProtection = new(xlsxWorkbookProtection)
	}
	if opts == nil {
		opts = &WorkbookProtectionOptions{}
	}
	wb.WorkbookProtection = &xlsxWorkbookProtection{
		LockStructure: opts.LockStructure,
		LockWindows:   opts.LockWindows,
	}
	if opts.Password != "" {
		if opts.AlgorithmName == "" {
			opts.AlgorithmName = "SHA-512"
		}
		hashValue, saltValue, err := genISOPasswdHash(opts.Password, opts.AlgorithmName, "", int(workbookProtectionSpinCount))
		if err != nil {
			return err
		}
		wb.WorkbookProtection.WorkbookAlgorithmName = opts.AlgorithmName
		wb.WorkbookProtection.WorkbookSaltValue = saltValue
		wb.WorkbookProtection.WorkbookHashValue = hashValue
		wb.WorkbookProtection.WorkbookSpinCount = int(workbookProtectionSpinCount)
	}
	return nil
}

// UnprotectWorkbook provides a function to remove protection for workbook,
// specified the optional password parameter to remove workbook protection with
// password verification.
func (f *File) UnprotectWorkbook(password ...string) error {
	wb, err := f.workbookReader()
	if err != nil {
		return err
	}
	// password verification
	if len(password) > 0 {
		if wb.WorkbookProtection == nil {
			return ErrUnprotectWorkbook
		}
		if wb.WorkbookProtection.WorkbookAlgorithmName != "" {
			// check with given salt value
			hashValue, _, err := genISOPasswdHash(password[0], wb.WorkbookProtection.WorkbookAlgorithmName, wb.WorkbookProtection.WorkbookSaltValue, wb.WorkbookProtection.WorkbookSpinCount)
			if err != nil {
				return err
			}
			if wb.WorkbookProtection.WorkbookHashValue != hashValue {
				return ErrUnprotectWorkbookPassword
			}
		}
	}
	wb.WorkbookProtection = nil
	return err
}

// setWorkbook update workbook property of the spreadsheet. Maximum 31
// characters are allowed in sheet title.
func (f *File) setWorkbook(name string, sheetID, rid int) {
	wb, _ := f.workbookReader()
	wb.Sheets.Sheet = append(wb.Sheets.Sheet, xlsxSheet{
		Name:    name,
		SheetID: sheetID,
		ID:      "rId" + strconv.Itoa(rid),
	})
}

// getWorkbookPath provides a function to get the path of the workbook.xml in
// the spreadsheet.
func (f *File) getWorkbookPath() (path string) {
	if rels, _ := f.relsReader("_rels/.rels"); rels != nil {
		rels.mu.Lock()
		defer rels.mu.Unlock()
		for _, rel := range rels.Relationships {
			if rel.Type == SourceRelationshipOfficeDocument {
				path = strings.TrimPrefix(rel.Target, "/")
				return
			}
		}
	}
	return
}

// getWorkbookRelsPath provides a function to get the path of the workbook.xml.rels
// in the spreadsheet.
func (f *File) getWorkbookRelsPath() (path string) {
	wbPath := f.getWorkbookPath()
	wbDir := filepath.Dir(wbPath)
	if wbDir == "." {
		path = "_rels/" + filepath.Base(wbPath) + ".rels"
		return
	}
	path = strings.TrimPrefix(filepath.Dir(wbPath)+"/_rels/"+filepath.Base(wbPath)+".rels", "/")
	return
}

// deleteWorkbookRels provides a function to delete relationships in
// xl/_rels/workbook.xml.rels by given type and target.
func (f *File) deleteWorkbookRels(relType, relTarget string) (string, error) {
	var rID string
	rels, err := f.relsReader(f.getWorkbookRelsPath())
	if err != nil {
		return rID, err
	}
	if rels == nil {
		rels = &xlsxRelationships{}
	}
	for k, v := range rels.Relationships {
		if v.Type == relType && v.Target == relTarget {
			rID = v.ID
			rels.Relationships = append(rels.Relationships[:k], rels.Relationships[k+1:]...)
		}
	}
	return rID, err
}

// workbookReader provides a function to get the pointer to the workbook.xml
// structure after deserialization.
func (f *File) workbookReader() (*xlsxWorkbook, error) {
	var err error
	if f.WorkBook == nil {
		wbPath := f.getWorkbookPath()
		f.WorkBook = new(xlsxWorkbook)
		if attrs, ok := f.xmlAttr.Load(wbPath); !ok {
			d := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath))))
			if attrs == nil {
				attrs = []xml.Attr{}
			}
			attrs = append(attrs.([]xml.Attr), getRootElement(d)...)
			f.xmlAttr.Store(wbPath, attrs)
			f.addNameSpaces(wbPath, SourceRelationship)
		}
		if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath)))).
			Decode(f.WorkBook); err != nil && err != io.EOF {
			return f.WorkBook, err
		}
	}
	return f.WorkBook, err
}

// workBookWriter provides a function to save workbook.xml after serialize
// structure.
func (f *File) workBookWriter() {
	if f.WorkBook != nil {
		if f.WorkBook.DecodeAlternateContent != nil {
			f.WorkBook.AlternateContent = &xlsxAlternateContent{
				Content: f.WorkBook.DecodeAlternateContent.Content,
				XMLNSMC: SourceRelationshipCompatibility.Value,
			}
		}
		f.WorkBook.DecodeAlternateContent = nil
		output, _ := xml.Marshal(f.WorkBook)
		f.saveFileList(f.getWorkbookPath(), replaceRelationshipsBytes(f.replaceNameSpaceBytes(f.getWorkbookPath(), output)))
	}
}

// setContentTypePartRelsExtensions provides a function to set the content type
// for relationship parts and the Main Document part.
func (f *File) setContentTypePartRelsExtensions() error {
	var rels bool
	content, err := f.contentTypesReader()
	if err != nil {
		return err
	}
	for _, v := range content.Defaults {
		if v.Extension == "rels" {
			rels = true
		}
	}
	if !rels {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   "rels",
			ContentType: ContentTypeRelationships,
		})
	}
	return err
}

// setContentTypePartImageExtensions provides a function to set the content type
// for relationship parts and the Main Document part.
func (f *File) setContentTypePartImageExtensions() error {
	imageTypes := map[string]string{
		"bmp": "image/", "jpeg": "image/", "png": "image/", "gif": "image/",
		"svg": "image/", "tiff": "image/", "emf": "image/x-", "wmf": "image/x-",
		"emz": "image/x-", "wmz": "image/x-",
	}
	content, err := f.contentTypesReader()
	if err != nil {
		return err
	}
	content.mu.Lock()
	defer content.mu.Unlock()
	for _, file := range content.Defaults {
		delete(imageTypes, file.Extension)
	}
	for extension, prefix := range imageTypes {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   extension,
			ContentType: prefix + extension,
		})
	}
	return err
}

// setContentTypePartVMLExtensions provides a function to set the content type
// for relationship parts and the Main Document part.
func (f *File) setContentTypePartVMLExtensions() error {
	var vml bool
	content, err := f.contentTypesReader()
	if err != nil {
		return err
	}
	content.mu.Lock()
	defer content.mu.Unlock()
	for _, v := range content.Defaults {
		if v.Extension == "vml" {
			vml = true
		}
	}
	if !vml {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   "vml",
			ContentType: ContentTypeVML,
		})
	}
	return err
}

// addContentTypePart provides a function to add content type part relationships
// in the file [Content_Types].xml by given index and content type.
func (f *File) addContentTypePart(index int, contentType string) error {
	setContentType := map[string]func() error{
		"comments": f.setContentTypePartVMLExtensions,
		"drawings": f.setContentTypePartImageExtensions,
	}
	partNames := map[string]string{
		"chart":         "/xl/charts/chart" + strconv.Itoa(index) + ".xml",
		"chartsheet":    "/xl/chartsheets/sheet" + strconv.Itoa(index) + ".xml",
		"comments":      "/xl/comments" + strconv.Itoa(index) + ".xml",
		"drawings":      "/xl/drawings/drawing" + strconv.Itoa(index) + ".xml",
		"table":         "/xl/tables/table" + strconv.Itoa(index) + ".xml",
		"pivotTable":    "/xl/pivotTables/pivotTable" + strconv.Itoa(index) + ".xml",
		"pivotCache":    "/xl/pivotCache/pivotCacheDefinition" + strconv.Itoa(index) + ".xml",
		"sharedStrings": "/xl/sharedStrings.xml",
		"slicer":        "/xl/slicers/slicer" + strconv.Itoa(index) + ".xml",
		"slicerCache":   "/xl/slicerCaches/slicerCache" + strconv.Itoa(index) + ".xml",
	}
	contentTypes := map[string]string{
		"chart":         ContentTypeDrawingML,
		"chartsheet":    ContentTypeSpreadSheetMLChartsheet,
		"comments":      ContentTypeSpreadSheetMLComments,
		"drawings":      ContentTypeDrawing,
		"table":         ContentTypeSpreadSheetMLTable,
		"pivotTable":    ContentTypeSpreadSheetMLPivotTable,
		"pivotCache":    ContentTypeSpreadSheetMLPivotCacheDefinition,
		"sharedStrings": ContentTypeSpreadSheetMLSharedStrings,
		"slicer":        ContentTypeSlicer,
		"slicerCache":   ContentTypeSlicerCache,
	}
	s, ok := setContentType[contentType]
	if ok {
		if err := s(); err != nil {
			return err
		}
	}
	content, err := f.contentTypesReader()
	if err != nil {
		return err
	}
	content.mu.Lock()
	defer content.mu.Unlock()
	for _, v := range content.Overrides {
		if v.PartName == partNames[contentType] {
			return err
		}
	}
	content.Overrides = append(content.Overrides, xlsxOverride{
		PartName:    partNames[contentType],
		ContentType: contentTypes[contentType],
	})
	return f.setContentTypePartRelsExtensions()
}

// removeContentTypesPart provides a function to remove relationships by given
// content type and part name in the file [Content_Types].xml.
func (f *File) removeContentTypesPart(contentType, partName string) error {
	if !strings.HasPrefix(partName, "/") {
		partName = "/xl/" + partName
	}
	content, err := f.contentTypesReader()
	if err != nil {
		return err
	}
	content.mu.Lock()
	defer content.mu.Unlock()
	for k, v := range content.Overrides {
		if v.PartName == partName && v.ContentType == contentType {
			content.Overrides = append(content.Overrides[:k], content.Overrides[k+1:]...)
		}
	}
	return err
}