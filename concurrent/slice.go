package concurrent

import (
	"sort"
	"strings"

	"github.com/lnksnk/lnksnk/reflection"
)

type Slice struct {
	*Map
}

func NewSlice() (slce *Slice) {
	slce = &Slice{Map: NewMap()}
	return
}

func (slce *Slice) Append(a ...interface{}) {
	if al := len(a); slce != nil && al > 0 {
		if mp := slce.Map; mp != nil {
			adjustby := mp.Count()
			for an := range a {
				mp.Set(adjustby+an, constructValue(a[an]))
			}
		}
	}
}

func (slce *Slice) Del(index ...int) {
	if ixl := len(index); ixl > 0 {
		ix := 0
		if cnt := slce.Map.Count(); cnt >= ixl {
			for ix < ixl {
				if index[ix] < 0 || index[ix] >= cnt {
					index = append(index[:ix], index[ix+1:]...)
					ixl--
					continue
				}
				ix++
			}
			if ixl > 0 {
				sort.Slice(index, func(i, j int) bool { return index[i] < index[j] })
				a := make([]interface{}, ixl+1)
				for ixn, ix := range index {
					a[ixn] = ix
				}

				a[ixl] = func(keys []int, vals []interface{}) {
					cntdiff := cnt - slce.Map.Count()
					cnt = cnt - cntdiff
					keysl := len(keys)
					ki := 0
					for ; ki < keys[0]; ki++ {
					}
					adjstk := 0
					for keyi := 0; keyi < keysl; keyi++ {
						if dk := keys[keyi]; ki+adjstk == dk {
							if keyi < keysl-1 {
								adjstk++
								for ki+adjstk-1 < keys[keyi+1]-1 {
									if nv, nvok := slce.Map.Get(ki + adjstk); nvok {
										slce.Map.Set(ki, nv)
										ki++
									}
								}
							} else {
								adjstk++
								if nv, nvok := slce.Map.Get(ki + adjstk); nvok {
									slce.Map.Set(ki, nv)
									ki++
								}
								if adjstk == cntdiff {
									for ; ki < cnt+cntdiff; ki++ {
										if ki+cntdiff-1 < cnt+cntdiff-1 {
											if nv, nvok := slce.Map.Get(ki + cntdiff); nvok {
												slce.Map.Set(ki, nv)
											}
										} else {
											slce.Map.Del(ki)
										}
									}
								}
							}
						} else {
							break
						}
					}
				}
				slce.Map.Del(a...)
			}
		}
	}
}

func (slce *Slice) Get(index int) (value interface{}) {
	if slce != nil && index >= 0 {
		if mp := slce.Map; mp != nil {
			if val, valok := mp.Get(index); valok {
				value = val
			}
		}
	}
	return
}

func (slce *Slice) Iter(index ...int) func(func(int, any) bool) {
	return slce.Iterate(index...)
}

func (slce *Slice) Iterate(index ...int) func(func(int, interface{}) bool) {
	return func(yield func(idx int, val interface{}) bool) {
		if slce == nil {
			return
		}
		il := len(index)
		if mp := slce.Map; mp != nil {
			var ks []interface{}
			if il > 0 {
				ks = make([]interface{}, il)
				for in, ix := range index {
					ks[in] = ix
				}
			}
			for ix, iv := range mp.Iterate(ks...) {
				if !yield(ix.(int), iv) {
					break
				}
			}
		}
	}
}

func (slce *Slice) Indexes(index ...int) (idxs []int) {
	il := len(index)
	if slce != nil {
		if mp := slce.Map; mp != nil {
			var ks []interface{}
			if il > 0 {
				ks = make([]interface{}, il)
			}
			for in, ix := range index {
				ks[in] = ix
			}
			for _, k := range mp.Keys(ks...) {
				idxs = append(idxs, k.(int))
			}
		}
	}
	return
}
func (slce *Slice) Values(index ...int) (vals []interface{}) {
	il := len(index)
	if slce != nil {
		if mp := slce.Map; mp != nil {
			var ks []interface{}
			if il > 0 {
				ks = make([]interface{}, il)
				for in, ix := range index {
					ks[in] = ix
				}
			}
			vals = slce.Map.Values(ks...)
		}
	}
	return
}

func (slce *Slice) Invoke(index int, method string, a ...interface{}) (result []interface{}) {
	if method != "" {
		if val := slce.Get(index); val != nil {
			_, result = reflection.ReflectCallMethod(val, method, a...)
		}
	}
	return
}

func (slce *Slice) Field(index int, field string, a ...interface{}) (result []interface{}) {
	if field != "" {
		if val := slce.Get(index); val != nil {
			_, result = reflection.ReflectCallField(val, field, a...)
		}
	}
	return
}

func (slce *Slice) Find(k ...interface{}) (value interface{}, found bool) {
	if len(k) == 1 {
		if ks, _ := k[0].(string); ks != "" && strings.Contains(ks, ",") {
			ksarr := strings.Split(ks, ",")
			k = make([]interface{}, len(ksarr))
			for kn, kv := range ksarr {
				k[kn] = kv
			}
		}
	}
	found = findvalue(nil, slce, func(val interface{}) {
		value = val
	}, k...)
	return
}

func (slce *Slice) Dispose() {
	if slce != nil {
		if mp := slce.Map; mp != nil {
			slce.Map = nil
			vals := []interface{}{}
			for _, v := range mp.Iterate() {
				if vmp, _ := v.(*Map); vmp != nil {
					vals = append(vals, vmp)
				} else if vslce, _ := v.(*Slice); v != nil {
					vals = append(vals, vslce)
				}
			}
			for _, v := range vals {
				if vmp, _ := v.(*Map); vmp != nil {
					vmp.Dispose()
				} else if vslce, _ := v.(*Slice); v != nil {
					vslce.Dispose()
				}
			}
			mp.Dispose()
			mp = nil
		}
		slce = nil
	}
}

func (slce *Slice) ForEach(eachitem func(interface{}, int, bool, bool) bool, index ...int) {
	if slce != nil && eachitem != nil {
		if mp := slce.Map; mp != nil {
			var kidx []interface{}
			if idxl := len(index); idxl > 0 {
				kidx = make([]interface{}, idxl)
				for idxn, idx := range index {
					kidx[idxn] = idx
				}
			}
			mp.ForEach(func(k, v interface{}, first, last bool) bool {
				return eachitem(v, k.(int), first, last)
			}, kidx...)

		}
	}
}
