package state

import (
	"path"
	bolt "github.com/etcd-io/bbolt"
	"time"
)

type State struct {
	db *bolt.DB
}

func NewState(filename string, directory string) (*State, error) {
	absoluteFilePath := path.Join(directory, filename)
	db, err := bolt.Open(absoluteFilePath, 0600, &bolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})
	if err != nil {
		return nil, err
	}
	return &State {
		db,
	}, nil
}
