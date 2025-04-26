package fs

import (
	gofs "io/fs"
)

type IOFS interface {
	Open(name string) (gofs.File, error)
	ReadDir(name string) ([]gofs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type IOFSReadFile interface {
	ReadFile(name string) ([]byte, error)
}

type IOFSReadFileFunc func(string) ([]byte, error)

func (iofsreadfile IOFSReadFileFunc) ReadFile(name string) ([]byte, error) {
	return iofsreadfile(name)
}

type IOFSReadDir interface {
	ReadDir(name string) ([]gofs.DirEntry, error)
}

type IOFSReadDirFunc func(string) ([]gofs.DirEntry, error)

func (iofsreaddir IOFSReadDirFunc) ReadDir(name string) ([]gofs.DirEntry, error) {
	return iofsreaddir(name)
}

type IOFSOpen interface {
	Open(string) (gofs.File, error)
}

type IOFSOpenFunc func(string) (gofs.File, error)

func (iofsopen IOFSOpenFunc) Open(name string) (gofs.File, error) {
	return iofsopen(name)
}
