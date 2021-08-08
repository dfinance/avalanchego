package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/mvm/state"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

var (
	_ block.ChainVM = &VM{}
)

type VM struct {
	core.SnowmanVM
	avax.AddressManager

	codec   codec.Manager
	factory crypto.FactorySECP256K1R
	state   *state.State
	mempool []*types.Tx

	config      types.Config
	initialized bool
}

// Version implements common.VM interface.
func (vm *VM) Version() (string, error) {
	return "0.0.1", nil
}

// HealthCheck implements the health.Checkable interface.
func (vm *VM) HealthCheck() (interface{}, error) {
	return nil, nil
}

// Connected implements validators.Connector interface.
func (vm *VM) Connected(id ids.ShortID) error {
	return nil
}

// Disconnected implements validators.Connector interface.
func (vm *VM) Disconnected(id ids.ShortID) error {
	return nil
}

// CreateHandlers implements common.VM interface.
func (vm *VM) CreateHandlers() (map[string]*common.HTTPHandler, error) {
	handler, err := vm.NewHandler("mvm", &Service{vm})
	return map[string]*common.HTTPHandler{
		"": handler,
	}, err
}

// CreateStaticHandlers implements common.StaticVM interface.
func (vm *VM) CreateStaticHandlers() (map[string]*common.HTTPHandler, error) {
	return nil, nil
}

// CheckInitialized checks if chain initialized successfully (used by the API to prevent panics).
func (vm *VM) CheckInitialized() error {
	if vm.initialized {
		return nil
	}

	return fmt.Errorf("chain is not initialized")
}

// getUserSvc returns configured userSvc for user (use defer to close the service).
func (vm *VM) getUserSvc(userName, password string) (*userSvc, error) {
	db, err := vm.Ctx.Keystore.GetDatabase(userName, password)
	if err != nil {
		return nil, fmt.Errorf("retrieving user (%s): %w", userName, err)
	}

	return &userSvc{
		codec: vm.codec,
		db:    db,
	}, nil
}
