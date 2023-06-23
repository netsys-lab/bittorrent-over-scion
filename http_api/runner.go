package http_api

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/netsys-lab/bittorrent-over-scion/config"
	"github.com/netsys-lab/bittorrent-over-scion/http_api/storage"
)

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

	peerDiscoveryConfig := config.DefaultPeerDisoveryConfig()
	t, err := torrent.TorrentFile.DownloadToFile(outPath, torrent.Peer, api.LocalHost, "server", &peerDiscoveryConfig)
	if err != nil {
		torrent.SaveState(api.Storage.DB, storage.StateFinishedFailed, err.Error())
		return
	}
	defer func() {
		if t != nil && t.DhtNode != nil {
			t.DhtNode.Close()
		}
	}()

	torrent.SaveState(api.Storage.DB, storage.StateFinishedSuccessfully, "")
}
