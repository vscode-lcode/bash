package webdav

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestReaddir(t *testing.T) {
	fs := NewFileSystem(client)
	f := try.To1(fs.OpenFile(context.Background(), testDirpath, 0, 0))
	files1 := try.To1(f.Readdir(0))
	files2 := try.To1(os.ReadDir(testDirpath))
	assert.Equal(len(files1), len(files2))
	for i := range files1 {
		finfo := try.To1(files2[i].Info())
		eqalFileInfo(files1[i], finfo)
	}
}

func TestStats(t *testing.T) {
	fs := NewFileSystem(client)
	s1 := try.To1(fs.Stat(context.Background(), testFilepath))
	s2 := try.To1(os.Stat(testFilepath))
	eqalFileInfo(s1, s2)
}

func eqalFileInfo(s1, s2 fs.FileInfo) {
	assert.Equal(s1.Name(), s2.Name())
	assert.Equal(s1.Size(), s2.Size())
	assert.Equal(s1.Mode(), s2.Mode())
	assert.Equal(s1.ModTime().Sub(s2.ModTime()) < time.Second, true)
	assert.Equal(s1.IsDir(), s2.IsDir())
}
