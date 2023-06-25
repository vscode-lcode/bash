package server

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
)

type Hub struct {
	clients map[uint64]*bash.Client
	locker  *sync.RWMutex

	nextID uint64

	OnSessionOpen func(bash.Session) func()
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint64]*bash.Client),
		locker:  &sync.RWMutex{},
	}
}

func (hub *Hub) Close() (err error) {
	hub.locker.Lock()
	defer hub.locker.Unlock()
	for _, c := range hub.clients {
		c.Close()
	}
	hub.clients = make(map[uint64]*bash.Client)
	return
}

func (hub *Hub) Serve(l net.Listener) (err error) {
	for {
		conn := try.To1(l.Accept())
		go hub.serve(conn)
	}
}

func (hub *Hub) serve(conn net.Conn) (err error) {
	defer err2.Handle(&err)
	var w = make(chan Header)
	time.AfterFunc(200*time.Millisecond, func() {
		close(w)
	})
	go func() {
		r := hex.NewDecoder(io.LimitReader(conn, headerSize))
		var hdr Header
		_, err := io.ReadFull(r, hdr[:])
		if err != nil {
			return
		}
		w <- hdr
	}()
	hdr, ok := <-w
	if !ok || hdr.MsgType() == MsgInitSession {
		return hub.NewClientSession(conn)
	}
	client := try.To1(hub.getClient(hdr))
	try.To(client.HandleConn(conn))
	return
}

func (hub *Hub) getClient(hdr Header) (client *bash.Client, err error) {
	hub.locker.RLock()
	defer hub.locker.RUnlock()

	id := hdr.ID()

	client, ok := hub.clients[id]
	if !ok {
		return nil, ErrClientNotExists
	}
	if client == nil {
		return nil, ErrClientIDReused
	}

	return
}

var (
	ErrClientNotExists = fmt.Errorf("client session is not exists")
	ErrClientIDReused  = fmt.Errorf("client id is reused. %w", ErrClientNotExists)
)

func (hub *Hub) NewClientSession(conn net.Conn) (err error) {
	defer err2.Handle(&err)
	client := bash.NewClient(conn)
	var id uint64 = hub.genClientID()
	var hdr Header
	hdr.encode(id)
	client.ID = hdr.String()
	func() {
		hub.locker.Lock()
		defer hub.locker.Unlock()
		hub.clients[id] = client
	}()
	defer func() {
		hub.locker.Lock()
		defer hub.locker.Unlock()
		hub.clients[id] = nil
	}()

	if hub.OnSessionOpen != nil {
		onClose := hub.OnSessionOpen(client)
		defer onClose()
	}

	// arrived only when client exit
	_, err = io.ReadAll(conn)
	return
}

func (hub *Hub) genClientID() uint64 {
	hub.locker.RLock()
	defer hub.locker.RUnlock()
	for {
		id := hub.nextID
		hub.nextID++
		if client, ok := hub.clients[id]; !ok || client == nil {
			return id
		}
	}
}
