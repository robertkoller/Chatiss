package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"os"
)

const ChunkSize = 32 * 1024 // 32 KB per chunk

type FileInfo struct {
	Name        string `json:"name"`
	Size        uint64 `json:"size"`
	TotalChunks uint32 `json:"total_chunks"`
}

func CreateFileStart(session *Session, path string) ([]byte, FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, FileInfo{}, err
	}
	info := FileInfo{
		Name:        stat.Name(),
		Size:        uint64(stat.Size()),
		TotalChunks: uint32(math.Ceil(float64(stat.Size()) / ChunkSize)),
	}
	payload, err := json.Marshal(info)
	if err != nil {
		return nil, FileInfo{}, err
	}
	return packetToBytes(createPacket(Version1, TypeFileStart, FlagEmpty, session.ID, payload)), info, nil
}

// CreateFileChunk builds one chunk packet.
// Payload layout: [4 bytes chunk index][chunk data]
func CreateFileChunk(session *Session, index uint32, data []byte) []byte {
	payload := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(payload[:4], index)
	copy(payload[4:], data)
	return packetToBytes(createPacket(Version1, TypeFileChunk, FlagEmpty, session.ID, payload))
}

func CreateFileEnd(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypeFileEnd, FlagEmpty, session.ID, nil))
}

func parseFileStart(packet Packet) (FileInfo, error) {
	var info FileInfo
	if err := json.Unmarshal(packet.Payload, &info); err != nil {
		return FileInfo{}, errors.New("invalid file start payload")
	}
	return info, nil
}

func parseFileChunk(packet Packet) (index uint32, data []byte, err error) {
	if len(packet.Payload) < 4 {
		return 0, nil, errors.New("file chunk payload too short")
	}
	index = binary.BigEndian.Uint32(packet.Payload[:4])
	data = packet.Payload[4:]
	return index, data, nil
}
