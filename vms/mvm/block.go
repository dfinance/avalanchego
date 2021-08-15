package mvm

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ava-labs/avalanchego/vms/components/core"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// Block defines the VM's block.
type Block struct {
	*core.Block `serialize:"true"`

	Txs []*types.Tx `serialize:"true"`
	vm  *VM
}

// String implements fmt.Stringer interface.
func (b *Block) String() string {
	str := strings.Builder{}
	str.WriteString("\n")
	str.WriteString(fmt.Sprintf("ID:        %s\n", b.ID().String()))
	str.WriteString(fmt.Sprintf("Parent ID: %s\n", b.PrntID.String()))
	str.WriteString(fmt.Sprintf("Height:    %d\n", b.Height()))
	str.WriteString(fmt.Sprintf("Status:    %s\n", b.Status().String()))
	str.WriteString("TXs:\n")
	for txIdx, tx := range b.Txs {
		switch utx := tx.UnsignedTx.(type) {
		case *types.UnsignedMoveTx:
			str.WriteString(fmt.Sprintf("- [%d] (%T): %+v\n", txIdx, utx, utx))
		case *types.UnsignedGenesisTx:
			str.WriteString(fmt.Sprintf("- [%d] (%T): %+v\n", txIdx, utx, utx))
		default:
			str.WriteString(fmt.Sprintf("- [%d] (%T): unsupported UnsignedTx type\n", txIdx, tx.UnsignedTx))
		}

		str.WriteString("  Creds:\n")
		for credIdx, cred := range tx.Creds {
			str.WriteString(fmt.Sprintf("  - [%d] (%T): %+v\n", credIdx, cred, cred))
		}
		str.WriteString(fmt.Sprintf("  Unsigned bytes: 0x%s\n", hex.EncodeToString(tx.UnsignedBytes())))
		str.WriteString(fmt.Sprintf("  Signed bytes:   0x%s\n", hex.EncodeToString(tx.Bytes())))
	}

	return str.String()
}

// Initialize wraps core.Block Initialize setting VM pointer.
func (b *Block) Initialize(blockBz []byte, vm *VM) error {
	b.Block.Initialize(blockBz, &vm.SnowmanVM)
	b.vm = vm

	for txIdx, tx := range b.Txs {
		if err := tx.Sign(vm.codec, nil); err != nil {
			return fmt.Errorf("tx [%d]: sign failed: %w", txIdx, err)
		}
	}

	return nil
}

// Verify implement snowman.Block interface.
func (b *Block) Verify() error {
	if err := b.validate(); err != nil {
		err = fmt.Errorf("block validation: %w", err)
		b.vm.Ctx.Log.Error("Block.Verify: %v", err)
		return err
	}

	if err := b.commitTxs(); err != nil {
		if err := b.Reject(); err != nil {
			err = fmt.Errorf("rejecting block: %w", err)
			b.vm.Ctx.Log.Error("Block.Verify: %v", err)
			return err
		}

		err = fmt.Errorf("commiting Txs: %w", err)
		b.vm.Ctx.Log.Error("Block.Verify: %v", err)
		return err
	}

	if err := b.VM.SaveBlock(b.VM.DB, b); err != nil {
		err = fmt.Errorf("saving block: %w", err)
		b.vm.Ctx.Log.Error("Block.Verify: %v", err)
		return err
	}

	if err := b.VM.DB.Commit(); err != nil {
		err = fmt.Errorf("commiting DB state: %w", err)
		b.vm.Ctx.Log.Error("Block.Verify: %v", err)
		return err
	}

	return nil
}

// validate performs basic block validation.
func (b *Block) validate() error {
	accepted, err := b.Block.Verify()
	if err != nil {
		return fmt.Errorf("block validation: %w", err)
	}
	if accepted {
		return fmt.Errorf("block already accepted")
	}

	if len(b.Txs) == 0 {
		return fmt.Errorf("block has no transactions")
	}
	for txIdx, tx := range b.Txs {
		if tx == nil {
			return fmt.Errorf("tx [%d]: nil", txIdx)
		}

		if err := tx.Validate(b.vm.Ctx); err != nil {
			return fmt.Errorf("tx [%d]: validation: %w", txIdx, err)
		}

		for credIdx, cred := range tx.Creds {
			if cred == nil {
				return fmt.Errorf("tx [%d]: checking signature [%d]: nil", txIdx, credIdx)
			}

			if err := cred.Verify(); err != nil {
				return fmt.Errorf("tx [%d]: checking signature [%d]: %w", txIdx, credIdx, err)
			}
		}
	}

	return nil
}

// commitTxs iterates over block Txs and commits them.
func (b *Block) commitTxs() error {
	for txIdx, txRaw := range b.Txs {
		switch utx := txRaw.UnsignedTx.(type) {
		case *types.UnsignedMoveTx:
			events, err := b.commitUnsignedMoveTx(utx)
			b.vm.Ctx.Log.Info("Executing MoveTx (%s): error: %v", txRaw.ID().String(), err)
			if err != nil {
				if err := b.vm.txStorage.PutDroppedTx(txRaw, events, err); err != nil {
					return fmt.Errorf("tx [%d] (%T): saving dropped Tx: %w", txIdx, utx, err)
				}

				return fmt.Errorf("tx [%d] (%T): %w", txIdx, utx, err)
			}

			if err := b.vm.txStorage.PutCommittedTx(txRaw, events); err != nil {
				return fmt.Errorf("tx [%d] (%T): saving committed Tx: %w", txIdx, utx, err)
			}
		default:
			return fmt.Errorf("tx [%d] (%T): unknown UnsignedTx type", txIdx, txRaw)
		}
	}

	return nil
}

// commitUnsignedMoveTx commits types.UnsignedMoveTx Tx.
func (b *Block) commitUnsignedMoveTx(utx *types.UnsignedMoveTx) (stateTypes.Events, error) {
	var events stateTypes.Events
	var err error

	switch msg := utx.Msg.(type) {
	case *stateTypes.MsgDeployModule:
		events, err = b.vm.state.DeployContract(msg)
		if err != nil {
			err = fmt.Errorf("deploying contract: %w", err)
		}
	case *stateTypes.MsgExecuteScript:
		events, err = b.vm.state.ExecuteContract(msg, b.Height())
		if err != nil {
			err = fmt.Errorf("executing script: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown Msg (%T) type", utx.Msg)
	}

	return events, err
}
