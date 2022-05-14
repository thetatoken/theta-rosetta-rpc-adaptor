package common

import (
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type LDBDatabase struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	quitLock sync.Mutex // Mutex protecting the quit channel access
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	logger.Infof("Allocated cache and file handles, cache: %v, handles: %v", cache, handles)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	return &LDBDatabase{
		fn: file,
		db: db,
	}, nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, errors.New("KeyNotFound")
		}
		return nil, err
	}
	return dat, nil
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil && err == leveldb.ErrNotFound {
		return errors.New("KeyNotFound")
	}
	return err
}

func (db *LDBDatabase) Close() {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	err := db.db.Close()
	if err == nil {
		logger.Infof("Database closed")
	} else {
		logger.Errorf("Failed to close database, err: %v", err)
	}
}

func (db *LDBDatabase) LDB() *leveldb.DB {
	return db.db
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}
