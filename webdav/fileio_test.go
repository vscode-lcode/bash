package webdav

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestRead(t *testing.T) {
	fs := NewFileSystem(client)
	f := try.To1(fs.OpenFile(context.Background(), testFilepath, 0, 0))
	s1 := try.To1(io.ReadAll(f))
	s2 := try.To1(os.ReadFile(testFilepath))
	assert.Equal(string(s1), string(s2))
}

func TestWrite(t *testing.T) {
	fs := NewFileSystem(client)
	f := try.To1(fs.OpenFile(context.Background(), testFilepath, 0, 0))
	s1 := try.To1(io.ReadAll(f))
	try.To(f.Close())

	f = try.To1(fs.OpenFile(context.Background(), testFilepath, 0, 0))
	try.To1(f.Write(s1))
	try.To(f.Close())

	s2 := try.To1(os.ReadFile(testFilepath))
	assert.Equal(string(s1), string(s2))
}
