package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/veggiedefender/torrent-client/server"
	"github.com/veggiedefender/torrent-client/torrentfile"
)

func main() {
	inPath := os.Args[1]
	outPath := os.Args[2]
	peer := os.Args[3]
	seed := os.Args[4]
	file := os.Args[5]
	numCons := os.Args[6]
	nCons, err := strconv.Atoi(numCons)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Input %s, Output %s, Peer %s, seed %s, file %s\n", inPath, outPath, peer, seed, file)
	tf, err := torrentfile.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	if seed == "true" {
		fmt.Println("Loading file to RAM")
		tf.Content, err = ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Loaded file to RAM")
		i := 0
		startPort := 42423
		for i < nCons {
			if i < nCons-1 {
				go func(port int) {
					peer := fmt.Sprintf("%s:%d", peer, port)
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
				peer := fmt.Sprintf("%s:%d", peer, startPort+i)
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
		err = tf.DownloadToFile(outPath, peer, nCons)
		if err != nil {
			log.Fatal(err)
		}
	}

}
