package main

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/anacrolix/tagflag"
	"github.com/veggiedefender/torrent-client/server"
	"github.com/veggiedefender/torrent-client/torrentfile"
)

var flags = struct {
	InPath                      string
	OutPath                     string
	Peer                        string
	Seed                        bool
	File                        string
	Local                       string
	PathSelectionResponsibility string
}{
	Seed:                        true,
	PathSelectionResponsibility: "server",
}

func main() {
	log.SetLevel(log.DebugLevel)
	tagflag.Parse(&flags)

	log.Infof("Input %s, Output %s, Peer %s, seed %s, file %s\n", flags.InPath, flags.OutPath, flags.Peer, flags.Seed, flags.File)
	tf, err := torrentfile.Open(flags.InPath)
	if err != nil {
		log.Fatal(err)
	}

	if flags.Seed {
		log.Info("Loading file to RAM")
		tf.Content, err = ioutil.ReadFile(flags.File)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Loaded file to RAM")
		// peer := fmt.Sprintf("%s:%d", flags.Peer, port)
		server, err := server.NewServer(flags.Peer, &tf, flags.PathSelectionResponsibility)
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
		err = tf.DownloadToFile(flags.OutPath, flags.Peer, flags.Local, flags.PathSelectionResponsibility)
		if err != nil {
			log.Fatal(err)
		}
	}

}
