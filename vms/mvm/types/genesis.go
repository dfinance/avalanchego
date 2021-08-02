package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

// GenesisState defines VM genesis state.
type (
	GenesisState struct {
		WriteSets []WriteSet `json:"write_set"`
	}

	// WriteSet defines a single dvm.VMValue writeSet Operation.
	WriteSet struct {
		Address string `json:"address"`
		Path    string `json:"path"`
		Value   string `json:"value"`
	}
)

// NewGenesisState builds a valid GenesisState from serialized data.
func NewGenesisState(stateBz []byte) (GenesisState, error) {
	state := GenesisState{}
	if len(stateBz) > 0 {
		if err := json.Unmarshal(stateBz, &state); err != nil {
			return GenesisState{}, fmt.Errorf("unmarshal JSON for provided state: %w", err)
		}
	} else {
		if err := json.Unmarshal([]byte(defaultGenesisState), &state); err != nil {
			return GenesisState{}, fmt.Errorf("unmarshal JSON for default state: %w", err)
		}
	}

	if err := state.Validate(); err != nil {
		return GenesisState{}, fmt.Errorf("validation: %w", err)
	}

	return state, nil
}

// String implements fmt.Stringer interface.
func (m WriteSet) String() string {
	return fmt.Sprintf("%s:%s", m.Address, m.Path)
}

// Validate performs a basic validation of WriteSet object.
func (m WriteSet) Validate() error {
	if m.Address == "" {
		return fmt.Errorf("address: empty")
	}

	if m.Path == "" {
		return fmt.Errorf("path: empty")
	}

	if m.Value == "" {
		return fmt.Errorf("value: empty")
	}

	return nil
}

// ToBytes converts WriteSet to dvmTypes.VMAccessPath and []byte representation for value.
func (m WriteSet) ToBytes() (*dvm.VMAccessPath, []byte, error) {
	bzAddr, err := hex.DecodeString(m.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("address: hex decode: %w", err)
	}
	if len(bzAddr) != DVMAddressLength {
		return nil, nil, fmt.Errorf("address: incorrect length (should be %d bytes)", DVMAddressLength)
	}

	bzPath, err := hex.DecodeString(m.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("path: hex decode: %w", err)
	}

	bzValue, err := hex.DecodeString(m.Value)
	if err != nil {
		return nil, nil, fmt.Errorf("value: hex decode: %w", err)
	}

	return &dvm.VMAccessPath{
		Address: bzAddr,
		Path:    bzPath,
	}, bzValue, nil
}

// Validate performs GenesisState validation.
func (s GenesisState) Validate() error {
	// VM WriteSets
	writeOpsSet := make(map[string]struct{}, len(s.WriteSets))
	for _, ws := range s.WriteSets {
		if err := ws.Validate(); err != nil {
			return fmt.Errorf("writeSet (%s): %w", ws, err)
		}

		if _, _, err := ws.ToBytes(); err != nil {
			return fmt.Errorf("writeSet (%s): %w", ws, err)
		}

		writeOpId := ws.String()
		if _, ok := writeOpsSet[writeOpId]; ok {
			return fmt.Errorf("writeSet (%s): duplicated (%s)", ws, writeOpId)
		}
		writeOpsSet[writeOpId] = struct{}{}
	}

	return nil
}

func (s GenesisState) ToVMExecuteResponses() []*dvm.VMExecuteResponse {
	exec := &dvm.VMExecuteResponse{
		WriteSet: make([]*dvm.VMValue, 0, len(s.WriteSets)),
		Status:   &dvm.VMStatus{},
	}

	for _, ws := range s.WriteSets {
		vmAccessPath, vmValue, _ := ws.ToBytes()

		exec.WriteSet = append(exec.WriteSet, &dvm.VMValue{
			Type:  dvm.VmWriteOp_Value,
			Value: vmValue,
			Path:  vmAccessPath,
		})
	}

	return []*dvm.VMExecuteResponse{exec}
}
