package state

import (
	"github.com/theQRL/zond/db"
)

type State struct {
	db *db.DB
}

func (s *State) DB() *db.DB {
	return s.db
}

func NewState(directory string, filename string) (*State, error) {
	d, err := db.NewDB(directory, filename)
	if err != nil {
		return nil, err
	}
	return &State {
		d,
	}, nil
}
