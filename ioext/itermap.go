package ioext

import (
	"runtime"
	"sync"
)

type IterateMapEvents[K comparable, V any] interface {
	Deleted(map[K]V)
	Changed(K, V, V)
	Disposed(map[K]V)
}

type IterateMap[K comparable, V any] interface {
	Get(name K) (value V, found bool)
	Set(name K, value V)
	Delete(name ...K)
	Clear()
	Close()
	Contains(name K) bool
	Iterate() func(func(K, V) bool)
	Events() IterateMapEvents[K, V]
}

type MapIterateEvents[K comparable, V any] struct {
	EventDeleted  func(map[K]V)
	EventDisposed func(map[K]V)
	EventChanged  func(K, V, V)
}

func (mpevents *MapIterateEvents[K, V]) Disposed(mp map[K]V) {
	if mpevents == nil {
		return
	}
	if evt := mpevents.EventDisposed; evt != nil {
		evt(mp)
	}
}

func (mpevents *MapIterateEvents[K, V]) Deleted(mp map[K]V) {
	if mpevents == nil {
		return
	}
	if evt := mpevents.EventDeleted; evt != nil {
		evt(mp)
	}
}

func (mpevents *MapIterateEvents[K, V]) Changed(name K, old, new V) {
	if mpevents == nil {
		return
	}
	if evt := mpevents.EventChanged; evt != nil {
		evt(name, old, new)
	}
}

func MapIterator[K comparable, V any](a ...any) IterateMap[K, V] {
	var orgmap map[K]V
	intmp := false
	var mpevents IterateMapEvents[K, V]
	for d := range a {
		if orgmapd, dok := interface{}(d).(map[K]V); dok {
			if dok {
				if orgmap == nil {
					orgmap = orgmapd
				}
				continue
			}
		}
		if mpeventsd, dok := interface{}(d).(IterateMapEvents[K, V]); dok {
			if mpevents == nil {
				mpevents = mpeventsd
			}
			continue
		}
	}
	if orgmap == nil {
		orgmap = map[K]V{}
		intmp = true
	}
	itrmp := &itermap[K, V]{orgsncmp: &sync.Map{}, dpsemp: intmp, mpevents: mpevents}

	runtime.SetFinalizer(itrmp, finalizeItermap[K, V])
	return itrmp
}

type itermap[K comparable, V any] struct {
	//orgmp    map[K]V
	//lck      sync.Mutex
	orgsncmp *sync.Map
	dpsemp   bool
	mpevents IterateMapEvents[K, V]
}

func finalizeItermap[K comparable, V any](imp *itermap[K, V]) {
	imp.Close()
}

// Contains implements IterateMap.
func (imp *itermap[K, V]) Contains(name K) bool {
	if imp == nil {
		return false
	}

	/*imp.lck.Lock()
	orgmp := imp.orgmp
	if orgmp == nil {
		imp.lck.Unlock()
		return false
	}
	_, ok := orgmp[name]
	imp.lck.Unlock()*/
	_, ok := imp.orgsncmp.Load(name)
	return ok
}

// Clear implements IterateMap.
func (imp *itermap[K, V]) Clear() {
	if imp == nil {
		return
	}

	if orgsncmp := imp.orgsncmp; orgsncmp != nil {
		var tmp map[K]V
		var evtdeleted func(map[K]V)
		if mpevents := imp.mpevents; mpevents != nil {
			tmp = map[K]V{}
			evtdeleted = mpevents.Deleted
		}
		orgsncmp.Range(func(key, value any) bool {
			if evtdeleted != nil {
				tmp[key.(K)] = value.(V)
			}
			orgsncmp.Delete(key)
			return true
		})
		if len(tmp) > 0 && evtdeleted != nil {
			go evtdeleted(tmp)
		}
	}

	/*var tmp map[K]V
	var evtdeleted func(map[K]V)
	imp.lck.Lock()
	orgmp := imp.orgmp
	if mpevents := imp.mpevents; mpevents != nil {
		tmp = map[K]V{}
		evtdeleted = mpevents.Deleted
	}

	for k, v := range orgmp {
		if evtdeleted != nil {
			tmp[k] = v
		}
		delete(orgmp, k)
	}
	imp.lck.Unlock()
	if len(tmp) > 0 && evtdeleted != nil {
		evtdeleted(tmp)
	}*/
}

// Delete implements IterateMap.
func (imp *itermap[K, V]) Delete(name ...K) {
	nml := len(name)
	if imp != nil && nml > 0 {
		nmei := 0
		for nmei < nml {
			if nms, nmsok := interface{}(name[nmei]).(string); nmsok {
				if nms == "" {
					name = append(name[:nmei], name[nmei+1:]...)
					nml--
					continue
				}
			}
			nmei++
		}
	}
	if nml == 0 || imp == nil {
		return
	}

	if orgsncmp := imp.orgsncmp; orgsncmp != nil {
		nms := map[K]bool{}
		for _, nme := range name {
			if !nms[nme] {
				nms[nme] = true
			}
		}
		var tmp map[K]V
		var evtdeleted func(map[K]V)
		if mpevents := imp.mpevents; mpevents != nil {
			tmp = map[K]V{}
			evtdeleted = mpevents.Deleted
		}
		orgsncmp.Range(func(key, value any) bool {
			if nms[key.(K)] {
				if evtdeleted != nil {
					tmp[key.(K)] = value.(V)
				}
				orgsncmp.Delete(key)
			}
			return true
		})
		if len(tmp) > 0 && evtdeleted != nil {
			go evtdeleted(tmp)
		}
	}

	/*var tmp map[K]V
	var evtdeleted func(map[K]V)
	imp.lck.Lock()
	orgmp := imp.orgmp
	if orgmp == nil {
		imp.lck.Unlock()
		return
	}
	if mpevents := imp.mpevents; mpevents != nil {
		tmp = map[K]V{}
		evtdeleted = mpevents.Deleted
	}
	for _, k := range name {
		if evtdeleted != nil {
			tmp[k] = orgmp[k]
		}
		delete(orgmp, k)
	}
	imp.lck.Unlock()
	if evtdeleted != nil {
		go evtdeleted(tmp)
	}*/
}

// Get implements IterateMap.
func (imp *itermap[K, V]) Get(name K) (value V, found bool) {
	if nms, nmsok := interface{}(name).(string); nmsok && nms == "" {
		return
	}
	if imp == nil {
		return
	}
	if orgsncmp := imp.orgsncmp; orgsncmp != nil {
		if v, vk := orgsncmp.Load(name); vk {
			found = vk
			value, _ = v.(V)
		}
	}
	/*if imp == nil || imp.orgmp == nil {
		return
	}

	imp.lck.Lock()
	orgmp := imp.orgmp
	if orgmp == nil {
		imp.lck.Unlock()
		return
	}
	value, found = imp.orgmp[name]
	imp.lck.Unlock()*/
	return
}

// Set implements IterateMap.
func (imp *itermap[K, V]) Set(name K, value V) {
	if nms, nmsok := interface{}(name).(string); nmsok && nms == "" {
		return
	}
	if imp == nil {
		return
	}
	var evtchngd func(K, V, V)
	var prvval V

	if mpevents := imp.mpevents; mpevents != nil {
		evtchngd = mpevents.Changed
	}

	if orgsncmp := imp.orgsncmp; orgsncmp != nil {
		if prvv, prvok := orgsncmp.Load(name); prvok {
			prvval, _ = prvv.(V)
		}
		orgsncmp.Store(name, value)
		if evtchngd != nil {
			go evtchngd(name, prvval, value)
		}
	}

	/*imp.lck.Lock()
	orgmp := imp.orgmp
	if mpevents := imp.mpevents; mpevents != nil {
		evtchngd = mpevents.Changed
	}
	if orgmp == nil {
		imp.orgmp = map[K]V{name: value}
		imp.dpsemp = true
		imp.lck.Unlock()
		if evtchngd != nil {
			evtchngd(name, prvval, value)
		}
		return
	}
	prvval = orgmp[name]
	orgmp[name] = value
	imp.lck.Unlock()
	if evtchngd != nil {
		go evtchngd(name, prvval, value)
	}*/
}

func (imp *itermap[K, V]) Events() (events IterateMapEvents[K, V]) {
	if imp == nil {
		return nil
	}
	//imp.lck.Lock()
	if events = imp.mpevents; events == nil {
		imp.mpevents = &MapIterateEvents[K, V]{}
		events = imp.mpevents
	}
	//imp.lck.Unlock()
	return
}

func (imp *itermap[K, V]) Iterate() func(yield func(key K, value V) bool) {
	return func(yield func(key K, value V) bool) {
		if imp == nil {
			return
		}
		orgsncmp := imp.orgsncmp
		if orgsncmp == nil {
			return
		}
		orgsncmp.Range(func(key, value any) bool {
			return yield(key.(K), value.(V))
		})
		/*tmp := map[K]V{}
		imp.lck.Lock()
		orgmp := imp.orgmp
		if orgmp == nil {
			imp.lck.Unlock()
			return
		}
		for k, v := range orgmp {
			tmp[k] = v
		}
		imp.lck.Unlock()
		for k, v := range tmp {
			if !yield(k, v) {
				return
			}
		}*/
	}
}

func (imp *itermap[K, V]) DisposeMap(mp map[K]V) {
	for k := range mp {
		delete(mp, k)
	}
}

func (imp *itermap[K, V]) Close() {
	if imp == nil {
		return
	}

	orgsncmp := imp.orgsncmp
	mpevents := imp.mpevents
	imp.mpevents = nil
	imp.orgsncmp = nil

	if imp.dpsemp {
		imp.dpsemp = false
		if orgsncmp != nil {
			var evtdispose func(map[K]V)
			var tmp map[K]V
			if mpevents != nil {
				tmp = map[K]V{}
				evtdispose = mpevents.Disposed
			}
			orgsncmp.Range(func(key, value any) bool {
				if evtdispose != nil {
					tmp[key.(K)] = value.(V)
				}
				orgsncmp.Delete(key)
				return true
			})
			if len(tmp) > 0 && evtdispose != nil {
				go evtdispose(tmp)
			}
		}
	}
	/*var evtdispose func(map[K]V)
	var tmp map[K]V
	imp.lck.Lock()
	orgmp := imp.orgmp
	mpevents := imp.mpevents
	imp.mpevents = nil
	imp.orgmp = nil
	if imp.dpsemp {
		imp.dpsemp = false
		if orgmp != nil {
			if mpevents != nil {
				if len(orgmp) > 0 {
					tmp = make(map[K]V)
					for k, v := range orgmp {
						tmp[k] = v
					}
					evtdispose = mpevents.Disposed
				}
			}
			imp.DisposeMap(orgmp)
		}
	}
	imp.lck.Unlock()
	if evtdispose != nil {
		go evtdispose(tmp)
	}*/
	runtime.SetFinalizer(imp, nil)
}

func init() {
}
