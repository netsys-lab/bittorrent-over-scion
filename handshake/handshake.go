package handshake

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
)

// A Handshake is a special message that a peer uses to identify itself
type Handshake struct {
	Pstr       string
	InfoHash   [20]byte
	PeerID     [20]byte
	DhtSupport bool
}

// New creates a new handshake with the standard pstr
func New(infoHash, peerID [20]byte, dhtSupport bool) *Handshake {
	return &Handshake{
		Pstr:       "BitTorrent protocol",
		InfoHash:   infoHash,
		PeerID:     peerID,
		DhtSupport: dhtSupport,
	}
}

// New creates a new handshake with the standard pstr
func NewEmpty() *Handshake {
	return &Handshake{
		Pstr: "BitTorrent protocol",
	}
}

// Serialize serializes the handshake to a buffer
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], h.getReservedBytes()) // 8 reserved bytes
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func (h *Handshake) getReservedBytes() []byte {
	bytes := make([]byte, 8)
	if h.DhtSupport {
		bytes[7] |= 1 // set last bit of reserved bytes
	}
	return bytes
}

// Read parses a handshake from a stream
func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte
	var reserved [8]byte

	copy(reserved[:], handshakeBuf[pstrlen:pstrlen+8])
	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := Handshake{
		Pstr:       string(handshakeBuf[0:pstrlen]),
		InfoHash:   infoHash,
		PeerID:     peerID,
		DhtSupport: (reserved[7] & 1) > 0,
	}

	log.Infof("Received Handshake: %+v", h)

	return &h, nil
}
