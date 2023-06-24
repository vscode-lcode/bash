package webdav

import (
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
	utils "github.com/vscode-lcode/bash/internal/test-utils"
	"github.com/vscode-lcode/bash/server"
)

var client bash.Session

func TestMain(m *testing.M) {
	l := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	hub := server.NewHub()
	defer hub.Close()
	go hub.Serve(l)
	go func() {
		cmd := exec.Command("bash", "+o", "history", "-i")
		conn := try.To1(net.Dial("tcp", l.Addr().String()))
		cmd.Stdin = conn
		if utils.Debug {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		cmd.Start()
		cmd.Wait()
	}()
	clients := server.ExportClients(hub)
	time.Sleep(500 * time.Millisecond)
	for _, c := range clients {
		client = c
		break
	}
	m.Run()
}

func TestClient(t *testing.T) {
	b, err := client.Run("echo -n hello")
	if err != nil {
		t.Error(err)
	}
	s := string(b)
	t.Log(s)
}
