package fs

import "github.com/lnksnk/lnksnk/ioext"

type FileSystems interface {
	ioext.IterateMap[string, FileSystem]
}

type filesystems struct {
	ioext.IterateMap[string, FileSystem]
}

// Clear implements FileSystems.
func (f *filesystems) Clear() {
	if f == nil {
		return
	}
	if itr := f.IterateMap; itr != nil {
		itr.Clear()
	}
}

// Close implements FileSystems.
func (f *filesystems) Close() {
	if f == nil {
		return
	}
	itr := f.IterateMap
	f.IterateMap = nil
	if itr != nil {
		itr.Close()
	}
}

// Empty implements FileSystems.
func (f *filesystems) Empty() bool {
	if f == nil {
		return true
	}
	if itr := f.IterateMap; itr != nil {
		return itr.Empty()
	}
	return true
}

// Contains implements FileSystems.
func (f *filesystems) Contains(name string) bool {
	if f == nil {
		return false
	}
	if itr := f.IterateMap; itr != nil {
		return itr.Contains(name)
	}
	return false
}

// Delete implements FileSystems.
func (f *filesystems) Delete(name ...string) {
	if f == nil {
		return
	}
	if itr := f.IterateMap; itr != nil {
		itr.Delete(name...)
	}
}

// Events implements FileSystems.
func (f *filesystems) Events() ioext.IterateMapEvents[string, FileSystem] {
	if f == nil {
		return nil
	}
	if itr := f.IterateMap; itr != nil {
		return itr.Events()
	}
	return nil
}

// Get implements FileSystems.
func (f *filesystems) Get(name string) (value FileSystem, found bool) {
	if f == nil {
		return
	}
	if itr := f.IterateMap; itr != nil {
		return itr.Get(name)
	}
	return
}

// Iterate implements FileSystems.
func (f *filesystems) Iterate() func(func(string, FileSystem) bool) {
	if f == nil {
		return func(f func(string, FileSystem) bool) {
		}
	}
	if itr := f.IterateMap; itr != nil {
		return itr.Iterate()
	}
	return func(f func(string, FileSystem) bool) {
	}
}

// Set implements FileSystems.
func (f *filesystems) Set(name string, value FileSystem) {
	if f == nil {
		return
	}
	if itr := f.IterateMap; itr != nil {
		itr.Set(name, value)
	}
}

func NewFileSystems() FileSystems {
	return &filesystems{IterateMap: ioext.MapIterator[string, FileSystem]()}
}
