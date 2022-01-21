package main

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"io/ioutil"

	"github.com/anacrolix/tagflag"
	"github.com/netsys-lab/dht"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"

	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/server"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
)

var flags = struct {
	InPath            string `help:"Path to torrent file that should be processed"`
	OutPath           string `help:"Path where BitTorrent writes the downloaded file"`
	Peer              string `help:"Remote SCION address"`
	Seed              bool   `help:"Start BitTorrent in Seeder mode"`
	File              string `help:"Load the file to which the torrent of InPath refers. Only required if seed=true"`
	Local             string `help:"Local SCION address of the seeder"`
	NumPaths          int    `help:"Optional: Limit the number of paths the seeder uses to upload to each leecher. Per default 0, meaning the seeder aims to distribute paths in a fair manner to all leechers"`
	DialBackStartPort int    `help:"Optional: Start port of the connections the seeder uses to dial back to the leecher."`
	LogLevel          string `help:"Optional: Change log level"`
	EnableDht         bool   `help:"Optional: Run a dht network to announce peers"`
	DhtPort           int    `help:"Optional: Configure the port to run the dht network"`
	DhtBootstrapAddr  string `help:"Optional: SCION address of the dht network"`
	PrintMetrics      bool   `help:"Optional: Display per-path metrics at the end of the download. Only for seed=false"`
}{
	Seed:              false,
	NumPaths:          0,
	DialBackStartPort: 45000,
	LogLevel:          "INFO",
	PrintMetrics:      false,
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
	tf.PrintMetrics = flags.PrintMetrics
	if flags.Seed {
		log.Info("Loading file to RAM...")
		tf.Content, err = ioutil.ReadFile(flags.File)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Loaded file to RAM")
		// peer := fmt.Sprintf("%s:%d", flags.Peer, port)
		server, err := server.NewServer(flags.Local, &tf, "server", flags.NumPaths, flags.DialBackStartPort, &peerDiscoveryConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Created Server")

		err = server.ListenHandshake()
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
