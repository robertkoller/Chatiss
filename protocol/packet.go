package protocol

import (
	"encoding/binary"
	"errors"
	"time"
)

const headerSize = 16

// Takes inputted fields and creates a Packet from it
func createPacket(version byte, typeOfPacket byte, flags byte, sessionID uint32, payload []byte) Packet {
	header := Header{
		Magic:         MagicByte,
		Version:       version,
		PacketType:    typeOfPacket,
		Flags:         flags,
		SessionID:     sessionID,
		PayloadLength: uint32(len(payload)),
		Timestamp:     uint32(time.Now().Unix()),
	}
	return Packet{
		Header:  header,
		Payload: payload,
	}
}

// Takes our Packet struct and turns it into the sendable byte version for transport
func packetToBytes(packet Packet) []byte {
	final := make([]byte, headerSize+int(packet.Header.PayloadLength))
	final[0] = packet.Header.Magic
	final[1] = packet.Header.Version
	final[2] = packet.Header.PacketType
	final[3] = packet.Header.Flags
	binary.BigEndian.PutUint32(final[4:8], packet.Header.SessionID)
	binary.BigEndian.PutUint32(final[8:12], packet.Header.PayloadLength)
	binary.BigEndian.PutUint32(final[12:16], packet.Header.Timestamp)
	copy(final[headerSize:], packet.Payload)
	return final
}

// Decyphers sent bytes into our packet struct
func decypherPacket(packet []byte) (Packet, error) {
	if len(packet) < headerSize {
		return Packet{}, errors.New("packet too short")
	}
	if packet[0] != MagicByte {
		return Packet{}, errors.New("magic byte missing")
	}

	payloadLen := binary.BigEndian.Uint32(packet[8:12])
	if len(packet) < headerSize+int(payloadLen) {
		return Packet{}, errors.New("payload length mismatch")
	}

	header := Header{
		Magic:         packet[0],
		Version:       packet[1],
		PacketType:    packet[2],
		Flags:         packet[3],
		SessionID:     binary.BigEndian.Uint32(packet[4:8]),
		PayloadLength: payloadLen,
		Timestamp:     binary.BigEndian.Uint32(packet[12:16]),
	}

	return Packet{
		Header:  header,
		Payload: packet[headerSize : headerSize+int(payloadLen)],
	}, nil
}
