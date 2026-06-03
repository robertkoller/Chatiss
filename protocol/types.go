package protocol

const (
	MagicByte byte = 0xBC
	Version1  byte = 0x01
)

// Types of packets we are going to send
const (
	TypeHandshake    byte = 0x01
	TypeHandshakeAck byte = 0x02
	TypeText         byte = 0x03
	TypeTextAck      byte = 0x04
	TypeCallStart    byte = 0x05
	TypeCallAudio    byte = 0x06
	TypeCallEnd      byte = 0x07
	TypeFileStart    byte = 0x08
	TypeFileChunk    byte = 0x09
	TypeFileEnd      byte = 0x0A
	TypeFileAck      byte = 0x0B
	TypePing         byte = 0x0C
	TypePong         byte = 0x0D
	TypeError        byte = 0xFF
)

// Flags (idt I need these yet but for now we keep)
const (
	FlagEncrypted  byte = 0x01
	FlagCompressed byte = 0x02
	FlagFinalChunk byte = 0x04
	FlagEmpty      byte = 0x05
)

type Header struct {
	Magic         byte
	Version       byte
	PacketType    byte
	Flags         byte
	SessionID     uint32
	PayloadLength uint32
	Timestamp     uint32
}

type Packet struct {
	Header  Header
	Payload []byte
}
