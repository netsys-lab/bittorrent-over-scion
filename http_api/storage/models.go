package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path/filepath"
	"time"
)

type State int

func (state State) String() string {
	return [...]string{
		"not started yet",
		"running",
		"failed",
		"completed",
		"cancelled",
	}[state]
}

func (state State) IsFinished() bool {
	switch state {
	case StateFinishedFailed, StateFinishedSuccessfully, StateFinishedCancelled:
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
)

type File struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	TorrentID uint64 `json:"-"`

	Path   string `json:"path"`
	Length uint64 `json:"length"`
}

type Torrent struct {
	/* persisted in database */

	// gorm.Model without DeletedAt
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	// own attributes
	FriendlyName   string `json:"name"`
	Peer           string `json:"peer"`
	State          State  `json:"-"`
	Status         string `json:"status"`
	Files          []File `json:"files"`
	RawTorrentFile []byte `json:"-"`

	/* only in memory */
	TorrentFile *torrentfile.TorrentFile `gorm:"-" json:"-"`
	P2pTorrent  *p2p.Torrent             `gorm:"-" json:"-"`
	CancelFunc  *context.CancelFunc      `gorm:"-" json:"-"`
}

func (torrent *Torrent) MarshalJSON() ([]byte, error) {
	type Alias Torrent
	return json.Marshal(&struct {
		State string `json:"state"`
		*Alias
	}{
		State: torrent.State.String(),
		Alias: (*Alias)(torrent),
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