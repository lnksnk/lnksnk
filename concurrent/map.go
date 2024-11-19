package concurrent

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lnksnk/lnksnk/reflection"
)

type Map struct {
	elmmp *sync.Map
	cnt   atomic.Int64
}

func NewMap() (enmmp *Map) {
	enmmp = &Map{elmmp: &sync.Map{}}
	return
}

func (enmmp *Map) Count() (cnt int) {
	if enmmp != nil {
		cnt = int(enmmp.cnt.Load())
	}
	return
}

func (enmmp *Map) Exist(key interface{}) (exist bool) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			_, exist = elmmp.Load(key)
		}
	}
	return
}

func (enmmp *Map) Keys(k ...interface{}) (keys []interface{}) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			if kl := len(k); kl > 0 {
				mpk := map[interface{}]bool{}
				ki := 0
				for ki < kl {
					if !mpk[k[ki]] {
						mpk[k[ki]] = true
						continue
					}
					k = append(k[:ki], k[ki+1:]...)
					kl--
				}
				elmmp.Range(func(key, value any) bool {
					if mpk[key] {
						keys = append(keys, key)
						kl--
					}
					return !(kl > 0)
				})
				return
			}
			elmmp.Range(func(key, value any) bool {
				keys = append(keys, key)
				return true
			})
		}
	}
	return
}

func (enmmp *Map) Values(k ...interface{}) (values []interface{}) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			if kl := len(k); kl > 0 {
				mpk := map[interface{}]bool{}
				ki := 0
				for ki < kl {
					if !mpk[k[ki]] {
						mpk[k[ki]] = true
						continue
					}
					k = append(k[:ki], k[ki+1:]...)
					kl--
				}
				elmmp.Range(func(key, value any) bool {
					if mpk[key] {
						values = append(values, value)
						kl--
					}
					return !(kl > 0)
				})
				return
			}
			elmmp.Range(func(key, value any) bool {
				values = append(values, value)
				return true
			})
		}
	}
	return
}

func (enmmp *Map) Del(key ...interface{}) {
	if keysl := len(key); enmmp != nil && keysl > 0 {
		if elmmp := enmmp.elmmp; elmmp != nil {
			var donedel func([]interface{}, []interface{}) = nil
			if donedel, _ = key[keysl-1].(func([]interface{}, []interface{})); donedel != nil {
				keysl--
				key = key[:keysl]
			}
			var delkeys []interface{} = nil
			var delvalues []interface{} = nil
			delkeysi := 0
			elmmp.Range(func(ke, value any) bool {
				for kn, k := range key {
					if k == ke {
						if delval, keyexisted := elmmp.LoadAndDelete(k); keyexisted {
							enmmp.cnt.Add(-1)
							func() {
								if donedel == nil {
									if vm, _ := delval.(*Map); vm != nil {
										vm.Dispose()
										return
									}
									if vsl, _ := delval.(*Slice); vsl != nil {
										vsl.Dispose()
									}
									return
								}
								delkeys = append(delkeys, k)
								delvalues = append(delvalues, delval)
								delvalues[delkeysi] = delval
								delkeysi++
							}()
						}
						key = append(key[:kn], key[kn+1:]...)
						keysl--
						break
					}
				}
				return keysl > 0
			})

			if delkeysi > 0 && donedel != nil {
				defer func() {
					delvalues = nil
					delkeys = nil
				}()
				donedel(delkeys[:delkeysi], delvalues[:delkeysi])
			}
		}
	}
}

func (enmmp *Map) Find(k ...interface{}) (value interface{}, found bool) {
	if len(k) == 1 {
		if ks, _ := k[0].(string); ks != "" && strings.Contains(ks, ",") {
			ksarr := strings.Split(ks, ",")
			k = make([]interface{}, len(ksarr))
			for kn, kv := range ksarr {
				k[kn] = kv
			}
		}
	}
	found = findvalue(enmmp, nil, func(val interface{}) {
		value = val
	}, k...)
	return
}

func findvalue(enmmp *Map, slce *Slice, onfound func(value interface{}), k ...interface{}) (found bool) {
	if enmmp == nil && slce == nil {
		return
	}
	if kl := len(k); onfound != nil && kl > 0 {
		var nextelmp = func(enmp *Map, slc *Slice) *sync.Map {
			if enmp != nil {
				return enmmp.elmmp
			} else if slc != nil {
				return slc.elmmp
			}
			return nil
		}
		var value interface{} = nil
		for kn, key := range k {
			if elmp := nextelmp(enmmp, slce); elmp != nil {
				if value, found = elmp.Load(key); found {
					if kl-1 == kn {
						onfound(value)
						return
					} else if enmmp, found = value.(*Map); found && enmmp != nil {
						slce = nil
						continue
					} else if slce, found = value.(*Slice); found && slce != nil {
						enmmp = nil
						continue
					}
					return false
				}
				return false
			}
			return false
		}
	}
	return
}

func (enmmp *Map) Invoke(key any, method string, a ...interface{}) (result []interface{}) {
	if method != "" {
		if val, valfnd := enmmp.Get(key); valfnd && val != nil {
			_, result = reflection.ReflectCallMethod(val, method, a...)
		}
	}
	return
}

func (enmmp *Map) Field(key any, field string, a ...interface{}) (result []interface{}) {
	if field != "" {
		if val, valfnd := enmmp.Get(key); valfnd && val != nil {
			_, result = reflection.ReflectCallField(val, field, a...)
		}
	}
	return
}

func (enmmp *Map) Get(key interface{}) (value interface{}, loaded bool) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			value, loaded = elmmp.Load(key)
		}
	}
	return
}

func (enmmp *Map) Iter(k ...interface{}) func(func(any, any) bool) {
	return enmmp.Iterate(k...)
}

func (enmmp *Map) Iterate(k ...interface{}) func(func(any, any) bool) {
	if enmmp == nil {
		return func(yield func(key any, val any) bool) {
			if enmmp == nil {
				return
			}
		}
	}
	return iteratemap(enmmp.elmmp, k...)
}

func iteratemap(mp *sync.Map, k ...interface{}) func(func(any, any) bool) {
	return func(yield func(any, any) bool) {
		if mp == nil {
			return
		}
		if kl := len(k); kl > 0 {
			mpks := map[interface{}]bool{}
			ki := 0
			for ki < kl {
				if !mpks[k[ki]] {
					mpks[k[ki]] = true
					ki++
					continue
				}
				//remove duplicate lookup k
				k = append(k[:ki], k[ki+1:])
				kl--
			}
			mp.Range(func(key, value any) bool {
				if mpks[key] {
					kl--
					return !(kl > 0)
				}
				return !(kl > 0)
			})
			return
		}
		mp.Range(func(key, value any) bool {
			return yield(key, value)
		})
	}
}

func (enmmp *Map) Range(f func(interface{}, interface{}) bool, k ...interface{}) {
	if enmmp == nil || f == nil {
		return
	}
	if elmmp := enmmp.elmmp; elmmp != nil {
		for key, value := range iteratemap(elmmp, k...) {
			if f(key, value) {
				break
			}
		}
	}
}

func (enmmp *Map) ForEach(f func(interface{}, interface{}, bool, bool) bool, k ...interface{}) {
	if enmmp != nil && f != nil {
		first := true
		prv := []interface{}{}
		nxt := []interface{}{}
		prfrm := func(key, value any) bool {
			if len(prv) == 0 {
				prv = append(prv, key, value)
				return true
			}
			if len(nxt) == 0 {
				nxt = append(nxt, key, value)
				return true
			}
			if first {
				if f(prv[0], prv[1], first, false) {
					return false
				}
				first = false
				copy(prv, nxt)
				copy(nxt, []interface{}{key, value})
				return true
			}
			if f(prv[0], prv[1], first, false) {
				return false
			}
			first = false
			copy(prv, nxt)
			copy(nxt, []interface{}{key, value})
			return true
		}
		if elmmp := enmmp.elmmp; elmmp != nil {
			if kl := len(k); kl > 0 {
				mpk := map[interface{}]bool{}
				ki := 0
				for ki < kl {
					if !mpk[k[ki]] {
						mpk[k[ki]] = true
						ki++
						continue
					}
					k = append(k[:ki], k[ki+1:]...)
					kl--
				}
				elmmp.Range(func(key, value any) bool {
					if mpk[key] {
						return prfrm(key, value)
					}
					return !(kl > 0)
				})
				goto wrpup
			}
			elmmp.Range(func(key, value any) bool {
				return prfrm(key, value)
			})
		wrpup:
			if len(prv) == 2 {
				if len(nxt) == 2 {
					if f(prv[0], prv[1], first, false) {
						return
					}
					f(nxt[0], nxt[1], false, true)
					return
				}
				f(prv[0], prv[1], true, true)
			}
		}
	}
}

func (enmmp *Map) Set(key interface{}, value interface{}) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			var oldval, loaded = elmmp.Load(key)
			if !loaded {
				enmmp.cnt.Add(1)
			}
			if oldval != value {
				elmmp.Store(key, constructValue(value))
			} else {
				loaded = false
			}
			if loaded {
				if oldval != nil {
					oldval = nil
				}
			}
		}
	}
}

func (enmmp *Map) Dispose() {
	if enmmp != nil {
		if elmp := enmmp.elmmp; elmp != nil {
			enmmp.elmmp = nil

			elmp.Range(func(ke, value any) bool {
				//delkvs[ke] = value
				elmp.Delete(ke)
				if vm, _ := value.(*Map); vm != nil {
					vm.Dispose()
				}
				if vsl, _ := value.(*Slice); vsl != nil {
					vsl.Dispose()
				}
				return false
			})
			elmp = nil
		}
	}
}
