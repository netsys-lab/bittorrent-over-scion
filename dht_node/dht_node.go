package dht_node
// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"net"
	"sync/atomic"

	dhtlog "github.com/anacrolix/log" // logger for dht Node
	"github.com/anacrolix/torrent/metainfo"
	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/dht"
	"github.com/netsys-lab/dht/krpc"
	peer_store "github.com/netsys-lab/dht/peer-store"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
)

type DhtNode struct {
	Node              *dht.Server
	peersStream       *dht.Announce
	stats             *dhtStats
	infoHash          [20]byte
	nodeAddr          *net.UDPAddr
	peerPort          uint16
	onNewPeerReceived func(peer peers.Peer)
}

type dhtStats struct {
	receivedPeers     uint32
	blockedPeers      uint32
	numberOfAnnounces uint32
	zeroPortsReceived uint32
}

// New creates a new DHT Node.
// peerPort, the port the controlling peer is listening to
// onNewPeerReceived, a function to be executed when a new Peer was found, used for adding the new peer to the
// controlling peers storage
func New(
	nodeAddr *net.UDPAddr,
	torrentInfoHash [20]byte,
	startingNodes []dht.Addr,
	peerPort uint16,
	onNewPeerReceived func(peer peers.Peer)) (*DhtNode, error) {

	log.Infof("creating new dht node, initial nodes: %+v, listening on: %+v, peer port: %d", startingNodes, nodeAddr, peerPort)
	stats := &dhtStats{}

	con, err := appnet.Listen(nodeAddr)
	if err != nil {
		log.Error("error creating connection for dht Node")
		return nil, err
	}

	dhtConf := dht.NewDefaultServerConfig()
	dhtConf.Conn = con
	dhtConf.PeerStore = &peer_store.InMemory{}
	dhtConf.Logger = dhtlog.Default.FilterLevel(dhtlog.Debug)
	dhtConf.OnAnnouncePeer = func(infoHash metainfo.Hash, scionAddr snet.UDPAddr, port int, portOk bool) {
		log.Debugf("handling announce for %s - %s - %d - %t", infoHash, scionAddr.String(), port, portOk)
		var infoH [20]byte
		copy(infoH[:], infoHash.Bytes())
		if torrentInfoHash != infoH || !portOk || port == 0 {
			atomic.AddUint32(&stats.blockedPeers, 1)
			if port == 0 {
				atomic.AddUint32(&stats.zeroPortsReceived, 1)
			}
			log.Infof("rejected peer %s - %s - %d - %t", infoHash, scionAddr, port, portOk)
			return
		}

		dhtConf.PeerStore.AddPeer(infoHash, krpc.NodeAddr{
			IP:   scionAddr.Host.IP,
			Port: port,
			IA:   scionAddr.IA,
		})
		atomic.AddUint32(&stats.receivedPeers, 1)
	}
	dhtConf.StartingNodes = func() ([]dht.Addr, error) {
		return startingNodes, nil
	}
	node, err := dht.NewServer(dhtConf)
	if err != nil {
		log.Errorf("error creating dht Node: %v", err)
		return nil, err
	}
	log.Infof("created dht server with id %+v", node.ID())

	dhtNode := DhtNode{
		Node:              node,
		infoHash:          torrentInfoHash,
		onNewPeerReceived: onNewPeerReceived,
		stats:             stats,
		peerPort:          peerPort,
		nodeAddr:          nodeAddr,
	}
	dhtNode.announceAndGetPeers()
	return &dhtNode, nil
}

func (d *DhtNode) Port() *uint16 {
	if d != nil {
		return nil
	}
	port := uint16(d.nodeAddr.Port)
	return &port
}

// announceAndGetPeers get peers via DHT and announce presence
func (d *DhtNode) announceAndGetPeers() {
	log.Info("announcing via dht")
	atomic.AddUint32(&d.stats.numberOfAnnounces, 1)
	if d.peersStream != nil {
		d.peersStream.Close()
	}
	ps, err := d.Node.Announce(d.infoHash, int(d.peerPort), false)
	if err != nil {
		log.Error(err)
		return
	}
	d.peersStream = ps
	go d.consumePeers()
}

func convertPeer(peer dht.Peer) peers.Peer {
	return peers.Peer{
		Addr:  peer.String(),
		Index: 0,
	}
}

func (d *DhtNode) consumePeers() {
	for v := range d.peersStream.Peers {
		for _, cp := range v.Peers {
			atomic.AddUint32(&d.stats.receivedPeers, 1)
			if cp.Port == 0 {
				atomic.AddUint32(&d.stats.blockedPeers, 1)
				atomic.AddUint32(&d.stats.zeroPortsReceived, 1)
				continue
			}
			d.onNewPeerReceived(convertPeer(cp))
		}
	}
}

func (d *DhtNode) Close() {
	if d.peersStream != nil {
		d.peersStream.Close()
	}
	d.PrintStats()
	d.Node.Close()
}

func (d *DhtNode) PrintStats() {
	log.Printf("Announced %d times, recieved %d peers, blocked %d peers, blocked 0-port %d peers",
		d.stats.numberOfAnnounces, d.stats.receivedPeers, d.stats.blockedPeers, d.stats.zeroPortsReceived)
}
