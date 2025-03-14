package types

import (
	"encoding/binary"
	"time"
)

// Win FILETIME type
// http://msdn.microsoft.com/en-us/library/cc230324.aspx
type FileTime struct {
	Low  uint32 // Windows FILETIME structure
	High uint32 // Windows FILETIME structure
}

const (
	tick       uint64 = 10000000
	gregToUnix uint64 = 11644473600
)

func winToUnix(low, high uint32) int64 {
	gregTime := ((uint64(high) << 32) + uint64(low)) / tick
	if gregTime < gregToUnix {
		return 0
	}
	return int64(gregTime - gregToUnix)
}

func (f FileTime) Time() time.Time {
	return time.Unix(winToUnix(f.Low, f.High), 0)
}

func (f FileTime) String() string {
	return f.Time().String()
}

func (f FileTime) Type() string {
	return "FileTime"
}

func (f FileTime) Length() int {
	return 8
}

func MakeFileTime(b []byte) (Type, error) {
	if len(b) < 8 {
		return FileTime{}, ErrType
	}
	return MustFileTime(b), nil
}

func MustFileTime(b []byte) FileTime {
	return FileTime{
		Low:  binary.LittleEndian.Uint32(b[:4]),
		High: binary.LittleEndian.Uint32(b[4:8]),
	}
}
