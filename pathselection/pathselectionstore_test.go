package pathselection

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/common"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/spath"
	"github.com/stretchr/testify/assert"
)

type SamplePath struct {
	meta snet.PathMetadata
	raw  string
}

func (s *SamplePath) Metadata() *snet.PathMetadata {
	return &s.meta
}

func (s *SamplePath) Copy() snet.Path {
	return nil
}

func (s *SamplePath) Destination() addr.IA {
	ia, _ := addr.IAFromString(s.raw)
	return ia
}

func (s *SamplePath) Path() spath.Path {
	return spath.Path{}
}

func (s *SamplePath) UnderlayNextHop() *net.UDPAddr {
	return nil
}

func mustParseIntf(intfShort string) snet.PathInterface {
	parts := strings.Split(intfShort, "#")
	if len(parts) != 2 {
		panic(fmt.Sprintf("bad interface %q", intfShort))
	}
	ia, _ := addr.IAFromString("1-ff00:0:" + parts[0])
	ifid, _ := strconv.Atoi(parts[1])
	return snet.PathInterface{IA: ia, ID: common.IFIDType(ifid)}
}

func makePath(intfStrs ...string) snet.Path {
	intfs := make([]snet.PathInterface, len(intfStrs))
	for i, intfStr := range intfStrs {
		intfs[i] = mustParseIntf(intfStr)
	}
	return &SamplePath{
		meta: snet.PathMetadata{
			Interfaces: intfs,
		},
	}
}

var _ snet.Path = (*SamplePath)(nil)

func TestMultiplePeers(t *testing.T) {
	store := NewPathSelectionStore()
	addr := "19-ffaa:1:c3f,[141.44.25.148]:43000"
	addr2 := "19-ffaa:1:c3f,[141.44.25.151]:43000"
	addr3 := "19-ffaa:1:c3f,[141.44.25.152]:43000"
	pAddr, _ := snet.ParseUDPAddr(addr)
	pAddr2, _ := snet.ParseUDPAddr(addr2)
	pAddr3, _ := snet.ParseUDPAddr(addr3)
	p := PeerPathEntry{
		PeerAddrStr:    addr,
		PeerAddr:       *pAddr,
		AvailablePaths: make([]snet.Path, 0),
		UsedPaths:      make([]snet.Path, 0),
	}

	p2 := PeerPathEntry{
		PeerAddrStr:    addr2,
		PeerAddr:       *pAddr2,
		AvailablePaths: make([]snet.Path, 0),
		UsedPaths:      make([]snet.Path, 0),
	}

	p3 := PeerPathEntry{
		PeerAddrStr:    addr3,
		PeerAddr:       *pAddr3,
		AvailablePaths: make([]snet.Path, 0),
		UsedPaths:      make([]snet.Path, 0),
	}

	path1 := makePath("a#1", "b#1")
	path11 := makePath("a#1", "f#1")
	path2 := makePath("c#1", "d#1")
	path21 := makePath("c#1", "1#1")
	path3 := makePath("a#1", "c#1")

	p.AvailablePaths = append(p.AvailablePaths, path1)
	p.AvailablePaths = append(p.AvailablePaths, path11)
	p2.AvailablePaths = append(p2.AvailablePaths, path2)
	p2.AvailablePaths = append(p2.AvailablePaths, path21)
	p3.AvailablePaths = append(p3.AvailablePaths, path3)

	t.Run("TestAddFirstPeer", func(t *testing.T) {
		store.AddPeerEntry(p)
		assert.Equal(t, len(store.data), 1)
		assert.Equal(t, len(store.data[addr].UsedPaths), 2)
	})

	t.Run("TestNonConflictingPeer", func(t *testing.T) {
		store.AddPeerEntry(p2)
		assert.Equal(t, len(store.data), 2)
		assert.Equal(t, len(store.data[addr].UsedPaths), 2)
		assert.Equal(t, len(store.data[addr].UsedPaths), len(store.data[addr].AvailablePaths))
		assert.Equal(t, len(store.data[addr2].UsedPaths), 2)
	})

	t.Run("TestConflictingPeer", func(t *testing.T) {
		store.AddPeerEntry(p3)
		assert.Equal(t, len(store.data), 3)
		assert.Equal(t, len(store.data[addr].UsedPaths), 1)
		assert.Equal(t, len(store.data[addr].UsedPaths), len(store.data[addr].AvailablePaths)-1)
		assert.Equal(t, len(store.data[addr2].UsedPaths), 2)
		assert.Equal(t, len(store.data[addr2].UsedPaths), len(store.data[addr2].AvailablePaths))
		assert.Equal(t, len(store.data[addr3].UsedPaths), 1)
	})

}

// Test if adding a peer that does not conflict still uses all available paths
func TestAddNonConflictingPeer(t *testing.T) {

}
