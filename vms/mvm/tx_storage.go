package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

const (
	txCacheSize = 2048
)

type txStorage struct {
	codec          codec.Manager
	txsDB          database.Database
	droppedTxCache cache.Cacher
}

func newTXStorage(codec codec.Manager, db database.Database) *txStorage {
	return &txStorage{
		codec:          codec,
		txsDB:          db,
		droppedTxCache: &cache.LRU{Size: txCacheSize},
	}
}

func (s *txStorage) Close() error {
	if err := s.txsDB.Close(); err != nil {
		return fmt.Errorf("closing txs storage: %w", err)
	}

	return nil
}

func (s *txStorage) PutCommittedTx(tx *types.Tx, events stateTypes.Events) error {
	txStateBz, err := s.buildTxStateBz(tx, types.TxStatusCommitted, events)
	if err != nil {
		return fmt.Errorf("building Tx state: %w", err)
	}
	txID := tx.ID()

	if err := s.txsDB.Put(txID[:], txStateBz); err != nil {
		return fmt.Errorf("storing committed Tx state: %w", err)
	}

	return nil
}

func (s *txStorage) PutDroppedTx(tx *types.Tx, events stateTypes.Events) error {
	txStateBz, err := s.buildTxStateBz(tx, types.TxStatusCommitted, events)
	if err != nil {
		return fmt.Errorf("building Tx state: %w", err)
	}

	s.droppedTxCache.Put(tx.ID(), txStateBz)

	return nil
}

func (s *txStorage) GetTxState(txID ids.ID) (*types.TxState, error) {
	if txStateBz, found := s.droppedTxCache.Get(txID); found {
		return s.unmarshalTxStateBz(txStateBz.([]byte))
	}

	txIDBz := txID[:]
	found, err := s.txsDB.Has(txIDBz)
	if err != nil {
		return nil, fmt.Errorf("checking if Tx state exists in DB: %w", err)
	}
	if !found {
		return nil, nil
	}

	txStateBz, err := s.txsDB.Get(txIDBz)
	if err != nil {
		return nil, fmt.Errorf("reading Tx state from DB: %w", err)
	}

	return s.unmarshalTxStateBz(txStateBz)
}

func (s *txStorage) buildTxStateBz(tx *types.Tx, status types.TxStatus, events stateTypes.Events) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("tx: nil")
	}
	if err := status.Validate(); err != nil {
		return nil, fmt.Errorf("status: invalid: %w", err)
	}

	txState := types.TxState{
		Tx:       *tx,
		TxStatus: types.TxStatusCommitted,
		Events:   events,
	}

	txStateBz, err := s.codec.Marshal(types.CodecVersion, &txState)
	if err != nil {
		return nil, fmt.Errorf("serializing Tx state: %w", err)
	}

	return txStateBz, nil
}

func (s *txStorage) unmarshalTxStateBz(txStateBz []byte) (*types.TxState, error) {
	var txState types.TxState
	if _, err := s.codec.Unmarshal(txStateBz, &txState); err != nil {
		return nil, fmt.Errorf("deserializing Tx state: %w", err)
	}

	return &txState, nil
}
