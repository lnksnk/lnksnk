package ioext

import (
	"reflect"
	"runtime"
	"sync"
)

type IterateMapEvents[K comparable, V any] interface {
	Deleted(map[K]V)
	Changed(K, V, V)
	Disposed(map[K]V)
	Add(K, V)
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
	EventAdd      func(K, V)
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

func (mpevents *MapIterateEvents[K, V]) Add(name K, new V) {
	if mpevents == nil {
		return
	}
	if evt := mpevents.EventAdd; evt != nil {
		evt(name, new)
	}
}

func MapIterator[K comparable, V any]() IterateMap[K, V] {

	itrmp := &itermap[K, V]{orgsncmp: &sync.Map{}}

	runtime.SetFinalizer(itrmp, finalizeItermap[K, V])
	return itrmp
}

type itermap[K comparable, V any] struct {
	orgsncmp *sync.Map
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

	var prvval V

	mpevents := imp.mpevents

	if orgsncmp := imp.orgsncmp; orgsncmp != nil {
		if prvv, prvok := orgsncmp.Load(name); prvok {
			prvval, _ = prvv.(V)
			if pval, val := reflect.ValueOf(prvval), reflect.ValueOf(value); !pval.Equal(val) {
				orgsncmp.Store(name, value)
				if mpevents != nil {
					go mpevents.Changed(name, prvval, value)
				}
			}
			return
		}
		orgsncmp.Store(name, value)
		if mpevents != nil {
			go mpevents.Add(name, value)
		}
	}
}

func (imp *itermap[K, V]) Events() (events IterateMapEvents[K, V]) {
	if imp == nil {
		return nil
	}
	if events = imp.mpevents; events == nil {
		imp.mpevents = &MapIterateEvents[K, V]{}
		events = imp.mpevents
	}
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
	runtime.SetFinalizer(imp, nil)
}

func init() {
}
