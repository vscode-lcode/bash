package server

import (
	"net"
	"testing"

	"github.com/lainio/err2/try"
	utils "github.com/vscode-lcode/bash/internal/test-utils"
)

func TestServer(t *testing.T) {
	if !utils.Debug {
		return
	}
	l := try.To1(net.Listen("tcp", "127.0.0.1:43499"))
	hub := NewHub()
	try.To(hub.Serve(l))
}
