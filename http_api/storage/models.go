package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	"github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path/filepath"
	"time"
)

type State int

func (state State) String() string {
	return [...]string{
		"not_started_yet",
		"running",
		"failed",
		"completed",
		"cancelled",
		"seeding",
	}[state]
}

func (state State) IsFinished() bool {
	switch state {
	case StateFinishedFailed, StateFinishedSuccessfully, StateFinishedCancelled, StateSeeding:
		return true
	}
	return false
}

const (
	StateNotStartedYet        State = 0
	StateRunning                    = 1
	StateFinishedFailed             = 2
	StateFinishedSuccessfully       = 3
	StateFinishedCancelled          = 4
	StateSeeding                    = 5
)

var StringToState = map[string]State{
	"not_started_yet": StateNotStartedYet,
	"running":         StateRunning,
	"failed":          StateFinishedFailed,
	"completed":       StateFinishedSuccessfully,
	"cancelled":       StateFinishedCancelled,
	"seeding":         StateSeeding,
}

type File struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	TorrentID uint64 `json:"-"`

	Path   string `json:"path"`
	Length uint64 `json:"length"`

	Progress uint64 `gorm:"-" json:"progress"` // in bytes
}

type Tracker struct {
	ID  uint64 `gorm:"primaryKey" json:"id"`
	URL string `gorm:"uniqueIndex" json:"url"`
}

type Peer struct {
	ID        uint64 `gorm:"primaryKey"`
	TorrentID uint64

	Address string
}

func (peer *Peer) MarshalJSON() ([]byte, error) {
	return json.Marshal(peer.Address)
}

type TorrentMetrics struct {
	ReadBandwidth    int64 `json:"rx"`
	WrittenBandwidth int64 `json:"tx"`
	NumConns         uint  `json:"numConns"`
	NumPaths         int   `json:"numPaths"`
}

type Torrent struct {
	/* persisted in database */

	// gorm.Model without DeletedAt
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	// own attributes
	FriendlyName     string `json:"name"`
	Peers            []Peer `json:"peers"`
	SeedOnCompletion bool   `json:"seedOnCompletion"`
	SeedPort         uint16 `json:"seedPort"`
	EnableDht        bool   `json:"enableDht"`
	EnableTrackers   bool   `json:"enableTrackers"`
	State            State  `json:"-"`
	Status           string `json:"status"`
	Files            []File `json:"files"`
	RawTorrentFile   []byte `json:"-"`

	/* only in memory */
	Metrics     *TorrentMetrics          `gorm:"-" json:"metrics"`
	TorrentFile *torrentfile.TorrentFile `gorm:"-" json:"-"`
	P2pTorrent  *p2p.Torrent             `gorm:"-" json:"-"`
	CancelFunc  *context.CancelFunc      `gorm:"-" json:"-"`
	SeedAddr    string                   `gorm:"-" json:"seedAddr"`
	PeerSet     peers.PeerSet            `gorm:"-" json:"-"`
}

func (torrent *Torrent) MarshalJSON() ([]byte, error) {
	type Alias Torrent

	numDownloadedPieces := 0
	if torrent.P2pTorrent != nil {
		numDownloadedPieces = torrent.P2pTorrent.NumDownloadedPieces
	}

	return json.Marshal(&struct {
		State               string `json:"state"`
		NumPieces           int    `json:"numPieces"`
		NumDownloadedPieces int    `json:"numDownloadedPieces"`
		PieceLength         int    `json:"pieceLength"`
		*Alias
	}{
		State:               torrent.State.String(),
		NumPieces:           len(torrent.TorrentFile.PieceHashes),
		NumDownloadedPieces: numDownloadedPieces,
		PieceLength:         torrent.TorrentFile.PieceLength,
		Alias:               (*Alias)(torrent),
	})
}

func (torrent *Torrent) SaveState(db *gorm.DB, state State, status string) {
	oldState := torrent.State
	torrent.State = state
	torrent.Status = status
	result := db.Save(torrent)
	if result.Error != nil {
		log.Error(result.Error)
	}
	log.Infof("[HTTP API] State of torrent %d changed from '%s' to '%s'!", torrent.ID, oldState.String(), torrent.State.String())
}

func (torrent *Torrent) GetFileDir(fs *FS) string {
	return filepath.Join(fs.FileDir, fmt.Sprintf("%d", torrent.ID))
}
