package server

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/netsys-lab/bittorrent-over-scion/bitfield"
	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/dht_node"
	"github.com/netsys-lab/bittorrent-over-scion/handshake"
	"github.com/netsys-lab/bittorrent-over-scion/message"
	ps "github.com/netsys-lab/bittorrent-over-scion/pathselection"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"

	smp "github.com/netsys-lab/scion-path-discovery/api"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/netsys-lab/scion-path-discovery/socket"

	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
)

type ExtPeer struct {
	sock      *smp.MPPeerSock
	selection *ServerSelection
}

// A Client is a TCP connection with a peer
type Server struct {
	Conns             []packets.UDPConn
	Choked            bool
	peers             peers.PeerSet
	infoHash          [20]byte
	lAddr             string
	localAddr         *snet.UDPAddr
	listener          *net.Listener
	Bitfield          bitfield.Bitfield
	torrentFile       *torrentfile.TorrentFile
	NumPaths          int
	DialBackStartPort int
	discoveryConfig   *config.PeerDiscoveryConfig
	dhtNode           *dht_node.DhtNode // dht note controlled by this server
	pathStore         *ps.PathSelectionStore
	extPeers          []ExtPeer
}

//LastSelection users could add more fields
type ServerSelection struct {
	lastSelectedPathSet pathselection.PathSet
	numPaths            int
	usedPaths           []snet.Path
}

// We use server-side pathselection, meaning the server connects back to the client
func (s *ServerSelection) CustomPathSelectAlg(pathSet *pathselection.PathSet) (*pathselection.PathSet, error) {
	// ps := pathSet.GetPathSmallHopCount(s.numPaths)
	// Filter by used paths
	ps := &pathselection.PathSet{
		Address: pathSet.Address,
		Paths:   make([]pathselection.PathQuality, 0),
	}

	for _, v := range s.usedPaths {
		pathQualityIndex := pathset.Paths.FindIndexByPathString(pathselection.PathToString(v))
		ps.Paths = append(ps.Paths, pathSet.Paths[pathQualityIndex])
	}

	return ps, nil
}

func NewServer(lAddr string, torrentFile *torrentfile.TorrentFile, pathSelectionResponsibility string, numPaths, dialBackPort int, discoveryConfig *config.PeerDiscoveryConfig) (*Server, error) {

	// Maybe there is an efficient way to do this, but for Bittorrent its not that useful...
	if pathSelectionResponsibility == "client" {
		return nil, errors.New("client based pathselection not supported yet")
	}

	localAddr, err := snet.ParseUDPAddr(lAddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		peers:             peers.NewPeerSet(0),
		Conns:             make([]packets.UDPConn, 0),
		lAddr:             lAddr,
		localAddr:         localAddr,
		torrentFile:       torrentFile,
		NumPaths:          numPaths,
		DialBackStartPort: dialBackPort,
		discoveryConfig:   discoveryConfig,
		pathStore:         ps.NewPathSelectionStore(),
		extPeers:          make([]ExtPeer, 0),
	}

	s.Bitfield = make([]byte, len(torrentFile.PieceHashes))
	for i := range torrentFile.PieceHashes {
		s.Bitfield.SetPiece(i)
	}

	if discoveryConfig.EnableDht {
		nodeAddr := localAddr.Copy()
		nodeAddr.Host.Port = int(discoveryConfig.DhtPort)

		startingNodes := append(torrentFile.Nodes, discoveryConfig.DhtNodes...)
		node, err := dht_node.New(nodeAddr, torrentFile.InfoHash, startingNodes, uint16(localAddr.Host.Port), func(peer peers.Peer) {
			log.Infof("received peer via dht: %s, peer already known: %t", peer, s.hasPeer(peer))
			s.peers.Add(peer)
		})
		if err != nil {
			return nil, err
		}
		s.dhtNode = node
	}

	return s, nil
}

func (s *Server) updateDisjointPathselection(p ExtPeer) {
	// Create a PeerPathEntry, add it to the store
	// Beforehand, fill available paths
	s.extPeers = append(s.extPeers, p)
	pp := ps.PeerPathEntry{
		PeerAddrStr:    p.sock.Peer.String(),
		PeerAddr:       *p.sock.Peer,
		AvailablePaths: make([]snet.Path, 0), // TODO: Get available paths from socket
		UsedPaths:      make([]snet.Path, 0),
	}

	s.pathStore.AddPeerEntry(pp)

	for _, v := range s.extPeers {
		// After adding, we get the used Paths, which we save in the selection
		// We need a unique identifier for paths to map them to PathQualities
		paths := s.pathStore.Get(v.sock.Peer.String()).UsedPaths
		v.selection.usedPaths = paths

		// Update pathselection in socket
		v.sock.ForcePathSelection()
	}

}

func (s *Server) ListenHandshake() error {
	var err error

	mpListener := smp.NewMPListener(s.lAddr, &smp.MPListenerOptions{
		Transport: "QUIC",
	})

	err = mpListener.Listen()
	if err != nil {
		return err
	}
	startPort := s.DialBackStartPort
	for {
		log.Info("waiting for MPPeer socket connect")
		remote, err := mpListener.WaitForMPPeerSockConnect()
		if err != nil {
			return err
		}
		log.Infof("Got new Client, dialing back")
		startPort += 101 // Just increase by a random number to avoid using often used ports (e.g. 50000)
		go func(remote *snet.UDPAddr, startPort int) {
			ladr := s.localAddr.Copy()
			ladr.Host.Port = startPort
			mpSock := smp.NewMPPeerSock(ladr.String(), remote, &smp.MPSocketOptions{
				Transport:                   "QUIC",
				PathSelectionResponsibility: "CLIENT",
				MultiportMode:               true,
			})
			log.Debugf("New Server listening on %s", ladr.String())
			err = mpSock.Listen()
			if err != nil {

				log.Errorf("Failed to listen %v", err)
				return
			}
			log.Debugf("Connecting to %s", remote.String())
			// TODO: We need to make this server selection editable
			// And maybe we need a method to force pathselection being done
			// (And a method to get all available paths)
			err = mpSock.Connect(&ServerSelection{
				numPaths: s.NumPaths,
			}, &socket.ConnectOptions{
				SendAddrPacket:      true,
				DontWaitForIncoming: true,
			})
			if err != nil {
				log.Error(err)
				return
			}
			conns := mpSock.UnderlaySocket.GetConnections()
			log.Debugf("Got new connections %d", len(conns))
			log.Infof("Starting upload to new client...")
			for i, conn := range conns {
				if i == 0 {
					log.Debugf("Skip incoming connection")
					continue
				}
				s.Conns = append(s.Conns, conn)
				log.Debugf("Starting reading on conn %d with handshake %d", i, i == 0)
				go s.handleConnection(conn, true)

			}
			for {
				// Filter for new connections
				conns := <-mpSock.OnConnectionsChange
				log.Debugf("Got new connections %d", len(conns))
				for i, conn := range conns {
					connAlreadyOpen := false
					for _, oldConn := range s.Conns {
						if oldConn.GetId() == conn.GetId() {
							connAlreadyOpen = true
							log.Debugf("Got already open conn for id %s", conn.GetId())
						}
					}
					if !connAlreadyOpen {
						s.Conns = append(s.Conns, conn)
						log.Debugf("Starting reading on conn %p with handshake %d", conn, i == 0)
						log.Debugf(conn.LocalAddr().String())
						go s.handleConnection(conn, true)
					}

				}
			}

		}(remote, startPort)

	}
}

func (s *Server) handleConnection(conn packets.UDPConn, waitForHandshake bool) error {
	if waitForHandshake {
		s.handleIncomingHandshake(conn)
	}

	for {
		msg, err := message.Read(conn)
		if err != nil {
			return err
		}

		if msg == nil { // keep-alive
			return nil
		}

		switch msg.ID {
		case message.MsgInterested:
			retMsg := message.Message{ID: message.MsgUnchoke, Payload: []byte{}}
			_, err := conn.Write(retMsg.Serialize())
			if err != nil {
				return err
			}
		case message.MsgRequest:
			index, begin, length := message.ParseRequest(msg)
			log.Debugf("got request for piece %d", index)
			if !waitForHandshake {
				fmt.Printf("Got request msg with index %d, begin %d, length %d\n", index, begin, length)
			}

			buf := make([]byte, 8)
			binary.BigEndian.PutUint32(buf[0:4], uint32(index))
			binary.BigEndian.PutUint32(buf[4:8], uint32(begin))
			buf = append(buf, s.torrentFile.Content[(index*s.torrentFile.PieceLength)+begin:(index*s.torrentFile.PieceLength)+begin+length]...)
			retMsg := message.Message{ID: message.MsgPiece, Payload: buf}
			_, err := conn.Write(retMsg.Serialize())
			if err != nil {
				return err
			}
		case message.MsgPort:
			log.Debug("got port message")
			if !s.discoveryConfig.EnableDht ||
				s.dhtNode == nil {
				log.Info("got port message but dht is not enabled")
				break
			}
			remote := conn.GetRemote()
			if remote == nil {
				log.Error("could not get remote from port message")
				break
			}
			remoteDhtPort, err := message.ParsePort(msg)
			if err != nil {
				log.Error("could not parse port message")
				break
			}
			remoteDht := remote.Copy()
			remoteDht.Host.Port = int(remoteDhtPort)
			log.Debugf("sending dht ping to %s",
				remoteDht)
			go s.dhtNode.Node.Ping(remoteDht)
		}
	}
}

func (s *Server) handleIncomingHandshake(conn packets.UDPConn) error {
	hs, err := handshake.Read(conn)
	if err != nil {
		return err
	}

	_, err = conn.Write(hs.Serialize())
	if err != nil {
		return err
	}

	msg := message.Message{ID: message.MsgBitfield, Payload: s.Bitfield}
	_, err = conn.Write(msg.Serialize())
	if err != nil {
		return err
	}

	if s.discoveryConfig.EnableDht && s.dhtNode != nil && hs.DhtSupport {
		log.Info("sending PORT msg")
		_, err := conn.Write(message.FormatPort(s.discoveryConfig.DhtPort).Serialize())
		if err != nil {
			log.Error("error sending PORT msg")
		}
	}

	return nil
}

func (s Server) hasPeer(peer peers.Peer) bool {
	return s.peers.Contains(peer)
}

func (s Server) Close() {
	if s.dhtNode != nil {
		s.dhtNode.Close()
	}
}
