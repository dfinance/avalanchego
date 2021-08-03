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
		b.VM.Ctx.Log.Error("Validating block: block validation: %v", err)
		return fmt.Errorf("block validation: %w", err)
	}
	if accepted {
		b.VM.Ctx.Log.Error("Validating block: already accepted")
		return fmt.Errorf("block already accepted")
	}

	if len(b.Txs) == 0 {
		b.VM.Ctx.Log.Error("Validating block: no transactions")
		return fmt.Errorf("block has no transactions")
	}
	for i, tx := range b.Txs {
		if err := tx.ValidateBasic(); err != nil {
			b.VM.Ctx.Log.Error("Validating block: validation: %v", err)
			return fmt.Errorf("tx [%d] (%T): validation: %w", i, tx, err)
		}
	}

	for txIdx, txRaw := range b.Txs {
		switch tx := txRaw.(type) {
		case types.TxDeployModule:
			if _, err := b.state.DeployContract(tx); err != nil {
				b.VM.Ctx.Log.Error("Validating block: DeployContract: %v", err)
				return fmt.Errorf("tx [%d] (%T): deploying contract: %w", txIdx, txRaw, err)
			}
		case types.TxExecuteScript:
			if _, err := b.state.ExecuteContract(tx, b.Height()); err != nil {
				b.VM.Ctx.Log.Error("Validating block: ExecuteContract: %v", err)
				return fmt.Errorf("tx [%d] (%T): executing script: %w", txIdx, txRaw, err)
			}
		default:
			b.VM.Ctx.Log.Error("Validating block: unknown type: %T", txRaw)
			return fmt.Errorf("tx [%d] (%T): unknown type", txIdx, txRaw)
		}
	}

	if err := b.VM.SaveBlock(b.VM.DB, b); err != nil {
		b.VM.Ctx.Log.Error("Validating block: SaveBlock: %v", err)
		return fmt.Errorf("saving block: %w", err)
	}

	if err := b.VM.DB.Commit(); err != nil {
		b.VM.Ctx.Log.Error("Validating block: Commit: %v", err)
		return fmt.Errorf("commiting DB state: %w", err)
	}

	return nil
}
