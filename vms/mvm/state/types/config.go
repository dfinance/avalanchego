package types

import (
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

// DVMConnectionConfig config defines DVM connection params.
type DVMConnectionConfig struct {
	// DVM virtual machine address to connect to.
	DVMAddress string `json:"dvm_address"`
	// Node's data server address to listen for connections from DVM.
	DataServerAddress string `json:"data_server_address"`

	// Retry policy: maximum retry attempts (0 - infinity)
	MaxAttempts uint `json:"max_attempts"`
	// Retry policy: request timeout per attempt [ms] (0 - infinite, no timeout)
	ReqTimeoutInMs uint `json:"req_timeout_in_ms"`
}

// NewDefaultDVMConnectionConfig creates a default DVMConnectionConfig.
func NewDefaultDVMConnectionConfig() DVMConnectionConfig {
	return DVMConnectionConfig{
		DVMAddress:        DefaultDVMAddress,
		DataServerAddress: DefaultDataServerAddress,
		MaxAttempts:       DefaultMaxAttempts,
		ReqTimeoutInMs:    DefaultReqTimeout,
	}
}

// Validate validates Config.
func (c DVMConnectionConfig) Validate() error {
	if _, err := url.Parse(c.DVMAddress); err != nil {
		return fmt.Errorf("dvm_address: invalid URL: %w", err)
	}

	if _, err := url.Parse(c.DataServerAddress); err != nil {
		return fmt.Errorf("data_server_address: invalid URL: %w", err)
	}

	return nil
}
