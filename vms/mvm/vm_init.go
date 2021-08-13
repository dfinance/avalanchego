package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/mvm/state"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
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

	ctx.Log.Info("VM.Initialize: M-chain")
	if len(upgradeData) != 0 {
		return fmt.Errorf("upgradeData: unsupported")
	}

	linerCodec := linearcodec.NewDefault()
	errs := wrappers.Errs{}
	errs.Add(
		linerCodec.RegisterType(&stateTypes.MsgExecuteScript{}),
		linerCodec.RegisterType(&stateTypes.MsgDeployModule{}),
		//
		linerCodec.RegisterType(&types.UnsignedGenesisTx{}),
		linerCodec.RegisterType(&types.UnsignedMoveTx{}),
		linerCodec.RegisterType(&secp256k1fx.Credential{}),
		linerCodec.RegisterType(&types.Tx{}),
		linerCodec.RegisterType(&types.TxState{}),
		//
		linerCodec.RegisterType(&Block{}),
	)
	if errs.Err != nil {
		err := fmt.Errorf("registering codec types: %w", errs.Err)
		ctx.Log.Error("VM.Initialize: %v", err)
		return err
	}

	codecManager := codec.NewDefaultManager()
	if err := codecManager.RegisterCodec(types.CodecVersion, linerCodec); err != nil {
		err = fmt.Errorf("building CodecManager: %w", err)
		ctx.Log.Error("VM.Initialize: %v", err)
		return err
	}
	vm.codec = codecManager

	vm.AddressManager = avax.NewAddressManager(ctx)

	config, err := types.NewConfig(configData)
	if err != nil {
		err = fmt.Errorf("building Config: %w", err)
		ctx.Log.Error("VM.Initialize: %v", err)
		return err
	}
	vm.config = config

	if err := vm.SnowmanVM.Initialize(ctx, dbManager.Current().Database, vm.ParseBlock, toEngine); err != nil {
		err = fmt.Errorf("initializing SnowmanVM: %w")
		ctx.Log.Error("VM.Initialize: %v", err)
		return err
	}

	internalState, genesisBlockInitialized, err := state.NewState(vm.Ctx.Log, config.DVMConnection, genesisData, vm.DB)
	if err != nil {
		err = fmt.Errorf("building internal state: %w", err)
		ctx.Log.Error("VM.Initialize: %v", err)
		return err
	}
	vm.state = internalState

	vm.txStorage = newTXStorage(vm.codec, prefixdb.New(txsDBPrefix, dbManager.Current().Database))

	if genesisBlockInitialized {
		ctx.Log.Info("VM.Initialize: creating genesis block")

		if err := vm.createAndCommitGenesisBlock(); err != nil {
			err = fmt.Errorf("creating and committing genesis block: %w", err)
			ctx.Log.Error("VM.Initialize: %v", err)
			return err
		}
	}

	vm.initialized = true
	ctx.Log.Info("VM.Initialize: M-chain initialized")

	return nil
}

// Shutdown implements common.VM interface.
func (vm *VM) Shutdown() error {
	vm.Ctx.Log.Info("VM.Shutdown: M-chain")
	vm.initialized = false

	if vm.state != nil {
		if err := vm.state.Close(); err != nil {
			return fmt.Errorf("closing internal state: %w", err)
		}
	}
	if vm.txStorage != nil {
		if err := vm.txStorage.Close(); err != nil {
			return fmt.Errorf("closing tx storage: %w", err)
		}
	}

	return nil
}

// createAndCommitGenesisBlock creates and commits genesis block with types.UnsignedGenesisTx Tx.
func (vm *VM) createAndCommitGenesisBlock() error {
	genTx := &types.Tx{UnsignedTx: &types.UnsignedGenesisTx{}}
	if err := genTx.Sign(vm.codec, nil); err != nil {
		return fmt.Errorf("signing Tx: %w", err)
	}

	if err := genTx.Validate(vm.Ctx); err != nil {
		return fmt.Errorf("validating Tx: %w", err)
	}

	block, err := vm.NewBlock(ids.Empty, 0, genTx)
	if err != nil {
		return fmt.Errorf("creating block: %w", err)
	}

	if err := vm.SaveBlock(vm.DB, block); err != nil {
		return fmt.Errorf("saving block: %w", err)
	}

	if err := block.Accept(); err != nil {
		return fmt.Errorf("accepting block: %w", err)
	}

	if err := vm.SetDBInitialized(); err != nil {
		return fmt.Errorf("setting DB initialized flag: %w", err)
	}

	if err := vm.DB.Commit(); err != nil {
		return fmt.Errorf("committing block: %w", err)
	}

	if err := vm.SetPreference(vm.LastAcceptedID); err != nil {
		return fmt.Errorf("setting LastAcceptedID: %w", err)
	}

	//vm.Ctx.Log.Info("Genesis block")
	//vm.Ctx.Log.Info(block.String())

	return nil
}
