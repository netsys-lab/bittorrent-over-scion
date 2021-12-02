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
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"

	smp "github.com/netsys-lab/scion-path-discovery/api"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/netsys-lab/scion-path-discovery/socket"

	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
)

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
}

//LastSelection users could add more fields
type ServerSelection struct {
	lastSelectedPathSet pathselection.PathSet
	numPaths            int
}

// We use server-side pathselection, meaning the server connects back to the client
func (s *ServerSelection) CustomPathSelectAlg(pathSet *pathselection.PathSet) (*pathselection.PathSet, error) {
	ps := pathSet.GetPathSmallHopCount(s.numPaths)
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
	}

	s.Bitfield = make([]byte, len(torrentFile.PieceHashes))
	for i := range torrentFile.PieceHashes {
		s.Bitfield.SetPiece(i)
	}

	if discoveryConfig.EnableDht {
		nodeAddr := *localAddr.Host
		nodeAddr.Port = int(discoveryConfig.DhtPort)

		startingNodes := append(torrentFile.Nodes, discoveryConfig.DhtNodes...)
		node, err := dht_node.New(&nodeAddr, torrentFile.InfoHash, startingNodes, uint16(localAddr.Host.Port), func(peer peers.Peer) {
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
			remoteDht := snet.UDPAddr{
				IA: remote.IA,
				Host: &net.UDPAddr{
					IP:   remote.Host.IP,
					Port: int(remoteDhtPort),
					Zone: remote.Host.Zone,
				},
			}
			log.Debugf("sending dht ping to %s",
				remoteDht)
			go s.dhtNode.Node.Ping(&remoteDht)
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

	if s.discoveryConfig.EnableDht && s.dhtNode != nil && hs.DhtSupport {
		defer func() {
			log.Info("sending ping")
			_, err := conn.Write(message.FormatPort(s.discoveryConfig.DhtPort).Serialize())
			if err != nil {
				log.Error("error sending ping")
			}
		}()
	}

	msg := message.Message{ID: message.MsgBitfield, Payload: s.Bitfield}
	_, err = conn.Write(msg.Serialize())
	if err != nil {
		return err
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
