package serveio

type ReaderWriter interface {
	Reader() Reader
	Writer() Writer
	Close() error
}

type readerwriter struct {
	rdr Reader
	wtr Writer
}

func (rdrwtr *readerwriter) Reader() Reader {
	if rdrwtr == nil {
		return nil
	}
	return rdrwtr.rdr
}

func (rdrwtr *readerwriter) Writer() Writer {
	if rdrwtr == nil {
		return nil
	}
	return rdrwtr.wtr
}

func (rdrwtr *readerwriter) Close() error {
	if rdrwtr == nil {
		return nil
	}
	rdr := rdrwtr.rdr
	rdrwtr.rdr = nil
	wtr := rdrwtr.wtr
	rdrwtr.wtr = nil
	if rdr != nil {
		rdr.Close()
	}
	if wtr != nil {
		wtr.Close()
	}
	return nil
}

func NewReaderWriter(rdr Reader, wtr Writer) ReaderWriter {
	return &readerwriter{rdr: rdr, wtr: wtr}
}
