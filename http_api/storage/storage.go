package storage

import (
	"errors"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"os"
)

type DbBackend int

const (
	Sqlite DbBackend = 0
)

type FS struct {
	FileDir string
}

type Storage struct {
	DbBackend DbBackend
	DB        *gorm.DB
	FS        *FS
}

func (s *Storage) Init(fileDir string, dsn string) error {
	// initialize file system
	s.FS = &FS{
		FileDir: fileDir,
	}
	err := os.MkdirAll(s.FS.FileDir, os.ModePerm)
	if err != nil {
		return err
	}

	// initialize database
	config := &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
	}
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

	return nil
}
