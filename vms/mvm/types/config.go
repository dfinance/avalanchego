package types

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// DVM connection and retry policy defaults.
const (
	DefaultDVMAddress        = "tcp://127.0.0.1:50051"
	DefaultDataServerAddress = "tcp://127.0.0.1:50061"
	DefaultMaxAttempts       = 0
	DefaultReqTimeout        = 0
)

// Config defines VM configuration.
type (
	Config struct {
		DVMConnection DVMConnection `json:"dvm_connection"`
	}

	DVMConnection struct {
		// DVM virtual machine address to connect to.
		DVMAddress string `json:"dvm_address"`
		// Node's data server address to listen for connections from DVM.
		DataServerAddress string `json:"data_server_address"`

		// Retry policy: maximum retry attempts (0 - infinity)
		MaxAttempts uint `json:"max_attempts"`
		// Retry policy: request timeout per attempt [ms] (0 - infinite, no timeout)
		ReqTimeoutInMs uint `json:"req_timeout_in_ms"`
	}
)

// NewDefaultConfig creates a default Config.
func NewDefaultConfig() Config {
	return Config{
		DVMConnection: DVMConnection{
			DVMAddress:        DefaultDVMAddress,
			DataServerAddress: DefaultDataServerAddress,
			MaxAttempts:       DefaultMaxAttempts,
			ReqTimeoutInMs:    DefaultReqTimeout,
		},
	}
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
	if _, err := url.Parse(c.DVMConnection.DVMAddress); err != nil {
		return fmt.Errorf("dvm_connection: dvm_address: invalid URL: %w", err)
	}

	if _, err := url.Parse(c.DVMConnection.DataServerAddress); err != nil {
		return fmt.Errorf("dvm_connection: data_server_address: invalid URL: %w", err)
	}

	return nil
}
