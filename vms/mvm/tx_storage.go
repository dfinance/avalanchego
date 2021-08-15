package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

const (
	txCacheSize = 2048
)

// txStorage encapsulates transactions storage operation.
type txStorage struct {
	log            logging.Logger
	codec          codec.Manager
	txsDB          database.Database
	droppedTxCache cache.Cacher
}

// newTXStorage creates a new txStorage instance.
func newTXStorage(logger logging.Logger, codec codec.Manager, db database.Database) *txStorage {
	return &txStorage{
		log:            logger,
		codec:          codec,
		txsDB:          db,
		droppedTxCache: &cache.LRU{Size: txCacheSize},
	}
}

// Close closes Tx DB.
func (s *txStorage) Close() error {
	if err := s.txsDB.Close(); err != nil {
		return fmt.Errorf("closing txs storage: %w", err)
	}

	return nil
}

// PutCommittedTx puts a new committed transaction to DB.
func (s *txStorage) PutCommittedTx(tx *types.Tx, events stateTypes.Events) error {
	txStateBz, err := s.buildTxStateBz(tx, types.TxStatusCommitted, events, nil)
	if err != nil {
		return fmt.Errorf("building Tx state: %w", err)
	}
	txID := tx.ID()

	s.log.Info("Tx storage: saving committed Tx: %s", tx.ID())
	if err := s.txsDB.Put(txID[:], txStateBz); err != nil {
		return fmt.Errorf("storing committed Tx state: %w", err)
	}

	return nil
}

// PutDroppedTx puts a new dropped transaction to LRU cache.
func (s *txStorage) PutDroppedTx(tx *types.Tx, events stateTypes.Events, err error) error {
	txStateBz, err := s.buildTxStateBz(tx, types.TxStatusDropped, events, err)
	if err != nil {
		return fmt.Errorf("building Tx state: %w", err)
	}

	s.log.Info("Tx storage: caching dropped Tx: %s", tx.ID())
	s.droppedTxCache.Put(tx.ID(), txStateBz)

	return nil
}

// GetTxState gets an existing transaction from LRU cache or DB.
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

// buildTxStateBz builds and serialized types.TxState object.
func (s *txStorage) buildTxStateBz(tx *types.Tx, status types.TxStatus, events stateTypes.Events, err error) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("tx: nil")
	}
	if err := status.Validate(); err != nil {
		return nil, fmt.Errorf("status: invalid: %w", err)
	}

	txState := types.TxState{
		Tx:         *tx,
		TxStatus:   status,
		Events:     events,
		ErrMessage: "",
	}
	if err != nil {
		txState.ErrMessage = err.Error()
	}

	txStateBz, err := s.codec.Marshal(types.CodecVersion, &txState)
	if err != nil {
		return nil, fmt.Errorf("serializing Tx state: %w", err)
	}

	return txStateBz, nil
}

// unmarshalTxStateBz deserializes types.TxState object.
func (s *txStorage) unmarshalTxStateBz(txStateBz []byte) (*types.TxState, error) {
	var txState types.TxState
	if _, err := s.codec.Unmarshal(txStateBz, &txState); err != nil {
		return nil, fmt.Errorf("deserializing Tx state: %w", err)
	}

	return &txState, nil
}
