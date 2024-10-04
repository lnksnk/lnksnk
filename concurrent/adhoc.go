package concurrent

import "reflect"

func constructValue(value interface{}, validkinds ...reflect.Kind) (result interface{}) {
	val, valok := value.(reflect.Value)
	if !valok {
		val = reflect.ValueOf(value)
	}
	kind := val.Kind()
	if len(validkinds) > 0 && kind != reflect.Slice && kind != reflect.Array && kind != reflect.Map {
		for _, vlknd := range validkinds {
			if vlknd == kind {
				goto cntnue
			}
		}
		return
	}
cntnue:
	if kind == reflect.Slice || kind == reflect.Array {
		vals := make([]reflect.Value, val.Len())
		values := make([]interface{}, val.Len())
		for n := range vals {
			values[n] = constructValue(vals[n], validkinds...)
		}
		valslice := NewSlize()
		valslice.Append(values...)
		result = valslice
	} else if kind == reflect.Map {
		keys := val.MapKeys()
		valmp := NewMap()
		for _, k := range keys {
			c_key := k.Convert(val.Type().Key())
			valmp.Set(constructValue(c_key), constructValue(val.MapIndex(c_key)))
		}
		result = valmp
	} else if kind == reflect.Invalid {
		result = nil
	} else {
		result = val.Interface()
	}
	return
}
