package main

import (
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/anacrolix/tagflag"
	"github.com/veggiedefender/torrent-client/server"
	"github.com/veggiedefender/torrent-client/torrentfile"
)

var flags = struct {
	InPath  string
	OutPath string
	Peer    string
	Seed    bool
	File    string
	Local   string
	numCons int
}{
	Seed: true,
}

func main() {
	log.SetLevel(log.DebugLevel)
	tagflag.Parse(&flags)

	fmt.Printf("Input %s, Output %s, Peer %s, seed %s, file %s\n", flags.InPath, flags.OutPath, flags.Peer, flags.Seed, flags.File)
	tf, err := torrentfile.Open(flags.InPath)
	if err != nil {
		log.Fatal(err)
	}

	if flags.Seed {
		fmt.Println("Loading file to RAM")
		tf.Content, err = ioutil.ReadFile(flags.File)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Loaded file to RAM")
		i := 0
		startPort := 42423
		for i < flags.numCons {
			if i < flags.numCons-1 {
				go func(port int) {
					peer := fmt.Sprintf("%s:%d", flags.Peer, port)
					server, err := server.NewServer(peer, &tf)
					if err != nil {
						log.Fatal(err)
					}

					fmt.Println("Created Server")

					err = server.ListenHandshake()
					if err != nil {
						log.Fatal(err)
					}
				}(startPort + i)
			} else {
				peer := fmt.Sprintf("%s:%d", flags.Peer, startPort+i)
				server, err := server.NewServer(peer, &tf)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println("Created Server")

				err = server.ListenHandshake()
				if err != nil {
					log.Fatal(err)
				}
			}
			i++
		}

		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = tf.DownloadToFile(flags.OutPath, flags.Peer, flags.numCons, flags.Local)
		if err != nil {
			log.Fatal(err)
		}
	}

}
