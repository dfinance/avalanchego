package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/mvm/state"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// Block defines the VM's block.
type Block struct {
	*core.Block `serialize:"true"`

	Txs   []types.Tx `serialize:"true"`
	state *state.State
}

func (b *Block) Initialize(blockBz []byte, vm *core.SnowmanVM, state *state.State) {
	b.Block.Initialize(blockBz, vm)
	b.state = state
}

// Verify implement snowman.Block interface.
func (b *Block) Verify() error {
	accepted, err := b.Block.Verify()
	if err != nil {
		return fmt.Errorf("block validation: %w", err)
	}
	if accepted {
		return fmt.Errorf("block already accepted")
	}

	if len(b.Txs) == 0 {
		return fmt.Errorf("block has not transactions")
	}
	for i, tx := range b.Txs {
		if err := tx.ValidateBasic(); err != nil {
			return fmt.Errorf("tx [%d] (%T): validation: %w", i, tx, err)
		}
	}

	for txIdx, txRaw := range b.Txs {
		switch tx := txRaw.(type) {
		case types.TxDeployModule:
			if err := b.state.DeployContract(tx); err != nil {
				return fmt.Errorf("tx [%d] (%T): deploying contract: %w", txIdx, txRaw, err)
			}
		case types.TxExecuteScript:
			if err := b.state.ExecuteContract(tx, b.Height()); err != nil {
				return fmt.Errorf("tx [%d] (%T): executing script: %w", txIdx, txRaw, err)
			}

		default:
			return fmt.Errorf("tx [%d] (%T): unknown type", txIdx, txRaw)
		}
	}

	if err := b.VM.SaveBlock(b.VM.DB, b); err != nil {
		return fmt.Errorf("saving block: %w", err)
	}

	if err := b.VM.DB.Commit(); err != nil {
		return fmt.Errorf("commiting DB state: %w", err)
	}

	return nil
}
