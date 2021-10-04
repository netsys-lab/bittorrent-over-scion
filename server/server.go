package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/martenwallewein/torrent-client/bitfield"
	"github.com/martenwallewein/torrent-client/handshake"
	"github.com/martenwallewein/torrent-client/message"
	"github.com/martenwallewein/torrent-client/peers"
	"github.com/martenwallewein/torrent-client/socket"
	"github.com/martenwallewein/torrent-client/torrentfile"

	smp "github.com/netsys-lab/scion-path-discovery/api"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/netsys-lab/scion-path-discovery/socket"

	log "github.com/sirupsen/logrus"
	"github.com/scionproto/scion/go/lib/snet"
)

// A Client is a TCP connection with a peer
type Server struct {
	Conns             []packets.UDPConn
	Choked            bool
	peers             []peers.Peer
	infoHash          [20]byte
	lAddr             string
	localAddr         *snet.UDPAddr
	listener          *net.Listener
	Bitfield          bitfield.Bitfield
	torrentFile       *torrentfile.TorrentFile
	NumPaths          int
	DialBackStartPort int
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

func NewServer(lAddr string, torrentFile *torrentfile.TorrentFile, pathSelectionResponsibility string, numPaths, dialBackPort int) (*Server, error) {

	// Maybe there is an efficient way to do this, but for Bittorrent its not that useful...
	if pathSelectionResponsibility == "client" {
		return nil, errors.New("client based pathselection not supported yet")
	}

	localAddr, err := snet.ParseUDPAddr(lAddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		peers:             make([]peers.Peer, 0),
		Conns:             make([]packets.UDPConn, 0),
		lAddr:             lAddr,
		localAddr:         localAddr,
		torrentFile:       torrentFile,
		NumPaths:          numPaths,
		DialBackStartPort: dialBackPort,
	}

	s.Bitfield = make([]byte, len(torrentFile.PieceHashes))
	for i := range torrentFile.PieceHashes {
		s.Bitfield.SetPiece(i)
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
		if !waitForHandshake {
			fmt.Printf("Got message %d\n", msg.ID)
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

	return nil
}
