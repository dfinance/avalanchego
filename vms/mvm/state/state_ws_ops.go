package state

import (
	"fmt"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

func (s *State) PutWriteSet(path *dvm.VMAccessPath, data []byte) error {
	if err := s.writeSetsDB.Put(NewWriteSetStorageKey(path), data); err != nil {
		return fmt.Errorf("saving writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WriteSet (%s): created/updated", types.StringifyVMAccessPath(path))

	return nil
}

func (s *State) DeleteWriteSet(path *dvm.VMAccessPath) error {
	if err := s.writeSetsDB.Delete(NewWriteSetStorageKey(path)); err != nil {
		return fmt.Errorf("removing writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WriteSet (%s): removed", types.StringifyVMAccessPath(path))

	return nil
}

func (s *State) HasWriteSet(path *dvm.VMAccessPath) (bool, error) {
	found, err := s.writeSetsDB.Has(NewWriteSetStorageKey(path))
	if err != nil {
		return false, fmt.Errorf("checking writeSet (%s) exists: %w", types.StringifyVMAccessPath(path), err)
	}

	return found, nil
}

func (s *State) GetWriteSet(path *dvm.VMAccessPath) ([]byte, error) {
	data, err := s.writeSetsDB.Get(NewWriteSetStorageKey(path))
	if err != nil {
		return nil, fmt.Errorf("getting writeSet (%s): %w", types.StringifyVMAccessPath(path), err)
	}
	s.log.Info("WriteSet (%s): read", types.StringifyVMAccessPath(path))

	return data, nil
}
