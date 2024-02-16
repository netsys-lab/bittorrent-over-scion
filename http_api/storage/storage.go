package storage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type DbBackend int

const (
	Sqlite DbBackend = 0
)

type FS struct {
	TorrentFileDir string
	FileDir        string
}

type Storage struct {
	DbBackend DbBackend
	DB        *gorm.DB
	FS        *FS
}

func (s *Storage) Init(dsn string) error {
	// initialize database
	config := &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
	}
	var err error
	switch s.DbBackend {
	case Sqlite:
		s.DB, err = gorm.Open(sqlite.Open(dsn), config)
	default:
		return errors.New("unknown storage backend")
	}
	if err != nil {
		return err
	}
	err = s.DB.AutoMigrate(
		&Torrent{},
		&File{},
		&Peer{},
		&Tracker{},
	)
	if err != nil {
		return err
	}

	// initialize home directory
	s.FS = &FS{}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	baseDir := filepath.Join(homeDir, ".bittorrent-over-scion")
	s.FS.TorrentFileDir = filepath.Join(baseDir, "torrents")
	err = os.MkdirAll(s.FS.TorrentFileDir, os.ModePerm)
	if err != nil {
		return err
	}
	s.FS.FileDir = filepath.Join(baseDir, "files")
	err = os.MkdirAll(s.FS.FileDir, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
