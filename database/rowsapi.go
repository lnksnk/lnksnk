package database

import "context"

type RowsAPI interface {
	Close() error
	ColumnTypes(...string) ([]*ColumnType, error)
	Columns(...string) ([]string, error)
	Data(...string) []interface{}
	DisplayData(...string) []interface{}
	Field(string) interface{}
	FieldByIndex(int) interface{}
	FieldIndex(string) int
	Err() error
	Next() bool
	Context() context.Context
	NextResultSet() bool
	Scan(castTypeVal func(valToCast interface{}, colType interface{}) (val interface{}, scanned bool)) error
}

type currows struct {
	crntrw RowsAPI
	rows   []RowsAPI
	lsterr error
}

// Close implements RowsAPI.
func (c *currows) Close() (err error) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	rows := c.rows
	c.crntrw = nil
	c.rows = nil
	if crntrw != nil {
		crntrw.Close()
		crntrw = nil
	}
	for len(rows) > 0 {
		rows[0].Close()
		rows = rows[1:]
	}
	return
}

func (c *currows) nextRows() (nxtrows RowsAPI) {
	if len(c.rows) > 0 {
		nxtrows = c.rows[0]
		c.rows = c.rows[1:]
	}
	return
}

// ColumnTypes implements RowsAPI.
func (c *currows) ColumnTypes(cols ...string) (cltpes []*ColumnType, err error) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		cltpes, err = crntrw.ColumnTypes(cols...)
		c.lsterr = err
	}
	return
}

// Columns implements RowsAPI.
func (c *currows) Columns(cols ...string) (cls []string, err error) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		cls, err = crntrw.Columns(cols...)
		c.lsterr = err
	}
	return
}

// Context implements RowsAPI.
func (c *currows) Context() (ctx context.Context) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		ctx = crntrw.Context()
	}
	return
}

// Data implements RowsAPI.
func (c *currows) Data(cols ...string) (data []interface{}) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		data = crntrw.Data(cols...)
		c.lsterr = crntrw.Err()
	}
	return
}

// DisplayData implements RowsAPI.
func (c *currows) DisplayData(cols ...string) (dspdata []interface{}) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		dspdata = crntrw.DisplayData(cols...)
		c.lsterr = crntrw.Err()
	}
	return
}

// Err implements RowsAPI.
func (c *currows) Err() (err error) {
	if c == nil {
		return
	}
	return c.lsterr
}

// Field implements RowsAPI.
func (c *currows) Field(col string) (val interface{}) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		val = crntrw.Field(col)
	}
	return
}

// FieldByIndex implements RowsAPI.
func (c *currows) FieldByIndex(idx int) (val interface{}) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	if crntrw != nil {
		val = crntrw.FieldByIndex(idx)
	}
	return
}

// FieldIndex implements RowsAPI.
func (c *currows) FieldIndex(col string) (idx int) {
	if c == nil {
		return
	}
	idx = -1
	crntrw := c.crntrw
	if crntrw != nil {
		idx = crntrw.FieldIndex(col)
	}

	return
}

// Next implements RowsAPI.
func (c *currows) Next() (next bool) {
	if c == nil {
		return false
	}
	crntrw := c.crntrw
	if crntrw == nil {
		if crntrw = c.nextRows(); crntrw == nil {
			return false
		}
	}
donext:
	if next = crntrw.Next(); !next {
		c.lsterr = crntrw.Err()
		c.crntrw = c.nextRows()
		if crntrw = c.crntrw; crntrw != nil {
			goto donext
		}
	}
	c.lsterr = nil
	return
}

// NextResultSet implements RowsAPI.
func (c *currows) NextResultSet() bool {
	if c == nil {
		return false
	}
	return false
}

// Scan implements RowsAPI.
func (c *currows) Scan(castTypeVal func(valToCast interface{}, colType interface{}) (val interface{}, scanned bool)) (err error) {
	if c == nil {
		return
	}
	crntrw := c.crntrw
	c.lsterr = nil
	if crntrw != nil {
		err = crntrw.Scan(castTypeVal)
		c.lsterr = err
	}
	return
}
