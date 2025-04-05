package dbms

import (
	"database/sql"
	"reflect"
)

type ColumnType interface {
	DatabaseTypeName() string
	Nullable() (nullable, ok bool)
	ScanType() reflect.Type
	DecimalSize() (precision, scale int64, ok bool)
	Length() (length int64, ok bool)
	Name() string
}

type columntype struct {
	dbcoltype *sql.ColumnType
}

func (coltpe *columntype) Name() string {
	return coltpe.dbcoltype.Name()
}

func (coltpe *columntype) DatabaseTypeName() string {
	return coltpe.dbcoltype.DatabaseTypeName()
}

func (coltpe *columntype) Nullable() (nullable, ok bool) {
	return coltpe.dbcoltype.Nullable()
}

func (coltpe *columntype) ScanType() reflect.Type {
	return coltpe.dbcoltype.ScanType()
}

func (coltpe *columntype) DecimalSize() (precision, scale int64, ok bool) {
	return coltpe.dbcoltype.DecimalSize()
}

func (coltpe *columntype) Length() (length int64, ok bool) {
	return coltpe.dbcoltype.Length()
}
