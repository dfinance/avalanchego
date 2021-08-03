package types

import (
	"fmt"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

type Tx interface {
	GetSigner() string
	ValidateBasic() error
}

type (
	// TxExecuteScript defines a Tx message to execute a script with args using DVM.
	TxExecuteScript struct {
		Signer string      `serialize:"true"` // Tx sender address
		Script []byte      `serialize:"true"` // Script source code
		Args   []ScriptArg `serialize:"true"` // Script arguments
	}

	ScriptArg struct {
		Type  dvm.VMTypeTag `serialize:"true"`
		Value []byte        `serialize:"true"`
	}
)

type (
	// TxDeployModule defines a Tx message to deploy a module (contract) using DVM.
	TxDeployModule struct {
		Signer  string   `serialize:"true"` // Tx sender address
		Modules [][]byte `serialize:"true"` // Modules source code
	}
)

// GetSigner implements Tx interface.
func (m TxExecuteScript) GetSigner() string {
	return m.Signer
}

// ValidateBasic implements Tx interface.
func (m TxExecuteScript) ValidateBasic() error {
	if m.Signer == "" {
		return fmt.Errorf("signer: empty")
	}

	if len(m.Script) == 0 {
		return fmt.Errorf("script: empty")
	}

	for i, arg := range m.Args {
		if _, err := StringifyDVMTypeTag(arg.Type); err != nil {
			return fmt.Errorf("args [%d]: type: %w", i, err)
		}
		if len(arg.Value) == 0 {
			return fmt.Errorf("args [%d]: value: empty", i)
		}
	}

	return nil
}

// GetSigner implements Tx interface.
func (m TxDeployModule) GetSigner() string {
	return m.Signer
}

// ValidateBasic implements Tx interface.
func (m TxDeployModule) ValidateBasic() error {
	if m.Signer == "" {
		return fmt.Errorf("signer: empty")
	}

	if len(m.Modules) == 0 {
		return fmt.Errorf("modules: empty")
	}
	for i, module := range m.Modules {
		if len(module) == 0 {
			return fmt.Errorf("modules [%d]: empty", i)
		}
	}

	return nil
}
