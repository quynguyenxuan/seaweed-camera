//go:build foundationdb
// +build foundationdb

// Package foundationdb provides a filer store implementation using FoundationDB as the backend.
//
// IMPORTANT DESIGN NOTE - DeleteFolderChildren and Transaction Limits:
//
// FoundationDB imposes strict transaction limits:
//   - Maximum transaction size: 10MB
//   - Maximum transaction duration: 5 seconds
//
// The DeleteFolderChildren operation always uses batched deletion with multiple small transactions
// to safely handle directories of any size. Even if called within an existing transaction context,
// it will create its own batch transactions to avoid exceeding FDB limits.
//
// This means DeleteFolderChildren is NOT atomic with respect to an outer transaction - it manages
// its own transaction boundaries for safety and reliability.

package foundationdb

import (
	"context"
	"fmt"

	"github.com/apple/foundationdb/bindings/go/src/fdb"

	"github.com/seaweedfs/seaweedfs/weed/filer"
)

// KV operations
func (store *FoundationDBStore) KeyPut(ctx context.Context, fdbKey fdb.Key, value []byte) error {

	// Check if there's a transaction in context
	if tx, exists := store.getTransactionFromContext(ctx); exists {
		tx.Set(fdbKey, value)
		return nil
	}

	_, err := store.database.Transact(func(tr fdb.Transaction) (interface{}, error) {
		tr.Set(fdbKey, value)
		return nil, nil
	})

	return err
}

func (store *FoundationDBStore) KeyGet(ctx context.Context, fdbKey fdb.Key) ([]byte, error) {
	var data []byte
	var err error

	// Check if there's a transaction in context
	if tx, exists := store.getTransactionFromContext(ctx); exists {
		data, err = tx.Get(fdbKey).Get()
	} else {
		var result interface{}
		result, err = store.database.ReadTransact(func(rtr fdb.ReadTransaction) (interface{}, error) {
			return rtr.Get(fdbKey).Get()
		})
		if err == nil {
			if resultBytes, ok := result.([]byte); ok {
				data = resultBytes
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("kv get %v: %w", string(fdbKey), err)
	}
	if len(data) == 0 {
		return nil, filer.ErrKvNotFound
	}

	return data, nil
}

func (store *FoundationDBStore) KeyDelete(ctx context.Context, fdbKey fdb.Key) error {
	// Check if there's a transaction in context
	if tx, exists := store.getTransactionFromContext(ctx); exists {
		tx.Clear(fdbKey)
		return nil
	}

	_, err := store.database.Transact(func(tr fdb.Transaction) (interface{}, error) {
		tr.Clear(fdbKey)
		return nil, nil
	})

	return err
}

func (store *FoundationDBStore) KeyList(ctx context.Context, rangeOptions fdb.RangeOptions) ([]fdb.KeyValue, error) {
	result, err := store.database.ReadTransact(func(rtr fdb.ReadTransaction) (interface{}, error) {
		// kvRange, err := f.kvDir.List(fdb.Range{}, []string{})
		kvRange := fdb.SelectorRange{}
		kvSlice, err := rtr.GetRange(kvRange, fdb.RangeOptions{}).GetSliceWithError()
		if err != nil {
			return nil, err
		}
		return kvSlice.([]fdb.KeyValue), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}

	return result, nil
}