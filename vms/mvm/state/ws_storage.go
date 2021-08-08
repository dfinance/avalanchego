package state

import (
	"bytes"
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"github.com/ava-labs/avalanchego/vms/mvm/state/types"
)

// wsStorage encapsulates writeSets storage operations.
type wsStorage struct {
	db  database.Database
	log logging.Logger
}

// newWSStorage creates a new wsStorage instance.
func newWSStorage(logger logging.Logger, db database.Database) *wsStorage {
	return &wsStorage{
		db:  db,
		log: logger,
	}
}

// Close closes storage DB.
func (s *wsStorage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("closing writeSets storage: %w", err)
	}

	return nil
}

// PutWriteSet sets writeSet data.
func (s *wsStorage) PutWriteSet(path *dvm.VMAccessPath, data []byte) error {
	if err := s.db.Put(s.newWriteSetStorageKey(path), data); err != nil {
		return fmt.Errorf("saving writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WS storage: WriteSet (%s): created/updated", types.StringifyVMAccessPath(path))

	return nil
}

// DeleteWriteSet removes writeSet data.
func (s *wsStorage) DeleteWriteSet(path *dvm.VMAccessPath) error {
	if err := s.db.Delete(s.newWriteSetStorageKey(path)); err != nil {
		return fmt.Errorf("removing writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WS storage: WriteSet (%s): removed", types.StringifyVMAccessPath(path))

	return nil
}

// HasWriteSet checks if writeSet exists.
func (s *wsStorage) HasWriteSet(path *dvm.VMAccessPath) (bool, error) {
	found, err := s.db.Has(s.newWriteSetStorageKey(path))
	if err != nil {
		return false, fmt.Errorf("checking writeSet (%s) exists: %w", types.StringifyVMAccessPath(path), err)
	}

	return found, nil
}

// GetWriteSet gets writeSet data.
func (s *wsStorage) GetWriteSet(path *dvm.VMAccessPath) ([]byte, error) {
	data, err := s.db.Get(s.newWriteSetStorageKey(path))
	if err != nil {
		return nil, fmt.Errorf("getting writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WSStorage: WriteSet (%s): read", types.StringifyVMAccessPath(path))

	return data, nil
}

// newWriteSetStorageKey builds storage key for writeSet.
func (s *wsStorage) newWriteSetStorageKey(path *dvm.VMAccessPath) []byte {
	if path == nil {
		return nil
	}

	return bytes.Join(
		[][]byte{
			path.Address,
			path.Path,
		},
		keyDelimiter,
	)
}
