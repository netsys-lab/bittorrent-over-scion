package client

import (
	"bytes"
	"fmt"

	smp "github.com/netsys-lab/scion-path-discovery/api"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/scionproto/scion/go/lib/snet"

	"github.com/veggiedefender/torrent-client/bitfield"
	"github.com/veggiedefender/torrent-client/peers"

	"github.com/veggiedefender/torrent-client/message"

	"github.com/veggiedefender/torrent-client/handshake"
)

// A Client is a TCP connection with a peer
type Client struct {
	Conn     packets.UDPConn
	Choked   bool
	Bitfield bitfield.Bitfield
	peer     peers.Peer
	infoHash [20]byte
	peerID   [20]byte
}

//LastSelection users could add more fields
type ClientSelection struct {
	lastSelectedPathSet pathselection.PathSet
}

//CustomPathSelectAlg this is where the user actually wants to implement its logic in
func (lastSel *ClientSelection) CustomPathSelectAlg(pathSet *pathselection.PathSet) (*pathselection.PathSet, error) {
	return pathSet.GetPathSmallHopCount(2), nil
}

func completeHandshake(conn packets.UDPConn, infohash, peerID [20]byte) (*handshake.Handshake, error) {
	// TODO: Add Deadline Methods
	// conn.SetDeadline(time.Now().Add(3 * time.Second))
	// defer conn.SetDeadline(time.Time{}) // Disable the deadline

	req := handshake.New(infohash, peerID)
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
	return res, nil
}

func recvBitfield(conn packets.UDPConn) (bitfield.Bitfield, error) {
	// conn.SetDeadline(time.Now().Add(5 * time.Second))
	// defer conn.SetDeadline(time.Time{}) // Disable the deadline

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
	fmt.Println(msg.Payload)
	return msg.Payload, nil
}

type MPClient struct {
}

func NewMPClient() *MPClient {
	return &MPClient{}
}

func (mp *MPClient) DialAndWaitForConnectBack(local string, peer peers.Peer, peerID, infoHash [20]byte) ([]*Client, error) {
	address, err := snet.ParseUDPAddr(peer.Addr)
	if err != nil {
		return nil, err
	}

	sel := ClientSelection{}

	mpSock := smp.NewMPPeerSock(local, address)
	err = mpSock.Listen()

	if err != nil {
		return nil, err
	}

	err = mpSock.Connect(&sel, nil)

	if err != nil {
		return nil, err
	}

	clients := make([]*Client, 0)
	for i, v := range mpSock.UnderlaySocket.GetConnections() {
		// Handshake only over first conn
		// TODO: Make this more flexible and don't stop all on error
		var bf bitfield.Bitfield
		if i == 0 {
			_, err = completeHandshake(v, infoHash, peerID)
			if err != nil {
				mpSock.UnderlaySocket.CloseAll()
				return nil, err
			}

			fmt.Println("Completed handshake")
			bf, err = recvBitfield(v)
			if err != nil {
				mpSock.UnderlaySocket.CloseAll()
				return nil, err
			}
		}
		c := Client{
			peer:     peer,
			peerID:   peerID,
			Conn:     v,
			infoHash: infoHash,
			Choked:   false,
			Bitfield: bf,
		}
		clients = append(clients, &c)
	}

	return clients, nil
}

// New connects with a peer, completes a handshake, and receives a handshake
// returns an err if any of those fail.
func New(peer peers.Peer, peerID, infoHash [20]byte) (*Client, error) {
	/*sock := socket.NewSocket("scion")
	conn, err := sock.Dial(peer.Addr, peer.Index)
	// conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Dial to %s done, starting handshake", peer.String())

	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	fmt.Println("Completed handshake")
	bf, err := recvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil*/
	return nil, nil
}

// Read reads and consumes a message from the connection
func (c *Client) Read() (*message.Message, error) {
	msg, err := message.Read(c.Conn)
	return msg, err
}

// SendRequest sends a Request message to the peer
func (c *Client) SendRequest(index, begin, length int) error {
	// fmt.Printf("Requesting %d, %d, %d\n", index, begin, length)
	req := message.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

// SendInterested sends an Interested message to the peer
func (c *Client) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendNotInterested sends a NotInterested message to the peer
func (c *Client) SendNotInterested() error {
	msg := message.Message{ID: message.MsgNotInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendUnchoke sends an Unchoke message to the peer
func (c *Client) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendHave sends a Have message to the peer
func (c *Client) SendHave(index int) error {
	msg := message.FormatHave(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}
