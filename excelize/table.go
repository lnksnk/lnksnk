package excelize

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	expressionFormat = regexp.MustCompile(`"(?:[^"]|"")*"|\S+`)
	conditionFormat  = regexp.MustCompile(`(or|\|\|)`)
	blankFormat      = regexp.MustCompile("blanks|nonblanks")
	matchFormat      = regexp.MustCompile("[*?]")
)

// parseTableOptions provides a function to parse the format settings of the
// table with default value.
func parseTableOptions(opts *Table) (*Table, error) {
	var err error
	if opts == nil {
		return &Table{ShowRowStripes: boolPtr(true)}, err
	}
	if opts.ShowRowStripes == nil {
		opts.ShowRowStripes = boolPtr(true)
	}
	if err = checkDefinedName(opts.Name); err != nil {
		return opts, err
	}
	return opts, err
}

// AddTable provides the method to add table in a worksheet by given worksheet
// name, range reference and format set. For example, create a table of A1:D5
// on Sheet1:
//
//	err := f.AddTable("Sheet1", &excelize.Table{Range: "A1:D5"})
//
// Create a table of F2:H6 on Sheet2 with format set:
//
//	disable := false
//	err := f.AddTable("Sheet2", &excelize.Table{
//	    Range:             "F2:H6",
//	    Name:              "table",
//	    StyleName:         "TableStyleMedium2",
//	    ShowFirstColumn:   true,
//	    ShowLastColumn:    true,
//	    ShowRowStripes:    &disable,
//	    ShowColumnStripes: true,
//	})
//
// Note that the table must be at least two lines including the header. The
// header cells must contain strings and must be unique, and must set the
// header row data of the table before calling the AddTable function. Multiple
// tables range reference that can't have an intersection.
//
// Name: The name of the table, in the same worksheet name of the table should
// be unique, starts with a letter or underscore (_), doesn't include a
// space or character, and should be no more than 255 characters
//
// StyleName: The built-in table style names
//
//	TableStyleLight1 - TableStyleLight21
//	TableStyleMedium1 - TableStyleMedium28
//	TableStyleDark1 - TableStyleDark11
func (f *File) AddTable(sheet string, table *Table) error {
	options, err := parseTableOptions(table)
	if err != nil {
		return err
	}
	var exist bool
	f.Pkg.Range(func(k, v interface{}) bool {
		if strings.Contains(k.(string), "xl/tables/table") {
			var t xlsxTable
			if err := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(v.([]byte)))).
				Decode(&t); err != nil && err != io.EOF {
				return true
			}
			if exist = t.Name == options.Name; exist {
				return false
			}
		}
		return true
	})
	if exist {
		return ErrExistsTableName
	}
	// Coordinate conversion, convert C1:B3 to 2,0,1,2.
	coordinates, err := rangeRefToCoordinates(options.Range)
	if err != nil {
		return err
	}
	// Correct table reference range, such correct C1:B3 to B1:C3.
	_ = sortCoordinates(coordinates)
	tableID := f.countTables() + 1
	sheetRelationshipsTableXML := "../tables/table" + strconv.Itoa(tableID) + ".xml"
	tableXML := strings.ReplaceAll(sheetRelationshipsTableXML, "..", "xl")
	// Add first table for given sheet.
	sheetXMLPath, _ := f.getSheetXMLPath(sheet)
	sheetRels := "xl/worksheets/_rels/" + strings.TrimPrefix(sheetXMLPath, "xl/worksheets/") + ".rels"
	rID := f.addRels(sheetRels, SourceRelationshipTable, sheetRelationshipsTableXML, "")
	if err = f.addSheetTable(sheet, rID); err != nil {
		return err
	}
	f.addSheetNameSpace(sheet, SourceRelationship)
	if err = f.addTable(sheet, tableXML, coordinates[0], coordinates[1], coordinates[2], coordinates[3], tableID, options); err != nil {
		return err
	}
	return f.addContentTypePart(tableID, "table")
}

// GetTables provides the method to get all tables in a worksheet by given
// worksheet name.
func (f *File) GetTables(sheet string) ([]Table, error) {
	var tables []Table
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return tables, err
	}
	if ws.TableParts == nil {
		return tables, err
	}
	for _, tbl := range ws.TableParts.TableParts {
		if tbl != nil {
			target := f.getSheetRelationshipsTargetByID(sheet, tbl.RID)
			tableXML := strings.ReplaceAll(target, "..", "xl")
			content, ok := f.Pkg.Load(tableXML)
			if !ok {
				continue
			}
			var t xlsxTable
			if err := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(content.([]byte)))).
				Decode(&t); err != nil && err != io.EOF {
				return tables, err
			}
			table := Table{
				rID:      tbl.RID,
				tID:      t.ID,
				tableXML: tableXML,
				Range:    t.Ref,
				Name:     t.Name,
			}
			if t.TableStyleInfo != nil {
				table.StyleName = t.TableStyleInfo.Name
				table.ShowColumnStripes = t.TableStyleInfo.ShowColumnStripes
				table.ShowFirstColumn = t.TableStyleInfo.ShowFirstColumn
				table.ShowLastColumn = t.TableStyleInfo.ShowLastColumn
				table.ShowRowStripes = &t.TableStyleInfo.ShowRowStripes
			}
			tables = append(tables, table)
		}
	}
	return tables, err
}

// DeleteTable provides the method to delete table by given table name.
func (f *File) DeleteTable(name string) error {
	if err := checkDefinedName(name); err != nil {
		return err
	}
	tbls, err := f.getTables()
	if err != nil {
		return err
	}
	for sheet, tables := range tbls {
		for _, table := range tables {
			if table.Name != name {
				continue
			}
			ws, _ := f.workSheetReader(sheet)
			for i, tbl := range ws.TableParts.TableParts {
				if tbl.RID == table.rID {
					ws.TableParts.TableParts = append(ws.TableParts.TableParts[:i], ws.TableParts.TableParts[i+1:]...)
					f.Pkg.Delete(table.tableXML)
					_ = f.removeContentTypesPart(ContentTypeSpreadSheetMLTable, "/"+table.tableXML)
					f.deleteSheetRelationships(sheet, tbl.RID)
					break
				}
			}
			if ws.TableParts.Count = len(ws.TableParts.TableParts); ws.TableParts.Count == 0 {
				ws.TableParts = nil
			}
			return err
		}
	}
	return newNoExistTableError(name)
}

// getTables provides a function to get all tables in a workbook.
func (f *File) getTables() (map[string][]Table, error) {
	tables := map[string][]Table{}
	for _, sheetName := range f.GetSheetList() {
		tbls, err := f.GetTables(sheetName)
		e := ErrSheetNotExist{sheetName}
		if err != nil && err.Error() != newNotWorksheetError(sheetName).Error() && err.Error() != e.Error() {
			return tables, err
		}
		tables[sheetName] = append(tables[sheetName], tbls...)
	}
	return tables, nil
}

// countTables provides a function to get table files count storage in the
// folder xl/tables.
func (f *File) countTables() int {
	count := 0
	f.Pkg.Range(func(k, v interface{}) bool {
		if strings.Contains(k.(string), "xl/tables/tableSingleCells") {
			var cells xlsxSingleXMLCells
			if err := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(v.([]byte)))).
				Decode(&cells); err != nil && err != io.EOF {
				count++
				return true
			}
			for _, cell := range cells.SingleXmlCell {
				if count < cell.ID {
					count = cell.ID
				}
			}
		}
		if strings.Contains(k.(string), "xl/tables/table") {
			var t xlsxTable
			if err := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(v.([]byte)))).
				Decode(&t); err != nil && err != io.EOF {
				count++
				return true
			}
			if count < t.ID {
				count = t.ID
			}
		}
		return true
	})
	return count
}

// addSheetTable provides a function to add tablePart element to
// xl/worksheets/sheet%d.xml by given worksheet name and relationship index.
func (f *File) addSheetTable(sheet string, rID int) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	table := &xlsxTablePart{
		RID: "rId" + strconv.Itoa(rID),
	}
	if ws.TableParts == nil {
		ws.TableParts = &xlsxTableParts{}
	}
	ws.TableParts.Count++
	ws.TableParts.TableParts = append(ws.TableParts.TableParts, table)
	return err
}

// setTableColumns provides a function to set cells value in header row for the
// table.
func (f *File) setTableColumns(sheet string, showHeaderRow bool, x1, y1, x2 int, tbl *xlsxTable) error {
	var (
		idx            int
		header         []string
		tableColumns   []*xlsxTableColumn
		getTableColumn = func(name string) *xlsxTableColumn {
			if tbl != nil && tbl.TableColumns != nil {
				for _, column := range tbl.TableColumns.TableColumn {
					if column.Name == name {
						return column
					}
				}
			}
			return nil
		}
	)
	for i := x1; i <= x2; i++ {
		idx++
		cell, err := CoordinatesToCellName(i, y1)
		if err != nil {
			return err
		}
		name, _ := f.GetCellValue(sheet, cell, Options{RawCellValue: true})
		if _, err := strconv.Atoi(name); err == nil {
			if showHeaderRow {
				_ = f.SetCellStr(sheet, cell, name)
			}
		}
		if name == "" || inStrSlice(header, name, true) != -1 {
			name = "Column" + strconv.Itoa(idx)
			if showHeaderRow {
				_ = f.SetCellStr(sheet, cell, name)
			}
		}
		header = append(header, name)
		if column := getTableColumn(name); column != nil {
			column.ID, column.DataDxfID, column.QueryTableFieldID = idx, 0, 0
			tableColumns = append(tableColumns, column)
			continue
		}
		tableColumns = append(tableColumns, &xlsxTableColumn{
			ID:   idx,
			Name: name,
		})
	}
	tbl.TableColumns = &xlsxTableColumns{
		Count:       len(tableColumns),
		TableColumn: tableColumns,
	}
	return nil
}

// checkDefinedName check whether there are illegal characters in the defined
// name or table name. Verify that the name:
// 1. Starts with a letter or underscore (_)
// 2. Doesn't include a space or character that isn't allowed
func checkDefinedName(name string) error {
	if utf8.RuneCountInString(name) > MaxFieldLength {
		return ErrNameLength
	}
	inCodeRange := func(code int, tbl []int) bool {
		for i := 0; i < len(tbl); i += 2 {
			if tbl[i] <= code && code <= tbl[i+1] {
				return true
			}
		}
		return false
	}
	for i, c := range name {
		if i == 0 {
			if inCodeRange(int(c), supportedDefinedNameAtStartCharCodeRange) {
				continue
			}
			return newInvalidNameError(name)
		}
		if inCodeRange(int(c), supportedDefinedNameAfterStartCharCodeRange) {
			continue
		}
		return newInvalidNameError(name)
	}
	return nil
}

// addTable provides a function to add table by given worksheet name,
// range reference and format set.
func (f *File) addTable(sheet, tableXML string, x1, y1, x2, y2, i int, opts *Table) error {
	// Correct the minimum number of rows, the table at least two lines.
	if y1 == y2 {
		y2++
	}
	hideHeaderRow := opts != nil && opts.ShowHeaderRow != nil && !*opts.ShowHeaderRow
	if hideHeaderRow {
		y1++
	}
	// Correct table range reference, such correct C1:B3 to B1:C3.
	ref, err := coordinatesToRangeRef([]int{x1, y1, x2, y2})
	if err != nil {
		return err
	}
	name := opts.Name
	if name == "" {
		name = "Table" + strconv.Itoa(i)
	}
	t := xlsxTable{
		XMLNS:       NameSpaceSpreadSheet.Value,
		ID:          i,
		Name:        name,
		DisplayName: name,
		Ref:         ref,
		AutoFilter: &xlsxAutoFilter{
			Ref: ref,
		},
		TableStyleInfo: &xlsxTableStyleInfo{
			Name:              opts.StyleName,
			ShowFirstColumn:   opts.ShowFirstColumn,
			ShowLastColumn:    opts.ShowLastColumn,
			ShowRowStripes:    *opts.ShowRowStripes,
			ShowColumnStripes: opts.ShowColumnStripes,
		},
	}
	_ = f.setTableColumns(sheet, !hideHeaderRow, x1, y1, x2, &t)
	if hideHeaderRow {
		t.AutoFilter = nil
		t.HeaderRowCount = intPtr(0)
	}
	table, err := xml.Marshal(t)
	f.saveFileList(tableXML, table)
	return err
}

// AutoFilter provides the method to add auto filter in a worksheet by given
// worksheet name, range reference and settings. An auto filter in Excel is a
// way of filtering a 2D range of data based on some simple criteria. For
// example applying an auto filter to a cell range A1:D4 in the Sheet1:
//
//	err := f.AutoFilter("Sheet1", "A1:D4", []excelize.AutoFilterOptions{})
//
// Filter data in an auto filter:
//
//	err := f.AutoFilter("Sheet1", "A1:D4", []excelize.AutoFilterOptions{
//	    {Column: "B", Expression: "x != blanks"},
//	})
//
// Column defines the filter columns in an auto filter range based on simple
// criteria
//
// It isn't sufficient to just specify the filter condition. You must also
// hide any rows that don't match the filter condition. Rows are hidden using
// the SetRowVisible function. Excelize can't filter rows automatically since
// this isn't part of the file format.
//
// Setting a filter criteria for a column:
//
// Expression defines the conditions, the following operators are available
// for setting the filter criteria:
//
//	==
//	!=
//	>
//	<
//	>=
//	<=
//	and
//	or
//
// An expression can comprise a single statement or two statements separated
// by the 'and' and 'or' operators. For example:
//
//	x <  2000
//	x >  2000
//	x == 2000
//	x >  2000 and x <  5000
//	x == 2000 or  x == 5000
//
// Filtering of blank or non-blank data can be achieved by using a value of
// Blanks or NonBlanks in the expression:
//
//	x == Blanks
//	x == NonBlanks
//
// Excel also allows some simple string matching operations:
//
//	x == b*      // begins with b
//	x != b*      // doesn't begin with b
//	x == *b      // ends with b
//	x != *b      // doesn't end with b
//	x == *b*     // contains b
//	x != *b*     // doesn't contain b
//
// You can also use '*' to match any character or number and '?' to match any
// single character or number. No other regular expression quantifier is
// supported by Excel's filters. Excel's regular expression characters can be
// escaped using '~'.
//
// The placeholder variable x in the above examples can be replaced by any
// simple string. The actual placeholder name is ignored internally so the
// following are all equivalent:
//
//	x     < 2000
//	col   < 2000
//	Price < 2000
func (f *File) AutoFilter(sheet, rangeRef string, opts []AutoFilterOptions) error {
	coordinates, err := rangeRefToCoordinates(rangeRef)
	if err != nil {
		return err
	}
	_ = sortCoordinates(coordinates)
	// Correct reference range, such correct C1:B3 to B1:C3.
	ref, _ := coordinatesToRangeRef(coordinates, true)
	wb, err := f.workbookReader()
	if err != nil {
		return err
	}
	sheetID, err := f.GetSheetIndex(sheet)
	if err != nil {
		return err
	}
	filterRange := fmt.Sprintf("'%s'!%s", sheet, ref)
	d := xlsxDefinedName{
		Name:         builtInDefinedNames[3],
		Hidden:       true,
		LocalSheetID: intPtr(sheetID),
		Data:         filterRange,
	}
	if wb.DefinedNames == nil {
		wb.DefinedNames = &xlsxDefinedNames{
			DefinedName: []xlsxDefinedName{d},
		}
	} else {
		var definedNameExists bool
		for idx := range wb.DefinedNames.DefinedName {
			definedName, localSheetID := wb.DefinedNames.DefinedName[idx], 0
			if definedName.LocalSheetID != nil {
				localSheetID = *definedName.LocalSheetID
			}
			if definedName.Name == builtInDefinedNames[3] && localSheetID == sheetID && definedName.Hidden {
				wb.DefinedNames.DefinedName[idx].Data = filterRange
				definedNameExists = true
			}
		}
		if !definedNameExists {
			wb.DefinedNames.DefinedName = append(wb.DefinedNames.DefinedName, d)
		}
	}
	columns := coordinates[2] - coordinates[0]
	return f.autoFilter(sheet, ref, columns, coordinates[0], opts)
}

// autoFilter provides a function to extract the tokens from the filter
// expression. The tokens are mainly non-whitespace groups.
func (f *File) autoFilter(sheet, ref string, columns, col int, opts []AutoFilterOptions) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if ws.SheetPr != nil {
		ws.SheetPr.FilterMode = true
	}
	ws.SheetPr = &xlsxSheetPr{FilterMode: true}
	filter := &xlsxAutoFilter{
		Ref: ref,
	}
	ws.AutoFilter = filter
	for _, opt := range opts {
		if opt.Column == "" || opt.Expression == "" {
			continue
		}
		fsCol, err := ColumnNameToNumber(opt.Column)
		if err != nil {
			return err
		}
		offset := fsCol - col
		if offset < 0 || offset > columns {
			return newInvalidAutoFilterColumnError(opt.Column)
		}
		fc := &xlsxFilterColumn{ColID: offset}
		token := expressionFormat.FindAllString(opt.Expression, -1)
		if len(token) != 3 && len(token) != 7 {
			return newInvalidAutoFilterExpError(opt.Expression)
		}
		expressions, tokens, err := f.parseFilterExpression(opt.Expression, token)
		if err != nil {
			return err
		}
		f.writeAutoFilter(fc, expressions, tokens)
		filter.FilterColumn = append(filter.FilterColumn, fc)
	}
	ws.AutoFilter = filter
	return nil
}

// writeAutoFilter provides a function to check for single or double custom
// filters as default filters and handle them accordingly.
func (f *File) writeAutoFilter(fc *xlsxFilterColumn, exp []int, tokens []string) {
	if len(exp) == 1 && exp[0] == 2 {
		// Single equality.
		var filters []*xlsxFilter
		filters = append(filters, &xlsxFilter{Val: tokens[0]})
		fc.Filters = &xlsxFilters{Filter: filters}
		return
	}
	if len(exp) == 3 && exp[0] == 2 && exp[1] == 1 && exp[2] == 2 {
		// Double equality with "or" operator.
		var filters []*xlsxFilter
		for _, v := range tokens {
			filters = append(filters, &xlsxFilter{Val: v})
		}
		fc.Filters = &xlsxFilters{Filter: filters}
		return
	}
	// Non default custom filter.
	expRel, andRel := map[int]int{0: 0, 1: 2}, map[int]bool{0: true, 1: false}
	for k, v := range tokens {
		f.writeCustomFilter(fc, exp[expRel[k]], v)
		if k == 1 {
			fc.CustomFilters.And = andRel[exp[k]]
		}
	}
}

// writeCustomFilter provides a function to write the <customFilter> element.
func (f *File) writeCustomFilter(fc *xlsxFilterColumn, operator int, val string) {
	operators := map[int]string{
		1:  "lessThan",
		2:  "equal",
		3:  "lessThanOrEqual",
		4:  "greaterThan",
		5:  "notEqual",
		6:  "greaterThanOrEqual",
		22: "equal",
	}
	customFilter := xlsxCustomFilter{
		Operator: operators[operator],
		Val:      val,
	}
	if fc.CustomFilters != nil {
		fc.CustomFilters.CustomFilter = append(fc.CustomFilters.CustomFilter, &customFilter)
		return
	}
	var customFilters []*xlsxCustomFilter
	customFilters = append(customFilters, &customFilter)
	fc.CustomFilters = &xlsxCustomFilters{CustomFilter: customFilters}
}

// parseFilterExpression provides a function to converts the tokens of a
// possibly conditional expression into 1 or 2 sub expressions for further
// parsing.
//
// Examples:
//
//	('x', '==', 2000) -> exp1
//	('x', '>',  2000, 'and', 'x', '<', 5000) -> exp1 and exp2
func (f *File) parseFilterExpression(expression string, tokens []string) ([]int, []string, error) {
	var expressions []int
	var t []string
	if len(tokens) == 7 {
		// The number of tokens will be either 3 (for 1 expression) or 7 (for 2
		// expressions).
		conditional, c := 0, tokens[3]
		if conditionFormat.MatchString(c) {
			conditional = 1
		}
		expression1, token1, err := f.parseFilterTokens(expression, tokens[:3])
		if err != nil {
			return expressions, t, err
		}
		expression2, token2, err := f.parseFilterTokens(expression, tokens[4:7])
		if err != nil {
			return expressions, t, err
		}
		return []int{expression1[0], conditional, expression2[0]}, []string{token1, token2}, nil
	}
	exp, token, err := f.parseFilterTokens(expression, tokens)
	if err != nil {
		return expressions, t, err
	}
	return exp, []string{token}, nil
}

// parseFilterTokens provides a function to parse the 3 tokens of a filter
// expression and return the operator and token.
func (f *File) parseFilterTokens(expression string, tokens []string) ([]int, string, error) {
	operators := map[string]int{
		"==": 2,
		"=":  2,
		"=~": 2,
		"eq": 2,
		"!=": 5,
		"!~": 5,
		"ne": 5,
		"<>": 5,
		"<":  1,
		"<=": 3,
		">":  4,
		">=": 6,
	}
	operator, ok := operators[strings.ToLower(tokens[1])]
	if !ok {
		// Convert the operator from a number to a descriptive string.
		return []int{}, "", newUnknownFilterTokenError(tokens[1])
	}
	token := tokens[2]
	// Special handling for Blanks/NonBlanks.
	re := blankFormat.MatchString(strings.ToLower(token))
	if re {
		// Only allow Equals or NotEqual in this context.
		if operator != 2 && operator != 5 {
			return []int{operator}, token, newInvalidAutoFilterOperatorError(tokens[1], expression)
		}
		token = strings.ToLower(token)
		// The operator should always be 2 (=) to flag a "simple" equality in
		// the binary record. Therefore we convert <> to =.
		if token == "blanks" {
			if operator == 5 {
				token = " "
			}
		} else {
			if operator == 5 {
				operator = 2
				token = "blanks"
			} else {
				operator = 5
				token = " "
			}
		}
	}
	// If the string token contains an Excel match character then change the
	// operator type to indicate a non "simple" equality.
	if re = matchFormat.MatchString(token); operator == 2 && re {
		operator = 22
	}
	return []int{operator}, token, nil
}