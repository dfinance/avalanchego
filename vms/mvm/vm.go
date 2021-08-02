package mvm

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/mvm/state"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

var (
	_ block.ChainVM = &VM{}
)

type VM struct {
	core.SnowmanVM

	codec   codec.Manager
	state   *state.State
	mempool [][]types.Tx

	config types.Config
}

// Version implements common.VM interface.
func (vm *VM) Version() (string, error) {
	return "0.0.1", nil
}
