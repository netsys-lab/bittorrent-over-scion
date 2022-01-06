package main

// SPDX-FileCopyrightText:  2019 NetSys Lab
// SPDX-License-Identifier: GPL-3.0-only

import (
	"io/ioutil"
	"time"

	"github.com/anacrolix/tagflag"
	"github.com/netsys-lab/dht"
	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"

	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/server"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
)

var flags = struct {
	InPath                      string
	OutPath                     string
	Peer                        string
	Seed                        bool
	File                        string
	Local                       string
	PathSelectionResponsibility string
	NumPaths                    int
	DialBackStartPort           int
	LogLevel                    string
	EnableDht                   bool
	DhtPort                     int
	DhtBootstrapAddr            string
	PrintMetrics                bool
	KeepAlive                   bool // only effects leecher, testing purpose only TODO: remove
}{
	Seed:                        false,
	PathSelectionResponsibility: "server",
	NumPaths:                    0,
	DialBackStartPort:           45000,
	LogLevel:                    "INFO",
	PrintMetrics:                false,
	KeepAlive:                   false,
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
		server, err := server.NewServer(flags.Peer, &tf, flags.PathSelectionResponsibility, flags.NumPaths, flags.DialBackStartPort, &peerDiscoveryConfig)
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
		t, err := tf.DownloadToFile(flags.OutPath, flags.Peer, flags.Local, flags.PathSelectionResponsibility, &peerDiscoveryConfig)
		if err != nil {
			log.Fatal(err)
		}
		if flags.KeepAlive {
			time.Sleep(1 * time.Hour)
		}
		if t.DhtNode != nil {
			t.DhtNode.Close()
		}
	}

}
