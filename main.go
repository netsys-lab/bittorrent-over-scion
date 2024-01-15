package main

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"context"
	"github.com/anacrolix/tagflag"
	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/http_api"
	"github.com/netsys-lab/bittorrent-over-scion/http_api/storage"
	"github.com/netsys-lab/bittorrent-over-scion/server"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
	"github.com/netsys-lab/dht"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

var flags = struct {
	InPath            string `help:"Path to torrent file that should be processed"`
	OutPath           string `help:"Path where BitTorrent writes the downloaded file"`
	Peer              string `help:"Remote SCION address"`
	Seed              bool   `help:"Start BitTorrent in Seeder mode"`
	File              string `help:"Load the file to which the torrent of InPath refers. Only required if seed=true"`
	Local             string `help:"Local SCION address of the seeder"`
	HttpApi           bool   `help:"Start HTTP API. This is a special mode, no direct downloading/seeding of specified file will happen."`
	HttpApiAddr       string `help:"Optional: Configure the IP and port the HTTP API will bind on (default 0.0.0.0:8000). Only for httpApi=true"`
	HttpApiMaxSize    int    `help:"Optional: Set the maximum size in bytes that is uploadable through HTTP API at once (all files in total, more specifically the maximum request body size, default ~128 MByte). Only for httpApi=true"`
	SeedStartPort     int    `help:"Optional: Start for ports used for the servers that seed individual torrents (unless explicitly specified). Only for httpApi=true"`
	NumPaths          int    `help:"Optional: Limit the number of paths the seeder uses to upload to each leecher. Per default 0, meaning the seeder aims to distribute paths in a fair manner to all leechers"`
	DialBackStartPort int    `help:"Optional: Start port of the connections the seeder uses to dial back to the leecher."`
	LogLevel          string `help:"Optional: Change log level"`
	EnableDht         bool   `help:"Optional: Run a dht network to announce peers"`
	DhtPort           int    `help:"Optional: Configure the port to run the dht network"`
	DhtBootstrapAddr  string `help:"Optional: SCION address of the dht network"`
	PrintMetrics      bool   `help:"Optional: Display per-path metrics at the end of the download. Only for seed=false"`
	ExportMetricsTo   string `help:"Optional: Export per-path metrics to a particular target, at the moment a csv file (e.g. /tmp/metrics.csv)"`
}{
	Seed:              false,
	HttpApi:           false,
	HttpApiAddr:       "0.0.0.0:8000",
	HttpApiMaxSize:    128 * 1000000, // 128 MByte
	SeedStartPort:     44000,
	NumPaths:          0,
	DialBackStartPort: 45000,
	LogLevel:          "INFO",
	PrintMetrics:      false,
	ExportMetricsTo:   "http://19-ffaa:1:c3f,141.44.25.148:80/btmetrics",
}

func setLogging(loglevel string) {

	switch loglevel {
	case "TRACE":
		log.SetLevel(log.TraceLevel)
		break
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
		break
	case "INFO":
		log.SetLevel(log.InfoLevel)
		break
	case "WARN":
		log.SetLevel(log.WarnLevel)
		break
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
		break
	case "FATAL":
		log.SetLevel(log.FatalLevel)
		break
	}
}

func main() {
	tagflag.Parse(&flags)
	setLogging(flags.LogLevel)

	if flags.HttpApi {
		log.Info("Starting in HTTP API mode...")

		log.Info("[HTTP API] Initializing storage...")
		storage_ := &storage.Storage{DbBackend: storage.Sqlite}
		err := storage_.Init("file::memory:?cache=shared") // in-memory SQLite database
		if err != nil {
			log.Fatal(err)
			return
		}

		log.Info("[HTTP API] Loading existing torrent tasks from storage...")
		api := http_api.HttpApi{
			LocalAddr:          flags.HttpApiAddr,
			MaxRequestBodySize: flags.HttpApiMaxSize,
			EnableDht:          flags.EnableDht, //TODO make this configurable per torrent?
			DhtPort:            uint16(flags.DhtPort),
			DhtBootstrapAddr:   flags.DhtBootstrapAddr,
			ScionLocalHost:     flags.Local,
			NumPaths:           flags.NumPaths,
			DialBackStartPort:  uint16(flags.DialBackStartPort),
			SeedStartPort:      uint16(flags.SeedStartPort),
			Storage:            storage_,
		}
		err = api.LoadFromStorage()
		if err != nil {
			log.Fatal(err)
			return
		}

		log.Info("[HTTP API] Starting web server...")
		err = api.ListenAndServe()
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	log.Infof("Input %s, Output %s, Peer %s, seed %t, file %s", flags.InPath, flags.OutPath, flags.Peer, flags.Seed, flags.File)

	peerDiscoveryConfig := config.DefaultPeerDisoveryConfig()

	peerDiscoveryConfig.EnableDht = flags.EnableDht
	dhtAddr, err := snet.ParseUDPAddr(flags.DhtBootstrapAddr)
	if err == nil {
		peerDiscoveryConfig.DhtNodes = []dht.Addr{dht.NewAddr(*dhtAddr)}
	}
	if flags.DhtPort > 0 {
		peerDiscoveryConfig.DhtPort = uint16(flags.DhtPort)
	}

	tf, err := torrentfile.Open(flags.InPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("TorrentFile{Announce: \"%s\", Length: %d, Name: \"%s\", PieceLength: %d}", tf.Announce, tf.Length, tf.Name, tf.PieceLength)

	tf.PrintMetrics = flags.PrintMetrics
	if flags.Seed {
		log.Info("Loading file to RAM...")
		tf.Content, err = ioutil.ReadFile(flags.File)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Loaded file to RAM")
		// peer := fmt.Sprintf("%s:%d", flags.Peer, port)
		conf := server.ServerConfig{
			LAddr:                       flags.Local,
			TorrentFile:                 &tf,
			PathSelectionResponsibility: "server",
			NumPaths:                    flags.NumPaths,
			DialBackPort:                flags.DialBackStartPort,
			DiscoveryConfig:             &peerDiscoveryConfig,
			ExportMetricsTarget:         flags.ExportMetricsTo,
		}
		server_, err := server.NewServer(&conf)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Created Server")

		err = server_.ListenHandshake(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		if err != nil {
			log.Fatal(err)
		}
	} else {
		t, err := tf.DownloadToFile(flags.OutPath, flags.Peer, flags.Local, "server", &peerDiscoveryConfig)
		if err != nil {
			log.Fatal(err)
		}
		if t.DhtNode != nil {
			t.DhtNode.Close()
		}
	}

}
