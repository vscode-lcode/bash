package server

import (
	"context"
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
	Timeout       time.Duration
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint64]*bash.Client),
		locker:  &sync.RWMutex{},
		Timeout: 2 * time.Second,
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
		go hub.ServeConn(conn)
	}
}

func (hub *Hub) ServeConn(conn net.Conn) (err error) {
	defer err2.Handle(&err, func() {
		conn.Close()
	})
	var h Header
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(200*time.Millisecond, func() {
		defer cancel()
		if err := ctx.Err(); err != nil {
			return
		}
		h.encodeMsgType(MsgInitSession)
	})
	go func() {
		defer cancel()
		r := hex.NewDecoder(io.LimitReader(conn, headerSize))
		if _, err := io.ReadFull(r, h[:]); err != nil {
			return
		}
	}()
	<-ctx.Done()
	if h.MsgType() == MsgInitSession {
		return hub.NewClientSession(conn)
	}
	client := try.To1(hub.getClient(h))
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
	client.Timeout = hub.Timeout
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
		if onClose != nil {
			defer onClose()
		}
	}

	// drop stdout
	// check if shell is running and exit
	var drop = make([]byte, 512)
	for {
		_, err = conn.Read(drop)
		if err != nil {
			return
		}
	}
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
