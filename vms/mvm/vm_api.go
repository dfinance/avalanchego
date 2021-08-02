package mvm

import "github.com/ava-labs/avalanchego/snow/engine/common"

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
