package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/mimes"
)

var lklzpexts = map[string]bool{".zip": true, ".tgz": true, ".gz": true, ".tar": true, ".jar": true, ".war": true}

func ArchiveFiles(archivepath string, rmngroot string) (archfiles []*ArchiveFile, ziperr error) {
	arcext := filepath.Ext(archivepath)
	rnmgrtl := len(rmngroot)
	if arcext == ".zip" {
		zpr, zprerr := zip.OpenReader(archivepath)
		if zprerr != nil {
			return
		}
		defer zpr.Close()
		ziparchdirpath := archivepath[strings.LastIndex(archivepath, "/")+1:]
		ziparchdirpath = ziparchdirpath[:len(ziparchdirpath)-len(arcext)]
		for _, zpf := range zpr.File {
			zpfh := zpf.FileHeader
			if ziparchdirpath != "" {
				if zpfh.Name == ziparchdirpath+"/" {
					if zpfh.FileInfo().IsDir() {
						if rmngroot == "" {
							rmngroot = "/" + ziparchdirpath
						} else {
							rmngroot = "/" + ziparchdirpath + rmngroot
						}
						rnmgrtl = len(rmngroot)
					}
					ziparchdirpath = ""
					continue
				}
				ziparchdirpath = ""
			}
			if zpfnmel := len(zpfh.Name); rmngroot == "" || (zpfnmel >= rnmgrtl && "/"+zpfh.Name[:rnmgrtl-1] == rmngroot) {
				lkppath := zpfh.Name
				lkproot := ""
				if spi := strings.LastIndex(lkppath, "/"); spi > -1 {
					lkproot = lkppath[:spi+1]
				}
				zfi := zpfh.FileInfo()
				zpname := zpfh.Name[rnmgrtl:]
				zproot := ""
				if zfi.IsDir() {
					if zpname != "" {
						zproot = zpname[:len(zpname)-len(zfi.Name()+"/")]
						zpname = zfi.Name()
					} else {
						continue
					}
				} else {
					zproot = zpname[:len(zpname)-len(zfi.Name())]
				}
				if zproot == "" {
					zproot = "/"
				} else if !zfi.IsDir() && strings.Contains(zproot, "/") {
					zproot = zproot[:strings.LastIndex(zproot, "/")+1]
					zpname = zpname[len(zproot):]
				}
				if zproot != "" && zproot[0] != '/' {
					zproot = "/" + zproot
				}
				if zfi.IsDir() {
					zpname += "/"
				}
				archf := &ArchiveFile{FileInfo: zfi, lkppath: lkppath, lkproot: lkproot, archivepath: archivepath, name: zpname, root: zproot, path: zproot + zpname}
				if !archf.IsDir() {
					archf.prepBuf = func() (err error) {
						if archf == nil {
							return
						}
						if zr, _ := zip.OpenReader(archf.archivepath); zr != nil {
							defer zr.Close()
							lkprtl := len(archf.lkproot)
							files := zr.File
							fn := len(files)
							for tfn, tzf := range files {
								if nml := len(tzf.FileHeader.Name); nml >= lkprtl && tzf.FileHeader.Name[:lkprtl] == archf.lkproot && tzf.FileHeader.Name == archf.lkppath {
									zf, zferr := tzf.Open()
									if zferr == nil {
										defer zf.Close()
										if archf.bf, zferr = ioext.NewBufferError(zf); zferr != nil {
											return zferr
										}
										return
									}
									return zferr
								}
								if fn > tfn {
									fn--
									tzf = files[fn]
									if nml := len(tzf.FileHeader.Name); nml >= lkprtl && tzf.FileHeader.Name[:lkprtl] == archf.lkproot && tzf.FileHeader.Name == archf.lkppath {
										if zf, zferr := tzf.Open(); zferr == nil {
											defer zf.Close()
											if archf.bf, zferr = ioext.NewBufferError(zf); zferr != nil {
												return zferr
											}
											return
										}
									}
									continue
								}
								break
							}
						}
						return
					}
				}
				archfiles = append(archfiles, archf)
			}
		}
		return
	}
	if arcext == ".tar" || arcext == ".gz" || arcext == ".tgz" {
		if f, ferr := os.Open(archivepath); ferr == nil {
			defer f.Close()
			var tarr *tar.Reader
			var gzr *gzip.Reader
			if arcext == ".tgz" || arcext == ".gz" {
				if arcext == ".gz" {
					tstarchivepath := archivepath[strings.LastIndex(archivepath, "/")+1:]
					if subarchext := filepath.Ext(tstarchivepath[:len(tstarchivepath)-len(arcext)]); subarchext == ".tar" {
						rmngroot = "/" + tstarchivepath[:len(tstarchivepath)-len(arcext)-len(subarchext)] + rmngroot
						rnmgrtl = len(rmngroot)
					}
				}
				gzrerr := error(nil)
				gzr, gzrerr = gzip.NewReader(f)
				if gzrerr != nil {
					return
				}
				if gzr != nil {
					defer gzr.Close()
					tarr = tar.NewReader(gzr)
				}
			}
			if arcext == ".tar" {
				tarr = tar.NewReader(f)
			}
			if tarr == nil {
				return
			}
			for {
				trhead, trerr := tarr.Next()
				if trerr == io.EOF {
					break
				} else if trerr != nil {
					break
				}
				if trhead != nil {
					switch trhead.Typeflag {
					case tar.TypeReg, tar.TypeDir:
						if trfnmel := len(trhead.Name); rmngroot == "" || (trfnmel >= rnmgrtl && "/"+trhead.Name[:rnmgrtl-1] == rmngroot) {
							lkppath := trhead.Name
							lkproot := ""
							if spi := strings.LastIndex(lkppath, "/"); spi > -1 {
								lkproot = lkppath[:spi+1]
							}
							trfi := trhead.FileInfo()
							trname := trhead.Name[rnmgrtl:]
							trroot := ""
							if trfi.IsDir() {
								if trname != "" {
									trroot = trname[:len(trname)-len(trfi.Name()+"/")]
									trname = trfi.Name()
								} else {
									continue
								}
							} else {
								trroot = trname[:len(trname)-len(trfi.Name())]
							}
							if trroot == "" {
								trroot = "/"
							} else if !trfi.IsDir() && strings.Contains(trroot, "/") {
								trroot = trroot[:strings.LastIndex(trroot, "/")+1]
								trname = trname[len(trroot):]
							}
							if trroot != "" && trroot[0] != '/' {
								trroot = "/" + trroot
							}
							if trfi.IsDir() {
								trname += "/"
							}
							archf := &ArchiveFile{FileInfo: trfi, lkppath: lkppath, lkproot: lkproot, archivepath: archivepath, name: trname, root: trroot, path: trroot + trname}
							if !archf.IsDir() {
								archf.prepBuf = func() (err error) {
									if archf == nil {
										return
									}
									if f, ferr := os.Open(archf.archivepath); ferr == nil {
										defer f.Close()
										extrctext := filepath.Ext(archf.archivepath)
										var tarr *tar.Reader
										var gzr *gzip.Reader
										if extrctext == ".gz" || extrctext == ".tgz" {
											gzrerr := error(nil)
											gzr, gzrerr = gzip.NewReader(f)
											if gzrerr != nil {
												return
											}
											if gzr != nil {
												defer gzr.Close()
												tarr = tar.NewReader(gzr)
											}
										}
										if extrctext == ".tar" {
											tarr = tar.NewReader(f)
										}
										if tarr == nil {
											return
										}
										lkprtl := len(archf.lkproot)
										for {
											trhead, trerr := tarr.Next()
											if trerr == io.EOF {
												break
											} else if trerr != nil {
												break
											}

											if trhead != nil {
												switch trhead.Typeflag {
												case tar.TypeReg:
													if nml := len(trhead.Name); nml >= lkprtl && trhead.Name[:lkprtl] == archf.lkproot && trhead.Name == archf.lkppath {
														if archf.bf, trerr = ioext.NewBufferError(ioext.ReadFunc(tarr.Read)); trerr != nil {
															return trerr
														}
														return
													}
												}
											}
										}
									}
									return
								}
							}
							archfiles = append(archfiles, archf)
						}
					}
				}
			}
			return
		}
	}
	return
}

func archiveFileInfo(archfi *ArchiveFile, base string, activexts map[string]bool) (fi FileInfo) {
	if archfi == nil {
		return
	}
	if fi = archfi.finfo; fi == nil {
		ext := filepath.Ext(archfi.name)
		media := false
		if ext != "" && !archfi.IsDir() {
			_, _, media = mimes.FindMimeType(ext)
		} else {
			ext = ""
		}
		atv := ext != "" && len(activexts) > 0 && activexts[ext]
		archpath := archfi.Path()
		if archpath != "" && archpath[0] == '/' {
			archpath = archpath[1:]
		}
		fi = NewFileInfo(archfi.name, archfi.Size(), archfi.Mode(), archfi.ModTime(), archfi.IsDir(), archfi.Sys(), atv, !atv, media, archpath, base, archfi.Reader)
		archfi.finfo = fi
		return
	}
	return
}

type ArchiveFile struct {
	fs.FileInfo
	finfo       FileInfo
	archivepath string
	lkppath     string
	lkproot     string
	bf          *ioext.Buffer
	name        string
	root        string
	path        string
	prepBuf     func() error
}

func (arcf *ArchiveFile) Name() string {
	if arcf == nil {
		return ""
	}
	return arcf.name
}

func (arcf *ArchiveFile) Root() string {
	if arcf == nil {
		return ""
	}
	return arcf.root
}

func (arcf *ArchiveFile) Path() string {
	if arcf == nil {
		return ""
	}
	return arcf.path
}

func (arcf *ArchiveFile) Reader(ctx ...context.Context) io.Reader {
	if arcf == nil {
		return nil
	}
	if bf := arcf.bf; bf != nil {
		if len(ctx) > 0 {
			return bf.Reader(ctx[0])
		}
		return bf.Reader()
	}
	if prepBuf := arcf.prepBuf; prepBuf != nil {
		if prperr := prepBuf(); prperr == nil && arcf.bf != nil {
			if len(ctx) > 0 {
				return arcf.bf.Reader(ctx[0])
			}
			return arcf.bf.Reader()
		}
	}
	return nil
}
