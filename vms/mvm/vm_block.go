package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// NewBlock returns a new Block.
func (vm *VM) NewBlock(parentID ids.ID, height uint64, tx *types.Tx) (*Block, error) {
	if tx == nil {
		return nil, fmt.Errorf("tx: nil")
	}

	block := &Block{
		Block: core.NewBlock(parentID, height),
		Txs:   []*types.Tx{tx},
	}

	blockBz, err := vm.codec.Marshal(types.CodecVersion, block)
	if err != nil {
		return nil, fmt.Errorf("block marshal: %w", err)
	}

	if err := block.Initialize(blockBz, vm); err != nil {
		return nil, fmt.Errorf("block initialization: %w", err)
	}

	return block, nil
}

// ParseBlock parses bytes to a snowman.Block.
func (vm *VM) ParseBlock(blockBz []byte) (snowman.Block, error) {
	//vm.Ctx.Log.Info("VM.ParseBlock")

	block := &Block{}
	if _, err := vm.codec.Unmarshal(blockBz, block); err != nil {
		err = fmt.Errorf("block unmarshal: %w", err)
		vm.Ctx.Log.Error("VM.ParseBlock: %v", err)
		return nil, err
	}

	if err := block.Initialize(blockBz, vm); err != nil {
		err = fmt.Errorf("block initialization: %w", err)
		vm.Ctx.Log.Error("VM.ParseBlock: %v", err)
		return nil, err
	}

	//vm.Ctx.Log.Info(block.String())

	return block, nil
}

// BuildBlock implements block.ChainVM interface.
func (vm *VM) BuildBlock() (snowman.Block, error) {
	vm.Ctx.Log.Info("VM.BuildBlock")

	if len(vm.mempool) == 0 {
		return nil, fmt.Errorf("no block to propose")
	}

	// Get the value to put in the new Block
	value := vm.mempool[0]
	vm.mempool = vm.mempool[1:]

	// Notify consensus engine that there are more pending data for Blocks
	if len(vm.mempool) > 0 {
		defer vm.NotifyBlockReady()
	}

	preferredBlock, err := vm.GetBlock(vm.Preferred())
	if err != nil {
		err = fmt.Errorf("getting preferred block: %w", err)
		vm.Ctx.Log.Error("VM.BuildBlock: %v", err)
		return nil, err
	}
	preferredHeight := preferredBlock.(*Block).Height()

	// Build the block
	block, err := vm.NewBlock(vm.Preferred(), preferredHeight+1, value)
	if err != nil {
		err = fmt.Errorf("building new block: %w", err)
		vm.Ctx.Log.Error("VM.BuildBlock: %v", err)
		return nil, err
	}

	return block, nil
}

// issueTx appends Tx to the mempool and notifies the consensus engine if Tx not exists.
func (vm *VM) issueTx(tx *types.Tx) error {
	existingTx, err := vm.txStorage.GetTxState(tx.ID())
	if err != nil {
		return fmt.Errorf("checking if Tx exists in state: %w", err)
	}
	if existingTx != nil {
		return fmt.Errorf("tx (%s) exists in state with status (%s)", tx.ID(), existingTx.TxStatus)
	}

	vm.mempool = append(vm.mempool, tx)
	vm.NotifyBlockReady()

	return nil
}
