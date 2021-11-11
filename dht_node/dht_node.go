package dht_node
// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"time"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	dhtlog "github.com/anacrolix/log" // logger for dht Node
	"github.com/anacrolix/torrent/metainfo"
	"github.com/scionproto/scion/go/lib/snet"

	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/dht"
	"github.com/netsys-lab/dht/krpc"
	peerStore "github.com/netsys-lab/dht/peer-store"
)

type DhtNode struct {
	Node              *dht.Server
	stats             *dhtStats
	infoHash          [20]byte
	nodeAddr          dht.Addr
	peerPort          uint16
	onNewPeerReceived func(peer peers.Peer)
}

type dhtStats struct {
	receivedPeers     uint32
	blockedPeers      uint32
	recievedSelf      uint32
	numberOfAnnounces uint32
	zeroPortsReceived uint32
}

// New creates a new DHT Node.
// peerPort, the port the controlling peer is listening to
// onNewPeerReceived, a function to be executed when a new Peer was found, used for adding the new peer to the
// controlling peers storage
func New(
	nodeAddr *snet.UDPAddr,
	torrentInfoHash [20]byte,
	startingNodes []dht.Addr,
	peerPort uint16,
	onNewPeerReceived func(peer peers.Peer)) (*DhtNode, error) {

	log.Infof("creating new dht node, initial nodes: %+v, listening on: %+v, peer port: %d", startingNodes, nodeAddr, peerPort)
	stats := &dhtStats{}

	con, err := appnet.Listen(nodeAddr.Host)
	if err != nil {
		log.Error("error creating connection for dht node")
		return nil, err
	}

	localNodeAddr := dht.NewAddr(*nodeAddr)
	dhtConf := dht.NewDefaultServerConfig()
	dhtConf.Conn = con
	dhtConf.PeerStore = &peerStore.InMemory{}
	dhtConf.Logger = dhtLog.Default.FilterLevel(dhtLog.Debug)

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
		return uniqueStartingNodes(append(startingNodes, localNodeAddr)), nil
	}

	node, err := dht.NewServer(dhtConf)
	if err != nil {
		log.Errorf("error creating dht node: %v", err)
		return nil, err
	}
	log.Infof("created dht server with id %+v", node.ID())

	dhtNode := DhtNode{
		Node:              node,
		infoHash:          torrentInfoHash,
		onNewPeerReceived: onNewPeerReceived,
		stats:             stats,
		peerPort:          peerPort,
		nodeAddr:          localNodeAddr,
	}
	go dhtNode.announceLoop()
	return &dhtNode, nil
}

func uniqueStartingNodes(nodes []dht.Addr) []dht.Addr {
	// filter duplicates
	nodesMap := make(map[string]dht.Addr)
	for _, n := range nodes {
		nodesMap[n.String()] = n
	}

	uniqueNodes := make([]dht.Addr, 0)
	for _, v := range nodesMap {
		uniqueNodes = append(uniqueNodes, v)
	}

	return uniqueNodes
}

func (d *DhtNode) Port() *uint16 {
	if d != nil {
		return nil
	}
	port := uint16(d.nodeAddr.Port())
	return &port
}

// announce every 15 min to make sure we do not become questionable to other nodes and to get fresh peers
func (d *DhtNode) announceLoop() {
	ps, err := d.announceAndGetPeers()
	if err != nil {
		log.Error(err)
	}

	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		if ps != nil {
			log.Info("closing traversal")
			ps.Close()
		}
		ps, err = d.announceAndGetPeers()
		if err != nil {
			log.Error(err)
		}
	}
}

// announceAndGetPeers get peers via DHT and announce presence
func (d *DhtNode) announceAndGetPeers() (*dht.Announce, error) {
	log.Info("announcing via dht")
	atomic.AddUint32(&d.stats.numberOfAnnounces, 1)
	ps, err := d.Node.Announce(d.infoHash, int(d.peerPort), false)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	go d.consumePeers(ps)
	return ps, nil
}

func convertPeer(peer dht.Peer) peers.Peer {
	return peers.Peer{
		Addr:  peer.String(),
		Index: 0,
	}
}

func (d *DhtNode) consumePeers(peerStream *dht.Announce) {
	log.Info("consuming peers")
	for v := range peerStream.Peers {
		log.Infof("handling %+v", v)
		for _, cp := range v.Peers {
			log.Infof("handling cp %+v", cp)
			atomic.AddUint32(&d.stats.receivedPeers, 1)
			if cp.Port == 0 {
				atomic.AddUint32(&d.stats.blockedPeers, 1)
				atomic.AddUint32(&d.stats.zeroPortsReceived, 1)
				continue
			}
			if cp.IP.Equal(d.nodeAddr.IP()) && cp.IA.Equal(d.nodeAddr.IA()) {
				atomic.AddUint32(&d.stats.blockedPeers, 1)
				atomic.AddUint32(&d.stats.recievedSelf, 1)
				continue
			}
			d.onNewPeerReceived(convertPeer(cp))
		}
	}
	log.Info("done consuming peers")
}

func (d *DhtNode) Close() {
	d.PrintStats()
	d.Node.Close()
}

func (d *DhtNode) PrintStats() {
	log.Printf("Stats %+v", d.stats)
}
