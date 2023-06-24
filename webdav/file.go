package webdav

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"

	"github.com/alessio/shellescape"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

func (f *File) Readdir(n int) (files []fs.FileInfo, err error) {
	defer err2.Handle(&err)
	cmd := fmt.Sprintf("TZ=UTC0 ls -Al --full-time %s", shellescape.Quote(f.name))
	conn := try.To1(f.Start(cmd))
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		line, _, err := r.ReadLine()
		if try.IsEOF(err) {
			break
		}
		finfo := parseLsLine(line)
		if !finfo.IsNil() {
			files = append(files, finfo)
		}
		if n > 0 && len(files) >= n {
			break
		}
	}
	return
}
func (f *File) Stat() (finfo fs.FileInfo, err error) {
	defer err2.Handle(&err, func() {})
	cmd := fmt.Sprintf("TZ=UTC0 ls -Ald --full-time %s", shellescape.Quote(f.name))
	b := try.To1(f.Run(cmd))
	if string(b) == "" {
		return nil, os.ErrNotExist
	}
	b = b[:len(b)-1]
	rfinfo := parseLsLine(b)
	if rfinfo.IsNil() {
		return nil, fmt.Errorf("get file %s stats failed", f.name)
	}
	return rfinfo, nil
}
