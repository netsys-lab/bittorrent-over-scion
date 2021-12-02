package torrentfile
// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"

	"github.com/netsys-lab/dht"
	"github.com/netsys-lab/scion-path-discovery/packets"

	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
)

// Port to listen on
const Port uint16 = 6881

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce     string
	Nodes        []dht.Addr
	InfoHash     [20]byte
	PieceHashes  [][20]byte
	PieceLength  int
	Length       int
	Name         string
	Content      []byte
	PrintMetrics bool
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string          `bencode:"announce"`
	Nodes    [][]interface{} `bencode:"nodes"`
	Info     bencodeInfo     `bencode:"info"`
}

// DownloadToFile downloads a torrent and writes it to a file
// This function leeches all pieces of a torrent but never starts seeding. When DHT is enabled in the
// PeerDiscoveryConfig, the peer will still announce its presence to receive other peers. We therefore announces our
// presence on a port we are not listening to.
func (t *TorrentFile) DownloadToFile(path string, peer string, local string, pathSelectionResponsibility string, pc *config.PeerDiscoveryConfig) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	targetPeers := peers.NewPeerSet(0)
	if peer != "" {
		_, err := snet.ParseUDPAddr(peer)
		if err != nil {
			log.Fatal(err)
		}
		// pAddr, _ := net.ResolveTCPAddr("tcp", peer)

		p := peers.Peer{
			Addr:  peer,
			Index: 0,
		}
		targetPeers.Add(p)

	}

	torrent := p2p.Torrent{
		PeerSet:                     targetPeers,
		PeerID:                      peerID,
		InfoHash:                    t.InfoHash,
		PieceHashes:                 t.PieceHashes,
		PieceLength:                 t.PieceLength,
		Length:                      t.Length,
		Name:                        t.Name,
		Local:                       local,
		PathSelectionResponsibility: pathSelectionResponsibility,
		DiscoveryConfig:             pc,
		Conns:                       make([]packets.UDPConn, 0),
	}

	if pc.EnableDht {
		peerAddr, err := snet.ParseUDPAddr(local)
		peerPort := uint16(peerAddr.Host.Port)
		nodeAddr := *peerAddr
		nodeAddr.Host.Port = int(pc.DhtPort)
		torrent.DhtNode, err = torrent.EnableDht(nodeAddr.Host, peerPort, t.InfoHash, append(t.Nodes, pc.DhtNodes...))
		if err != nil {
			log.Println("could not enable dht")
		}
	}

	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	if torrent.DhtNode != nil {
		torrent.DhtNode.Close()
	}

	if t.PrintMetrics {
		// TODO: Implement metrics
		// for i,v := range torrent.Conns {
		//	log.Infof("Average download bandwidth ")
		//}
	}

	log.Infof("Writing output file %s", path)
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	log.Infof("Done writing output file, download complete")
	return nil
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Open for file failed %s", path)
		fmt.Println(err)
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile()
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	nodes, err := bto.parseDhtNodes()
	if err != nil {
		return TorrentFile{}, err
	}

	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
		Nodes:       *nodes,
	}
	return t, nil
}

// parseDhtNodes receive DHT Nodes from torrent file as specified in BEP 5 Torrent File Extension
func (bto *bencodeTorrent) parseDhtNodes() (*[]dht.Addr, error) {
	nodes := make([]dht.Addr, len(bto.Nodes))
	for i, btoNode := range bto.Nodes {
		if len(btoNode) != 2 {
			return nil, errors.New("invalid node format")
		}
		host, hostOk := btoNode[0].(string)
		port, portOk := btoNode[1].(int64)
		if !hostOk || !portOk {
			return nil, errors.New("invalid node address")
		}
		val := fmt.Sprintf("%s:%d", host, port)
		addr, err := snet.ParseUDPAddr(val)
		if err != nil {
			return nil, errors.New("invalid node address")
		}
		nodes[i] = dht.NewAddr(*addr)
	}
	return &nodes, nil
}
