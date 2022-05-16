package client

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"bytes"
	"fmt"
	"net"
	"time"

	util "github.com/netsys-lab/bittorrent-over-scion/Utils"
	"github.com/netsys-lab/bittorrent-over-scion/bitfield"
	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/dht_node"
	"github.com/netsys-lab/bittorrent-over-scion/handshake"
	"github.com/netsys-lab/bittorrent-over-scion/message"
	"github.com/netsys-lab/bittorrent-over-scion/peers"

	smp "github.com/netsys-lab/scion-path-discovery/api"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/netsys-lab/scion-path-discovery/socket"

	"github.com/phayes/freeport"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
)

// A Client is a TCP connection with a peer
type Client struct {
	Conn            packets.UDPConn
	Choked          bool
	Bitfield        bitfield.Bitfield
	Peer            peers.Peer
	InfoHash        [20]byte
	PeerID          [20]byte
	DiscoveryConfig *config.PeerDiscoveryConfig
	DhtNode         *dht_node.DhtNode
	NetConn         net.Conn
}

//LastSelection users could add more fields
type ClientSelection struct {
	lastSelectedPathSet pathselection.PathSet
}

//CustomPathSelectAlg this is where the user actually wants to implement its logic in
func (lastSel *ClientSelection) CustomPathSelectAlg(pathSet *pathselection.PathSet) (*pathselection.PathSet, error) {
	// Connect via shortest path
	return pathSet.GetPathSmallHopCount(3), nil
}

//LastSelection users could add more fields
type ClientInitiatedSelection struct {
	lastSelectedPathSet pathselection.PathSet
}

//CustomPathSelectAlg this is where the user actually wants to implement its logic in
func (lastSel *ClientInitiatedSelection) CustomPathSelectAlg(pathSet *pathselection.PathSet) (*pathselection.PathSet, error) {
	// Connect via shortest path
	return pathSet.GetPathSmallHopCount(1), nil
}

// send BitTorrent handshake and wait for response, ping remotes DHT Node when existing as specified in BEP5
func completeHandshake(
	conn net.Conn,
	infohash, peerID [20]byte,
	discoveryConfig *config.PeerDiscoveryConfig) (*handshake.Handshake, error) {

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline
	// time.Sleep(3 * time.Second)
	// log.Infof("Starting handshake with remote %s...", conn.GetRemote())
	req := handshake.New(infohash, peerID, discoveryConfig.EnableDht)

	_, err := conn.Write(req.Serialize())

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	res, err := handshake.Read(conn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("Expected infohash %x but got %x", res.InfoHash, infohash)
	}
	if res.DhtSupport && discoveryConfig.EnableDht {
		log.Info("sending PORT msg")
		_, err = conn.Write(message.FormatPort(discoveryConfig.DhtPort).Serialize())
		if err != nil {
			log.Errorf("error sending PORT msg")
		}
	}
	return res, nil
}

func recvBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		err := fmt.Errorf("Expected bitfield but got %s", msg)
		return nil, err
	}
	if msg.ID != message.MsgBitfield {
		err := fmt.Errorf("Expected bitfield but got ID %d", msg.ID)
		return nil, err
	}
	// fmt.Println(msg.Payload)
	return msg.Payload, nil
}

type MPClient struct {
	Client
	mpSock *smp.MPPeerSock
}

func NewMPClient() *MPClient {
	return &MPClient{}
}

func (mp *MPClient) GetSocket() *smp.MPPeerSock {
	return mp.mpSock
}

func (mp *MPClient) DialAndWaitForConnectBack(
	local string,
	peer peers.Peer,
	peerID,
	infoHash [20]byte,
	discoveryConfig *config.PeerDiscoveryConfig,
	node *dht_node.DhtNode) ([]*Client, error) {
	var err error

	address, err := snet.ParseUDPAddr(peer.Addr)
	if err != nil {
		return nil, err
	}

	var localSocketAddr *snet.UDPAddr

	if local == "" {
		localSocketAddr, err = util.GetDefaultLocalAddr()
		if err != nil {
			return nil, err
		}
	} else {
		localSocketAddr, err = snet.ParseUDPAddr(local)
		if err != nil {
			return nil, err
		}
	}

	localSocketAddr.Host.Port, _ = freeport.GetFreePort()
	localSocketAddrStr := localSocketAddr.String()

	sel := ClientInitiatedSelection{}
	log.Debugf("Dialing from %s to %s", localSocketAddrStr, address)
	mpSock := smp.NewMPPeerSock(localSocketAddrStr, address, &smp.MPSocketOptions{
		Transport:                   "QUIC",
		PathSelectionResponsibility: "CLIENT", // TODO: Server
		MultiportMode:               true,
	})
	mp.mpSock = mpSock
	err = mpSock.Listen()

	if err != nil {
		return nil, err
	}

	// Connect via one path
	err = mpSock.Connect(&sel, &socket.ConnectOptions{
		DontWaitForIncoming:     true,
		SendAddrPacket:          true,
		NoPeriodicPathSelection: true,
		NoMetricsCollection:     true,
	})

	if err != nil {
		return nil, err
	}

	// Wait for incoming connections
	_, err = mpSock.WaitForPeerConnect(nil)

	if err != nil {
		return nil, err
	}
	clients := make([]*Client, 0)
	var bf bitfield.Bitfield
	conLen := len(mpSock.UnderlaySocket.GetConnections())
	for i, v := range mpSock.UnderlaySocket.GetConnections() {

		// Last one is incoming connection, which we need to skip here...
		if i == conLen-1 {
			continue
		}

		_, err = completeHandshake(v, infoHash, peerID, discoveryConfig)
		if err != nil {
			mpSock.UnderlaySocket.CloseAll()
			return nil, err
		}

		log.Debugf("Completed handshake over conn %p", v)
		bf, err = recvBitfield(v)
		if err != nil {
			mpSock.UnderlaySocket.CloseAll()
			return nil, err
		}

		log.Debugf("Connection GetRemote %s", v.GetRemote())

		c := Client{
			Peer:            peer,
			PeerID:          peerID,
			Conn:            v,
			InfoHash:        infoHash,
			Choked:          false,
			Bitfield:        bf,
			DiscoveryConfig: discoveryConfig,
			DhtNode:         node,
		}
		clients = append(clients, &c)
	}

	mp.InfoHash = infoHash
	mp.Peer = peer
	mp.PeerID = peerID
	mp.Bitfield = bf

	return clients, nil
}

func (c *Client) conn() net.Conn {
	if c.NetConn != nil {
		return c.NetConn
	}

	return c.Conn
}

func (c *Client) Handshake() error {
	_, err := completeHandshake(c.conn(), c.InfoHash, c.PeerID, c.DiscoveryConfig)
	if err != nil {
		return err
	}

	c.Bitfield, err = recvBitfield(c.conn())
	if err != nil {
		return err
	}

	return nil
}

func (mp *MPClient) WaitForNewClient() (*Client, error) {

	conn, err := mp.mpSock.UnderlaySocket.WaitForIncomingConn()
	if err != nil {
		return nil, err
	}
	c := Client{
		Peer:     mp.Peer,
		PeerID:   mp.PeerID,
		Conn:     conn,
		InfoHash: mp.InfoHash,
		Choked:   false,
		Bitfield: mp.Bitfield,
	}
	return &c, nil
}

// Read reads and consumes a message from the connection
func (c *Client) Read() (*message.Message, error) {
	msg, err := message.Read(c.conn())
	return msg, err
}

// SendRequest sends a Request message to the peer
func (c *Client) SendRequest(index, begin, length int) error {
	// fmt.Printf("Requesting %d, %d, %d\n", index, begin, length)
	req := message.FormatRequest(index, begin, length)
	_, err := c.conn().Write(req.Serialize())
	return err
}

// SendInterested sends an Interested message to the peer
func (c *Client) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.conn().Write(msg.Serialize())
	return err
}

// SendNotInterested sends a NotInterested message to the peer
func (c *Client) SendNotInterested() error {
	msg := message.Message{ID: message.MsgNotInterested}
	_, err := c.conn().Write(msg.Serialize())
	return err
}

// SendUnchoke sends an Unchoke message to the peer
func (c *Client) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.conn().Write(msg.Serialize())
	return err
}

// SendHave sends a Have message to the peer
func (c *Client) SendHave(index int) error {
	msg := message.FormatHave(index)
	_, err := c.conn().Write(msg.Serialize())
	return err
}
