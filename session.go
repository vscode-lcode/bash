package bash

import (
	"io"
)

type Session interface {
	Start(cmd string) (stream io.ReadWriteCloser, err error)
	Run(cmd string) (result []byte, err error)
}
