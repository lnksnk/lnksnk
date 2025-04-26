package fieldmapping

import (
	"reflect"
	"strings"
)

type FieldNameMapper interface {
	// FieldName returns a JavaScript name for the given struct field in the given type.
	// If this method returns "" the field becomes hidden.
	FieldName(t reflect.Type, f reflect.StructField) string

	// MethodName returns a JavaScript name for the given method in the given type.
	// If this method returns "" the method becomes hidden.
	MethodName(t reflect.Type, m reflect.Method) string
}

type FieldMapper struct {
	fldmppr FieldNameMapper
}

func NewFieldMapper(fldmppr FieldNameMapper) *FieldMapper {
	return &FieldMapper{fldmppr: fldmppr}
}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (fldmppr *FieldMapper) FieldName(t reflect.Type, f reflect.StructField) (fldnme string) {
	if f.Tag != "" {
		fldnme = f.Tag.Get("json")
	} else {
		fldnme = uncapitalize(f.Name) // fldmppr.fldmppr.FieldName(t, f)
	}
	return
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

// MethodName returns a JavaScript name for the given method in the given type.
// If this method returns "" the method becomes hidden.
func (fldmppr *FieldMapper) MethodName(t reflect.Type, m reflect.Method) (mthdnme string) {
	mthdnme = flduncapitalize(m.Name)
	return
}

func flduncapitalize(s string) (nme string) {
	if sl := len(s); sl > 0 {
		var nrxtsr = rune(0)
		for sn := range s {
			sr := s[sn]
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
				nme += string(sr)
			} else {
				nme += string(sr)
			}
			if sn <= (sl-1)-1 {
				nrxtsr = rune(s[sn+1])
			} else {
				nrxtsr = rune(0)
			}
			if 'a' <= nrxtsr && nrxtsr <= 'z' {
				nme += s[sn+1:]
				break
			}
		}
	}
	return nme
}
