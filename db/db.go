package db

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"path"
	"sync"
	"time"
)

type DB struct {
	db *bbolt.DB

	filename string

	Lock     sync.RWMutex
	exitLock sync.Mutex
}

func NewDB(directory string, filename string) (*DB, error) {
	dbDir := path.Join(directory, filename)
	db, err := bbolt.Open(dbDir, 0600, &bbolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})

	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("DB")) // TODO: Move this Bucket name to appropriate place
		if err != nil {
			return fmt.Errorf("could not create DB %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &DB{
		filename: filename,
		db:       db,
	}, nil
}

func (db *DB) Get(key []byte) ([]byte, error) {
	defer db.Lock.RUnlock()
	db.Lock.RLock()

	var value []byte

	err := db.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		value = b.Get(key)
		if value == nil {
			return errors.New("key not found")
		}
		return nil
	})

	return value, err
}

func (db *DB) GetFromBucket(key []byte, bucket []byte) ([]byte, error) {
	defer db.Lock.RUnlock()
	db.Lock.RLock()

	var value []byte

	err := db.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return errors.New("bucket not found")
		}
		value = b.Get(key)
		if value == nil {
			return errors.New("key not found")
		}
		return nil
	})

	return value, err
}

func (db *DB) Close() {
	db.exitLock.Lock()
	defer db.exitLock.Unlock()

	err := db.db.Close()
	if err == nil {
		log.Info("BoltDB Closed")
	} else {
		log.Error("Failed to close BoltDB", "err", err)
	}
}

func (db *DB) DB() *bbolt.DB {
	return db.db
}
