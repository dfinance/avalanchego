package types

import (
	"encoding/json"
	"fmt"

	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
)

// Config defines VM configuration.
type Config struct {
	DVMConnection stateTypes.DVMConnectionConfig `json:"dvm_connection"`
}

// NewConfig returns a valid Config from the serialized data or the default one.
func NewConfig(configBz []byte) (Config, error) {
	if len(configBz) == 0 {
		return Config{}, fmt.Errorf("empty config")
	}

	var config Config
	if err := json.Unmarshal(configBz, &config); err != nil {
		return Config{}, fmt.Errorf("unmarshal JSON: %w", err)
	}

	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("validation: %w", err)
	}

	return config, nil
}

// Validate validates Config.
func (c Config) Validate() error {
	if err := c.DVMConnection.Validate(); err != nil {
		return fmt.Errorf("dvm_connection: %w", err)
	}

	return nil
}
