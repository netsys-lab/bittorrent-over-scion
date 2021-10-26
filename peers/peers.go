package peers

import (
	"fmt"
)

// Peer encodes connection information for a peer
type Peer struct {
	Addr  string
	Index int
}

// Unmarshal parses peer IP addresses and ports from a buffer
func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 for IP, 2 for port
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		err := fmt.Errorf("Received malformed PeerSet")
		return nil, err
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		fmt.Errorf("decoding peer information currently not supported")
		//offset := i * peerSize
		//PeerSet[i].IP = net.IP(peersBin[offset : offset+4])
		//PeerSet[i].Port = binary.BigEndian.Uint16([]byte(peersBin[offset+4 : offset+6]))
	}
	return peers, nil
}
func (p Peer) String() string {
	return fmt.Sprintf("%s-%d", p.Addr, p.Index)
}

// PeerSet used for storing a number of unique Peers
type PeerSet struct {
	Peers map[Peer]struct{}
}

var peerMember struct{}

// NewPeerSet creates a new PeerSet and allocates memory to store initialSize number of Peers
func NewPeerSet(initialSize int) PeerSet {
	return PeerSet{Peers: make(map[Peer]struct{}, initialSize)}
}

func (ps PeerSet) Add(peer Peer) {
	ps.Peers[peer] = peerMember
}

func (ps PeerSet) Delete(peer Peer) {
	delete(ps.Peers, peer)
}

func (ps PeerSet) Contains(peer Peer) bool {
	_, present := ps.Peers[peer]
	return present
}
