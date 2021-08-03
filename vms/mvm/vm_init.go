package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/vms/mvm/state"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// Initialize implements common.VM interface.
func (vm *VM) Initialize(
	ctx *snow.Context,
	dbManager manager.Manager,
	genesisData []byte,
	upgradeData []byte,
	configData []byte,
	toEngine chan<- common.Message,
	_ []*common.Fx,
) error {

	ctx.Log.Info("M-chain initialization")
	if len(upgradeData) != 0 {
		return fmt.Errorf("upgradeData: unsupported")
	}

	linerCodec := linearcodec.NewDefault()
	if err := linerCodec.RegisterType(types.TxDeployModule{}); err != nil {
		ctx.Log.Error("Registering %T codec type: %v", types.TxDeployModule{}, err)
		return err
	}
	if err := linerCodec.RegisterType(types.TxExecuteScript{}); err != nil {
		ctx.Log.Error("Registering %T codec type: %v", types.TxExecuteScript{}, err)
		return err
	}
	codecManager := codec.NewDefaultManager()
	if err := codecManager.RegisterCodec(types.CodecVersion, linerCodec); err != nil {
		ctx.Log.Error("Building CodecManager: %v", err)
		return err
	}
	vm.codec = codecManager

	config, err := types.NewConfig(configData)
	if err != nil {
		ctx.Log.Error("Building Config: %v", err)
		return err
	}
	vm.config = config

	if err := vm.SnowmanVM.Initialize(ctx, dbManager.Current().Database, vm.ParseBlock, toEngine); err != nil {
		ctx.Log.Error("Initializing SnowmanVM: %v", err)
		return err
	}

	internalState, genesisBlockInitialized, err := state.NewState(vm.Ctx.Log, config.DVMConnection, genesisData, vm.DB)
	if err != nil {
		ctx.Log.Error("Building internal state: %v", err)
		return err
	}
	vm.state = internalState

	if genesisBlockInitialized {
		ctx.Log.Info("Creating genesis block")

		block, err := vm.NewBlock(ids.Empty, 0, nil)
		if err != nil {
			ctx.Log.Error("creating genesis block: %v", err)
			return err
		}

		if err := vm.SaveBlock(vm.DB, block); err != nil {
			ctx.Log.Error("saving genesis block: %v", err)
			return err
		}

		if err := block.Accept(); err != nil {
			ctx.Log.Error("accepting genesis block: %v", err)
			return err
		}

		if err := vm.SetDBInitialized(); err != nil {
			ctx.Log.Error("setting DB initialized: %v", err)
			return err
		}

		if err := vm.DB.Commit(); err != nil {
			ctx.Log.Error("committing genesis block: %v", err)
			return err
		}

		if err := vm.SetPreference(vm.LastAcceptedID); err != nil {
			ctx.Log.Error("setting LastAcceptedID: %v", err)
			return err
		}
	}

	return nil
}

// Shutdown implements common.VM interface.
func (vm *VM) Shutdown() error {
	if vm.state != nil {
		if err := vm.state.Close(); err != nil {
			return fmt.Errorf("closing internal state: %w", err)
		}
	}

	return nil
}
