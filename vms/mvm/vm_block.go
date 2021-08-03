package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// NewBlock returns a new Block.
func (vm *VM) NewBlock(parentID ids.ID, height uint64, txs []types.Tx) (*Block, error) {
	block := &Block{
		Block: core.NewBlock(parentID, height),
		Txs:   txs,
	}

	blockBz, err := vm.codec.Marshal(types.CodecVersion, block)
	if err != nil {
		return nil, fmt.Errorf("block marshal: %w", err)
	}
	block.Initialize(blockBz, &vm.SnowmanVM, vm.state)

	return block, nil
}

// ParseBlock parses [bytes] to a snowman.Block.
// This function is used by the VM's state to unmarshal blocks saved in state.
func (vm *VM) ParseBlock(blockBz []byte) (snowman.Block, error) {
	block := &Block{}
	if _, err := vm.codec.Unmarshal(blockBz, block); err != nil {
		vm.Ctx.Log.Error("Parsing block: %v", err)
		return nil, fmt.Errorf("block unmarshal: %w", err)
	}
	block.Initialize(blockBz, &vm.SnowmanVM, vm.state)

	return block, nil
}

// BuildBlock implements block.ChainVM interface.
func (vm *VM) BuildBlock() (snowman.Block, error) {
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
		vm.Ctx.Log.Error("Building block: getting preferred block: %v", err)
		return nil, fmt.Errorf("getting preferred block: %w", err)
	}
	preferredHeight := preferredBlock.(*Block).Height()

	// Build the block
	block, err := vm.NewBlock(vm.Preferred(), preferredHeight+1, value)
	if err != nil {
		vm.Ctx.Log.Error("Building block: creating a new Block: %v", err)
		return nil, fmt.Errorf("building new block: %w", err)
	}

	return block, nil
}

// proposeBlock appends data to mempool and notifies the consensus engine.
func (vm *VM) proposeBlock(data []types.Tx) {
	txs := make([]types.Tx, len(data))
	copy(txs, data)

	vm.mempool = append(vm.mempool, data)
	vm.NotifyBlockReady()
}
