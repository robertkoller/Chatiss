package sessions

import (
	"bytes"
	"errors"
	"sync"

	"github.com/robertkoller/Chatiss/protocol"
)

// FileTransfer tracks an in-progress incoming file.
type FileTransfer struct {
	Info     protocol.FileInfo
	chunks   map[uint32][]byte
	received uint32
	mu       sync.Mutex
}

func NewFileTransfer(info protocol.FileInfo) *FileTransfer {
	return &FileTransfer{
		Info:   info,
		chunks: make(map[uint32][]byte, info.TotalChunks),
	}
}

// AddChunk stores a chunk and returns true when all chunks have arrived.
func (transfer *FileTransfer) AddChunk(index uint32, data []byte) bool {
	transfer.mu.Lock()
	defer transfer.mu.Unlock()
	if _, exists := transfer.chunks[index]; !exists {
		chunk := make([]byte, len(data))
		copy(chunk, data)
		transfer.chunks[index] = chunk
		transfer.received++
	}
	return transfer.received == transfer.Info.TotalChunks
}

// Assemble returns the complete file bytes in order.
func (transfer *FileTransfer) Assemble() ([]byte, error) {
	transfer.mu.Lock()
	defer transfer.mu.Unlock()
	if transfer.received != transfer.Info.TotalChunks {
		return nil, errors.New("file transfer incomplete")
	}
	var buf bytes.Buffer
	for i := uint32(0); i < transfer.Info.TotalChunks; i++ {
		chunk, ok := transfer.chunks[i]
		if !ok {
			return nil, errors.New("missing chunk")
		}
		buf.Write(chunk)
	}
	return buf.Bytes(), nil
}
