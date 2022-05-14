package common

import (
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
)

type Store interface {
	Put(key common.Bytes, value interface{}) error
	Delete(key common.Bytes) error
	Get(key common.Bytes, value interface{}) error
}

// NewKVStore create a new instance of KVStore.
func NewKVStore(db *LDBDatabase) Store {
	return &KVStore{db}
}

// KVStore a Database wrapped object.
type KVStore struct {
	db *LDBDatabase
}

// Put upserts key/value into DB
func (store *KVStore) Put(key common.Bytes, value interface{}) error {
	encodedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	return store.db.Put(key, encodedValue)
}

// Delete deletes key entry from DB
func (store *KVStore) Delete(key common.Bytes) error {
	return store.db.Delete(key)
}

// Get looks up DB with key and returns result into value (passed by reference)
func (store *KVStore) Get(key common.Bytes, value interface{}) error {
	encodedValue, err := store.db.Get(key)
	if err != nil {
		return err
	}
	return rlp.DecodeBytes(encodedValue, value)
}
