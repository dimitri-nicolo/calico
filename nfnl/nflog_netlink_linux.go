package nfnl

import (
	"unsafe"
)

const (
	SizeofNflogMsgPktHdr       = 0x4
	SizeofNflogMsgPktHw        = 0xC
	SizeofNflogMsgPktTimestamp = 0x10
	SizeofNflogMsgConfigCmd    = 0x1
	SizeofNflogMsgConfigBufSiz = 0x4
	SizeofNflogMsgConfigMode   = 0x6
)

type NflogMsgPktHdr struct {
	hwProtocol uint16
	hook       uint8
	_pad       uint8
}

type NflogMsgPktHw struct {
	hwAddrlen uint16
	_pad      uint16
	hwAddr    [8]uint8
}

type NflogMsgPktTimestamp struct {
	sec  uint64
	usec uint64
}

type NflogMsgConfigCmd struct {
	command uint8
}

func NewNflogMsgConfigCmd(command int) *NflogMsgConfigCmd {
	return &NflogMsgConfigCmd{
		command: uint8(command),
	}
}

func DeserializeNflogMsgConfigCmd(b []byte) *NflogMsgConfigCmd {
	return (*NflogMsgConfigCmd)(unsafe.Pointer(&b[0:SizeofNflogMsgConfigCmd][0]))
}

func (msg *NflogMsgConfigCmd) Len() int {
	return SizeofNflogMsgConfigCmd
}

func (msg *NflogMsgConfigCmd) Serialize() []byte {
	return (*(*[SizeofNflogMsgConfigCmd]byte)(unsafe.Pointer(msg)))[:]
}

type NflogMsgConfigMode struct {
	copyRange uint32
	copyMode  uint8
	_pad      uint8
}

func NewNflogMsgConfigMode(copyRange int, copyMode int) *NflogMsgConfigMode {
	return &NflogMsgConfigMode{
		copyRange: uint32(copyRange),
		copyMode:  uint8(copyMode),
	}
}

func DeserializeNflogMsgConfigMode(b []byte) *NflogMsgConfigMode {
	return (*NflogMsgConfigMode)(unsafe.Pointer(&b[0:SizeofNflogMsgConfigMode][0]))
}

func (msg *NflogMsgConfigMode) Len() int {
	return SizeofNflogMsgConfigMode
}

func (msg *NflogMsgConfigMode) Serialize() []byte {
	return (*(*[SizeofNflogMsgConfigMode]byte)(unsafe.Pointer(msg)))[:]
}

type NflogMsgConfigBufSiz struct {
	bufsiz uint32
}

func NewNflogMsgConfigBufSiz(bufsiz int) *NflogMsgConfigBufSiz {
	return &NflogMsgConfigBufSiz{
		bufsiz: uint32(bufsiz),
	}
}

func DeserializeNflogMsgConfigBufSiz(b []byte) *NflogMsgConfigBufSiz {
	return (*NflogMsgConfigBufSiz)(unsafe.Pointer(&b[0:SizeofNflogMsgConfigBufSiz][0]))
}

func (msg *NflogMsgConfigBufSiz) Len() int {
	return SizeofNflogMsgConfigBufSiz
}

func (msg *NflogMsgConfigBufSiz) Serialize() []byte {
	return (*(*[SizeofNflogMsgConfigBufSiz]byte)(unsafe.Pointer(msg)))[:]
}
