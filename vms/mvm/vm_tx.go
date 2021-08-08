package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// newMoveTx builds and signs types.Tx containing types.UnsignedMoveTx UTx.
func (vm *VM) newMoveTx(msg stateTypes.Msg, keys []*crypto.PrivateKeySECP256K1R) (*types.Tx, error) {
	tx := &types.Tx{
		UnsignedTx: &types.UnsignedMoveTx{
			BaseTx: avax.BaseTx{
				NetworkID:    vm.Ctx.NetworkID,
				BlockchainID: vm.Ctx.ChainID,
			},
			Msg: msg,
		},
	}
	if err := tx.Sign(vm.codec, [][]*crypto.PrivateKeySECP256K1R{keys}); err != nil {
		return nil, fmt.Errorf("signing Tx: %w", err)
	}

	if err := tx.Validate(vm.Ctx); err != nil {
		return nil, fmt.Errorf("validating Tx: %w", err)
	}

	return tx, nil
}
