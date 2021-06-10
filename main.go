package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/veggiedefender/torrent-client/server"
	"github.com/veggiedefender/torrent-client/torrentfile"
)

func main() {
	inPath := os.Args[1]
	outPath := os.Args[2]
	peer := os.Args[3]
	seed := os.Args[4]
	file := os.Args[5]

	tf, err := torrentfile.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	if seed == "true" {
		tf.Content, err = ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		server, err := server.NewServer(peer, &tf)
		if err != nil {
			log.Fatal(err)
		}

		err = server.ListenHandshake()
		if err != nil {
			log.Fatal(err)
		}

		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = tf.DownloadToFile(outPath, peer)
		if err != nil {
			log.Fatal(err)
		}
	}

}
