package webdav

import (
	"context"
	"fmt"
	"os"

	"github.com/alessio/shellescape"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
	"golang.org/x/net/webdav"
)

type FileSystem struct {
	bash.Session
}

var _ webdav.FileSystem = (*FileSystem)(nil)

func NewFileSystem(sess bash.Session) *FileSystem {
	return &FileSystem{Session: sess}
}

func (c *FileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) (err error) {
	cmd := fmt.Sprintf("mkdir -p %s", shellescape.Quote(name))
	try.To1(c.Run(cmd))
	return
}
func (c *FileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (f webdav.File, err error) {
	f = OpenFile(c.Session, name)
	return
}
func (c *FileSystem) RemoveAll(ctx context.Context, name string) (err error) {
	cmd := fmt.Sprintf("rm -rf %s", shellescape.Quote(name))
	try.To1(c.Run(cmd))
	return
}
func (c *FileSystem) Rename(ctx context.Context, oldName, newName string) (err error) {
	cmd := fmt.Sprintf("mv %s %s", shellescape.Quote(oldName), shellescape.Quote(newName))
	try.To1(c.Run(cmd))
	return
}

func (c *FileSystem) Stat(ctx context.Context, name string) (f os.FileInfo, err error) {
	return OpenFile(c, name).Stat()
}
