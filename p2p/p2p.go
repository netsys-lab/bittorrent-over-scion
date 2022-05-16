package p2p

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/netsys-lab/dht"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/netsys-lab/scion-path-discovery/pathselection"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"

	util "github.com/netsys-lab/bittorrent-over-scion/Utils"
	"github.com/netsys-lab/bittorrent-over-scion/client"
	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/dht_node"
	"github.com/netsys-lab/bittorrent-over-scion/message"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
)

// KiB number of bytes of a kibibyte
const KiB = 1024

// MaxBlockSize is the largest number of bytes a request can ask for
const MaxBlockSize = 256 * KiB

// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
const MaxBacklog = 5

// Torrent holds data required to download a torrent from a list of peers
type Torrent struct {
	sync.Mutex
	PeerSet                     peers.PeerSet
	PeerID                      [20]byte
	InfoHash                    [20]byte
	PieceHashes                 [][20]byte
	PieceLength                 int
	Length                      int
	Name                        string
	Local                       string
	PathSelectionResponsibility string
	Conns                       []packets.UDPConn
	DhtNode                     *dht_node.DhtNode
	DiscoveryConfig             *config.PeerDiscoveryConfig
	workQueue                   chan *pieceWork
	results                     chan *pieceResult
}

var peerMember interface{}

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read() // this call blocks
	if err != nil {
		return err
	}

	if msg == nil { // keep-alive
		return nil
	}

	switch msg.ID {
	case message.MsgUnchoke:
		state.client.Choked = false
		log.Debug("Got unchoke message")
	case message.MsgChoke:
		state.client.Choked = true
	case message.MsgHave:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)
	case message.MsgPiece:
		n, err := message.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	case message.MsgPort:
		log.Debug("got port message")
		client := state.client
		if !client.DiscoveryConfig.EnableDht || client.DhtNode == nil {
			log.Info("received port message but dht is not enabled")
			break
		}
		remoteDhtPort, err := message.ParsePort(msg)
		if err != nil {
			log.Info("received port message but couldn't parse message")
			break
		}
		dhtAddr, _ := snet.ParseUDPAddr(client.Peer.Addr)
		dhtAddr.Host.Port = int(remoteDhtPort)
		log.Debugf("sending dht ping to %s", dhtAddr)
		go client.DhtNode.Node.Ping(dhtAddr)
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // Disable the deadline
	for state.downloaded < pw.length {
		// If unchoked, send requests until we have enough unfulfilled requests
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := min(MaxBlockSize, pw.length)
				// Last block might be shorter than the typical block
				bytesDue := pw.length - state.requested
				if bytesDue < blockSize {
					blockSize = bytesDue
				}
				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func checkIntegrity(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("Index %d failed integrity check", pw.index)
	}
	return nil
}

func (t *Torrent) downloadQUIC(peer peers.Peer) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	conn, err := quic.DialAddr(peer.Addr, tlsConf, nil)
	if err != nil {
		log.Errorf("Failed to dial to %s", peer.Addr)
		log.Error(err)
		return
	}

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Errorf("Failed to open stream to to %s", peer.Addr)
		log.Error(err)
		return
	}
	wStream := &util.WrappedStream{
		Stream: stream,
		Local:  t.Local,
		Remote: peer.Addr,
	}

	c := client.Client{
		NetConn: wStream,
		Choked:  false,
		// Bitfield:        clients[0].Bitfield, TODO: Fetch from state
		Peer:            peer,
		InfoHash:        t.InfoHash,
		PeerID:          t.PeerID,
		DiscoveryConfig: t.DiscoveryConfig,
	}

	err = c.Handshake()
	if err != nil {
		log.Errorf("Failed to handshake to %s", peer.Addr)
		log.Error(err)
		return
	}
	for pw := range t.workQueue {
		if !c.Bitfield.HasPiece(pw.index) {
			t.workQueue <- pw // Put piece back on the queue
			continue
		}

		// Download the piece
		buf, err := attemptDownloadPiece(&c, pw)
		if err != nil {
			log.Warn("Error downloading piece", err)
			t.workQueue <- pw // Put piece back on the queue
			return
		}

		c.SendHave(pw.index)
		t.results <- &pieceResult{pw.index, buf}
	}
}

func (t *Torrent) downloadTCP(peer peers.Peer) {
	connection, err := net.Dial("tcp", peer.Addr)
	if err != nil {
		log.Errorf("Failed to dial to %s", peer.Addr)
		log.Error(err)
		return
	}

	c := client.Client{
		NetConn: connection,
		Choked:  false,
		// Bitfield:        clients[0].Bitfield, TODO: Fetch from state
		Peer:            peer,
		InfoHash:        t.InfoHash,
		PeerID:          t.PeerID,
		DiscoveryConfig: t.DiscoveryConfig,
	}

	err = c.Handshake()
	if err != nil {
		log.Errorf("Failed to handshake to %s", peer.Addr)
		log.Error(err)
		return
	}
	for pw := range t.workQueue {
		if !c.Bitfield.HasPiece(pw.index) {
			t.workQueue <- pw // Put piece back on the queue
			continue
		}

		// Download the piece
		buf, err := attemptDownloadPiece(&c, pw)
		if err != nil {
			log.Warn("Error downloading piece", err)
			t.workQueue <- pw // Put piece back on the queue
			return
		}

		c.SendHave(pw.index)
		t.results <- &pieceResult{pw.index, buf}
	}
}

func (t *Torrent) downloadSCION(peer peers.Peer) {
	mpC := client.NewMPClient()
	var clients []*client.Client
	var err error
	if t.PathSelectionResponsibility == "server" {
		clients, err = mpC.DialAndWaitForConnectBack(t.Local, peer, t.PeerID, t.InfoHash, t.DiscoveryConfig, t.DhtNode)
		if err != nil {
			log.Error(err)
			log.Errorf("Could not handshake with %s. Disconnecting", peer)
			return
		}

		for _, c := range clients {
			t.Lock()
			t.Conns = append(t.Conns, c.Conn)
			t.Unlock()
		}

		go func() {
			sock := mpC.GetSocket()
			for {
				conns := <-sock.OnConnectionsChange
				log.Debugf("Got new connections %d", len(conns))
				for i, v := range conns {

					if i == len(conns)-1 { // dial conn
						continue
					}

					connAlreadyOpen := false
					for _, cl := range clients {
						if cl.Conn.GetId() == v.GetId() {
							connAlreadyOpen = true
							log.Debugf("Got already open conn for id %s", v.GetId())
							break
						}
					}

					if !connAlreadyOpen {
						c := client.Client{
							Conn:            v,
							Choked:          false,
							Bitfield:        clients[0].Bitfield,
							Peer:            clients[0].Peer,
							InfoHash:        clients[0].InfoHash,
							PeerID:          clients[0].PeerID,
							DiscoveryConfig: clients[0].DiscoveryConfig,
						}
						clients = append(clients, &c)
						go func(c *client.Client) {
							log.Infof("Starting Download from new client")
							t.Lock()
							t.Conns = append(t.Conns, c.Conn)
							t.Unlock()
							c.Handshake()
							for pw := range t.workQueue {
								if !c.Bitfield.HasPiece(pw.index) {
									t.workQueue <- pw // Put piece back on the queue
									continue
								}

								// Download the piece
								buf, err := attemptDownloadPiece(c, pw)
								if err != nil {
									log.Warn("Error downloading piece, retrying in a new connection...", err)
									c.Conn.Close()
									c.Conn.SetId("TMP")
									t.workQueue <- pw // Put piece back on the queue
									return
								}

								c.SendHave(pw.index)
								t.results <- &pieceResult{pw.index, buf}
							}
						}(&c)
					}
				}

			}
		}()
	} else {
		log.Error("Client based pathselection not supported")
		return
	}

	log.Infof("Completed handshake with %s, got %d clients", peer, len(clients))
	log.Infof("Starting download...")
	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(c *client.Client) {
			for pw := range t.workQueue {
				if !c.Bitfield.HasPiece(pw.index) {
					t.workQueue <- pw // Put piece back on the queue
					continue
				}

				// Download the piece
				buf, err := attemptDownloadPiece(c, pw)
				if err != nil {
					log.Warn("Error downloading piece, retrying in a new connection...", err)
					c.Conn.Close()
					c.Conn.SetId("TMP")
					t.workQueue <- pw // Put piece back on the queue
					wg.Done()
					return
				}

				// fmt.Println(buf[:128])
				err = checkIntegrity(pw, buf)
				if err != nil {
					log.Fatalf("Piece #%d failed integrity check\n", pw.index)
					t.workQueue <- pw // Put piece back on the queue
					continue
				}

				c.SendHave(pw.index)
				t.results <- &pieceResult{pw.index, buf}
			}
			wg.Done()
		}(c)

	}
	wg.Wait()
	log.Debug("Return from startDownloadWorker")
	select {
	case p, ok := <-t.workQueue:
		if ok {
			log.Debug("Got not downloaded pieces, retrying...")
			t.workQueue <- p
			t.startDownloadWorker(peer)
		} else {
			log.Debug("No further pieces, done")
			return
		}
	default:
		log.Info("No further pieces, done")
		return
	}
}

func (t *Torrent) startDownloadWorker(peer peers.Peer) {
	if peer.Type == peers.PeerTypes.SCION {
		t.downloadSCION(peer)
	} else if peer.Type == peers.PeerTypes.TCP {
		t.downloadTCP(peer)
	} else {
		t.downloadQUIC(peer)
	}

}

func (t *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) calculatePieceSize(index int) int {
	begin, end := t.calculateBoundsForPiece(index)
	return end - begin
}

// Download downloads the torrent. This stores the entire file in memory.
func (t *Torrent) Download() ([]byte, error) {
	log.Infof("Starting download for %s", t.Name)
	// Init queues for workers to retrieve work and send results
	t.workQueue = make(chan *pieceWork, len(t.PieceHashes))
	t.results = make(chan *pieceResult)
	for index, hash := range t.PieceHashes {
		length := t.calculatePieceSize(index)
		t.workQueue <- &pieceWork{index, hash, length}
	}

	// Start workers
	for peer := range t.PeerSet.Peers {
		// time.Sleep(100 * time.Millisecond)
		go t.startDownloadWorker(peer)
	}

	// Collect results into a buffer until full
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-t.results
		begin, end := t.calculateBoundsForPiece(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		// numWorkers := runtime.NumGoroutine() - 1 // subtract 1 for main thread
		if donePieces%30 == 0 {
			percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
			log.Infof("(%0.2f%%) Downloaded piece #%d from %d", percent, res.index, len(t.PieceHashes))
		}

	}
	close(t.workQueue)
	for i, v := range t.Conns {
		log.Debugf("Checking con %d for metrics", i)
		m := v.GetMetrics()
		if m != nil {
			path := v.GetPath()
			if path != nil {
				log.Debugf("Got following bw over path %s", pathselection.PathToString(*path))
			}
			bwMbits := make([]int64, 0)
			for _, b := range m.ReadBandwidth {
				bwMbits = append(bwMbits, int64(float64(b*8)/1024/1024))
			}
			log.Debug(bwMbits)
		}
	}
	return buf, nil
}

func (t *Torrent) EnableDht(addr *snet.UDPAddr, peerPort uint16, infoHash [20]byte, startingNodes []dht.Addr) (*dht_node.DhtNode, error) {
	node, err := dht_node.New(addr, infoHash, startingNodes, peerPort, func(peer peers.Peer) {
		peerKnown := t.hasPeer(peer)
		log.Infof("received peer via dht: %s, peer already known: %t", peer, peerKnown)
		t.PeerSet.Add(peer)
		if !peerKnown { // dont start two worker for same peer
			go t.startDownloadWorker(peer)
		}
	})
	return node, err
}

func (t *Torrent) hasPeer(peer peers.Peer) bool {
	return t.PeerSet.Contains(peer)
}
