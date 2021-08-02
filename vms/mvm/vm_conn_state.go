package mvm

import "github.com/ava-labs/avalanchego/ids"

// Connected implements validators.Connector interface.
func (vm *VM) Connected(id ids.ShortID) error {
	return nil
}

// Disconnected implements validators.Connector interface.
func (vm *VM) Disconnected(id ids.ShortID) error {
	return nil
}
