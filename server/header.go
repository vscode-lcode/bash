package server

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/vscode-lcode/bash"
)

type Header [12]byte

const headerSize = int64(cap(Header{}) * 2)

func (h *Header) Version() uint8 {
	return h[0]
}

func (h *Header) MsgType() MsgType {
	return MsgType(h[1])
}

func (h *Header) ID() uint64 {
	return binary.BigEndian.Uint64(h[4:12])
}

func (h *Header) encode(id uint64) {
	binary.BigEndian.PutUint64(h[4:12], id)
}

func (h *Header) encodeMsgType(msgType MsgType) {
	h[1] = byte(msgType)
}

func (h *Header) String() string {
	return hex.EncodeToString(h[:])
}

func ExportClients(hub *Hub) map[uint64]*bash.Client {
	return hub.clients
}

type MsgType uint8

const (
	MsgToSession MsgType = iota
	MsgInitSession
)
