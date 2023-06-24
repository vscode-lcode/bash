package webdav

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/alessio/shellescape"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
	"golang.org/x/net/webdav"
)

type File struct {
	bash.Session

	name string

	cursor int64
	init   *sync.Once
	err    error
	conn   io.ReadWriteCloser
}

var _ webdav.File = (*File)(nil)

func OpenFile(sess bash.Session, name string) webdav.File {
	return &File{
		Session: sess,
		name:    name,
		cursor:  0,
		init:    &sync.Once{},
	}
}

func (f *File) Close() error {
	f.cursor = 0
	f.init, f.err = &sync.Once{}, nil
	if conn := f.conn; conn != nil {
		time.AfterFunc(10*time.Second, func() {
			conn.Close()
		})
	}
	return nil
}
func (f *File) Read(p []byte) (n int, err error) {
	f.init.Do(func() {
		cmd := fmt.Sprintf("dd if=%s skip=%d", shellescape.Quote(f.name), f.cursor)
		cmd = fmt.Sprintf("%s %s", cmd, "iflag=skip_bytes")
		f.conn, f.err = f.Start(cmd)
	})
	if f.err != nil {
		return 0, f.err
	}
	n, err = f.conn.Read(p)
	f.cursor += int64(n)
	return
}
func (f *File) Write(p []byte) (n int, err error) {
	f.init.Do(func() {
		cmd := fmt.Sprintf("dd of=%s seek=%d", shellescape.Quote(f.name), f.cursor)
		cmd = fmt.Sprintf("%s %s", cmd, "oflag=seek_bytes")
		f.conn, f.err = f.Start(cmd)
	})
	if f.err != nil {
		return 0, f.err
	}
	n, err = f.conn.Write(p)
	f.cursor += int64(n)
	return
}
func (f *File) Seek(offset int64, whence int) (n int64, err error) {
	defer err2.Handle(&err)
	switch whence {
	case io.SeekStart:
		f.cursor = offset
	case io.SeekCurrent:
		f.cursor += offset
	case io.SeekEnd:
		stat := try.To1(f.Stat())
		f.cursor = stat.Size() + offset
	}
	n = f.cursor
	try.To(f.Close())
	return
}
