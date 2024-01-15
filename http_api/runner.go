package http_api

import (
	"context"
	"crypto/rand"
	"errors"
	util "github.com/netsys-lab/bittorrent-over-scion/Utils"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/bittorrent-over-scion/server"
	"github.com/netsys-lab/dht"
	"github.com/netsys-lab/scion-path-discovery/packets"
	"github.com/scionproto/scion/go/lib/snet"
	"io/ioutil"
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

func resetSeeder(torrent *storage.Torrent) {
	torrent.SeedOnCompletion = false
	torrent.CancelFunc = nil
	torrent.SeedAddr = ""
}

func (api *HttpApi) RunLeecher(ctx context.Context, torrent *storage.Torrent) {
	// this is just a simple test for cancellation, a code snippet to be used later
	if errors.Is(ctx.Err(), context.Canceled) {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedCancelled, "cancelled by user")
		return
	}

	torrent.SaveState(api.Storage.DB, storage.StateRunning, "")

	// get path of first file
	//TODO support multiple files & directory trees
	outPath := torrent.GetFileDir(api.Storage.FS)
	if len(torrent.Files[0].Path) == 0 {
		outPath = filepath.Join(outPath, "file")
	} else {
		outPath = filepath.Join(outPath, torrent.Files[0].Path)
	}

	// configure peer discovery
	peerDiscoveryConfig := config.DefaultPeerDisoveryConfig()
	peerDiscoveryConfig.EnableDht = api.EnableDht

	// generate random peer ID
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
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
		Local:                       api.ScionLocalHost,
		PathSelectionResponsibility: "server",
		DiscoveryConfig:             &peerDiscoveryConfig,
		Conns:                       make([]packets.UDPConn, 0),
	}

	if peerDiscoveryConfig.EnableDht {
		peerAddr, err := snet.ParseUDPAddr(api.ScionLocalHost)
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

	// start seeding if told to do so
	if torrent.SeedOnCompletion {
		api.RunSeeder(ctx, torrent)
	}
}

func (api *HttpApi) RunSeeder(ctx context.Context, torrent *storage.Torrent) {
	// this is just a simple test for cancellation, a code snippet to be used later
	if errors.Is(ctx.Err(), context.Canceled) {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedCancelled, "cancelled by user")
		return
	}

	outPath := torrent.GetFileDir(api.Storage.FS)
	if len(torrent.Files[0].Path) == 0 {
		outPath = filepath.Join(outPath, "file")
	} else {
		outPath = filepath.Join(outPath, torrent.Files[0].Path)
	}

	// load file into RAM
	var err error
	torrent.TorrentFile.Content, err = ioutil.ReadFile(outPath)
	if err != nil {
		// turn off seeding so that the user can try again to reactivate it
		resetSeeder(torrent)

		torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "Seeding failed: "+err.Error())
		return
	}

	// configure peer discovery
	peerDiscoveryConfig := config.DefaultPeerDisoveryConfig()
	peerDiscoveryConfig.EnableDht = api.EnableDht
	dhtAddr, err := snet.ParseUDPAddr(api.DhtBootstrapAddr)
	if err == nil {
		peerDiscoveryConfig.DhtNodes = []dht.Addr{dht.NewAddr(*dhtAddr)}
	}
	if api.DhtPort > 0 {
		peerDiscoveryConfig.DhtPort = uint16(api.DhtPort)
	}

	// take next automatic port if needed
	seedPort := torrent.SeedPort
	if seedPort == 0 {
		seedPort = api.SeedStartPort
		for {
			_, exists := api.usedUdpPorts[seedPort]
			if !exists || api.usedUdpPorts[seedPort] == false {
				break
			}

			seedPort += 1
		}
		api.usedUdpPorts[seedPort] = true
		defer func() {
			// the underlying implementation of the Listen/ListenHandshake functions do not consider closing any connections...
			// therefore currently, due to the SCION dispatcher not allowing to register the same port multiple times, a new port must be used
			//TODO close SCION connection somewhere so that the port is reusable in the same process
			//api.usedUdpPorts[seedPort] = false
		}()
	}

	// set host
	var localAddr *snet.UDPAddr
	if api.ScionLocalHost != "" {
		localAddr, err = snet.ParseUDPAddr(api.ScionLocalHost)
	} else {
		localAddr, err = util.GetDefaultLocalAddr() //TODO currently wastes a port
	}
	if err != nil {
		// turn of seeding so that the user can try again to reactivate it
		resetSeeder(torrent)

		torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "Seeding failed: "+err.Error())
		return
	}

	// set port
	localAddr.Host.Port = int(seedPort)
	torrent.SeedAddr = localAddr.String()

	// dial back port selection is a bit weird on the server implementation, we just use the DialBackStartPort
	// configured by CLI plus the offset of the selected seeding port from the seed start port for now
	dialBackStartPort := api.DialBackStartPort + (seedPort - api.SeedStartPort)

	// peer := fmt.Sprintf("%s:%d", flags.Peer, port)
	conf := server.ServerConfig{
		LAddr:                       torrent.SeedAddr,
		TorrentFile:                 torrent.TorrentFile,
		PathSelectionResponsibility: "server",
		NumPaths:                    api.NumPaths,
		DialBackPort:                int(dialBackStartPort),
		DiscoveryConfig:             &peerDiscoveryConfig,
		ExportMetricsTarget:         "",
	}
	server_, err := server.NewServer(&conf)
	if err != nil {
		// turn of seeding so that the user can try again to reactivate it
		resetSeeder(torrent)

		torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "seeding failed: "+err.Error())
		return
	}

	torrent.SaveState(api.Storage.DB, storage.StateSeeding, "")

	err = server_.ListenHandshake(ctx)
	if err != nil {
		// turn of seeding so that the user can try again to reactivate it
		resetSeeder(torrent)

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "")
		} else {
			torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "seeding failed: "+err.Error())
		}
		return
	}

}
