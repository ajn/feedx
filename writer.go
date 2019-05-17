package feedx

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bsm/bfs"
)

// WriterOptions configure the producer instance.
type WriterOptions struct {
	// Format specifies the format
	// Default: auto-detected from URL path.
	Format Format

	// Compression specifies the compression type.
	// Default: auto-detected from URL path.
	Compression Compression

	// Provides an optional last modified timestamp which is stored with the remote metadata.
	// Default: time.Now().
	LastMod time.Time
}

func (o *WriterOptions) norm(name string) error {
	if o.Format == nil {
		o.Format = DetectFormat(name)

		if o.Format == nil {
			return fmt.Errorf("feedx: unable to detect format from %q", name)
		}
	}

	if o.Compression == nil {
		o.Compression = DetectCompression(name)
	}

	if o.LastMod.IsZero() {
		o.LastMod = time.Now()
	}

	return nil
}

// Writer encodes feeds to remote locations.
type Writer struct {
	ctx    context.Context
	cancel context.CancelFunc

	remote *bfs.Object
	opt    WriterOptions
	num    int

	bw io.WriteCloser // bfs writer
	cw io.WriteCloser // compression writer
	ww *bufio.Writer
	fe FormatEncoder
}

// NewWriter inits a new feed writer.
func NewWriter(ctx context.Context, remote *bfs.Object, opt *WriterOptions) (*Writer, error) {
	var o WriterOptions
	if opt != nil {
		o = *opt
	}
	o.norm(remote.Name())

	ctx, cancel := context.WithCancel(ctx)
	return &Writer{
		ctx:    ctx,
		cancel: cancel,
		remote: remote,
		opt:    o,
	}, nil
}

// Encode appends a value to the feed.
func (w *Writer) Encode(v interface{}) error {
	if w.bw == nil {
		ts := timestampFromTime(w.opt.LastMod)
		bw, err := w.remote.Create(w.ctx, &bfs.WriteOptions{
			Metadata: map[string]string{metaLastModified: ts.String()},
		})
		if err != nil {
			return err
		}
		w.bw = bw
	}

	if w.cw == nil {
		cw, err := w.opt.Compression.NewWriter(w.bw)
		if err != nil {
			return err
		}
		w.cw = cw
	}

	if w.ww == nil {
		w.ww = bufio.NewWriter(w.cw)
	}

	if w.fe == nil {
		fe, err := w.opt.Format.NewEncoder(w.ww)
		if err != nil {
			return err
		}
		w.fe = fe
	}

	if err := w.fe.Encode(v); err != nil {
		return err
	}

	w.num++
	return nil
}

// NumWritten returns the number of written values.
func (w *Writer) NumWritten() int {
	return w.num
}

// Discard closes the writer and discards the contents.
func (w *Writer) Discard() error {
	w.cancel()
	return w.Commit()
}

// Commit closes the writer and persists the contents.
func (w *Writer) Commit() error {
	var err error
	if w.fe != nil {
		if e := w.fe.Close(); e != nil {
			err = e
		}
	}
	if w.ww != nil {
		if e := w.ww.Flush(); e != nil {
			err = e
		}
	}
	if w.cw != nil {
		if e := w.cw.Close(); e != nil {
			err = e
		}
	}
	if w.bw != nil {
		if e := w.bw.Close(); e != nil {
			err = e
		}
	}
	return err
}
