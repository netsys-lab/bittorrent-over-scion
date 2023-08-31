package http_api

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/scionproto/scion/go/lib/snet"
	"os"
	"path/filepath"
	"time"

	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/http_api/storage"
	log "github.com/sirupsen/logrus"
)

func runMetricsCollector(torrent *storage.Torrent, stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			var paths map[string]struct{}
			torrent.Metrics.NumConns = 0
			for _, conn := range torrent.P2pTorrent.Conns {
				metrics := conn.GetMetrics()

				// collect paths along the way
				if conn.GetPath() != nil {
					path := (*conn.GetPath()).Path()
					key := path.String()
					log.Debug(key)

					// path deduplication
					_, pathExistsAlready := paths[key]
					if !pathExistsAlready {
						paths[key] = struct{}{}
					}
				}

				if len(metrics.ReadBandwidth) > 0 {
					torrent.Metrics.ReadBandwidth += metrics.ReadBandwidth[len(metrics.ReadBandwidth)-1]
				}
				if len(metrics.WrittenBandwidth) > 0 {
					torrent.Metrics.WrittenBandwidth += metrics.WrittenBandwidth[len(metrics.WrittenBandwidth)-1]
				}
				torrent.Metrics.NumConns += 1
			}

			//TODO multiple files per torrent
			torrent.Files[0].Progress = uint64(torrent.P2pTorrent.NumDownloadedPieces * torrent.TorrentFile.PieceLength)
			// cap progress so it cannot be larger than the file itself
			if torrent.Files[0].Progress > torrent.Files[0].Length {
				torrent.Files[0].Progress = torrent.Files[0].Length
			}

			time.Sleep(3 * time.Second)
		}
	}
}

func (api *HttpApi) RunTorrent(ctx context.Context, torrent *storage.Torrent) {
	// this is just a simple test for cancellation, a code snippet to be used later
	if errors.Is(ctx.Err(), context.Canceled) {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedCancelled, "cancelled by user")
		return
	}

	torrent.SaveState(api.Storage.DB, storage.StateRunning, "")

	// create output directory if not existing
	outPath := torrent.GetFileDir(api.Storage.FS)
	err := os.MkdirAll(outPath, os.ModePerm)
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}

	// add file name
	//TODO support multiple files & directory trees
	if len(torrent.Files[0].Path) == 0 {
		outPath = filepath.Join(outPath, "file")
	} else {
		outPath = filepath.Join(outPath, torrent.Files[0].Path)
	}

	// configure peer discovery
	peerDiscoveryConfig := config.DefaultPeerDisoveryConfig()

	// generate random peer ID
	var peerID [20]byte
	_, err = rand.Read(peerID[:])
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}

	// make target peers
	//TODO allow multiple peers
	targetPeers := peers.NewPeerSet(0)
	_, err = snet.ParseUDPAddr(torrent.Peer)
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}
	p := peers.Peer{
		Addr:  torrent.Peer,
		Index: 0,
	}
	targetPeers.Add(p)

	torrent.P2pTorrent = &p2p.Torrent{
		PeerSet:                     targetPeers,
		PeerID:                      peerID,
		InfoHash:                    torrent.TorrentFile.InfoHash,
		PieceHashes:                 torrent.TorrentFile.PieceHashes,
		PieceLength:                 torrent.TorrentFile.PieceLength,
		Length:                      torrent.TorrentFile.Length,
		Name:                        torrent.TorrentFile.Name,
		Local:                       api.LocalHost,
		PathSelectionResponsibility: "server",
		DiscoveryConfig:             &peerDiscoveryConfig,
		Conns:                       make([]packets.UDPConn, 0),
	}

	if peerDiscoveryConfig.EnableDht {
		peerAddr, err := snet.ParseUDPAddr(api.LocalHost)
		peerPort := uint16(peerAddr.Host.Port)
		nodeAddr := peerAddr.Copy()
		nodeAddr.Host.Port = int(peerDiscoveryConfig.DhtPort)
		torrent.P2pTorrent.DhtNode, err = torrent.P2pTorrent.EnableDht(
			nodeAddr,
			peerPort,
			torrent.TorrentFile.InfoHash,
			append(torrent.TorrentFile.Nodes, peerDiscoveryConfig.DhtNodes...),
		)
		if err != nil {
			log.Warning("could not enable dht")
		}
		defer func() {
			if torrent.P2pTorrent.DhtNode != nil {
				torrent.P2pTorrent.DhtNode.Close()
			}
		}()
	}

	// start metrics collection for this torrent
	stopMetricsCollection := make(chan bool)
	go runMetricsCollector(torrent, stopMetricsCollection)

	// download single file
	//TODO multiple files per torrent
	buf, err := torrent.P2pTorrent.Download()
	stopMetricsCollection <- true
	time.Sleep(4 * time.Second) // to not have race conditions when writing status
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}

	// write single file to disk
	//TODO multiple files per torrent
	outFile, err := os.Create(outPath)
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {
			log.Error(err)
		}
	}(outFile)
	_, err = outFile.Write(buf)
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}

	//TODO multiple files per torrent
	torrent.Files[0].Progress = torrent.Files[0].Length

	torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "")
}
