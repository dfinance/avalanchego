package types

import (
	"fmt"

	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
)

type (
	TxState struct {
		Tx         Tx                `serialize:"true" json:"tx"`
		TxStatus   TxStatus          `serialize:"true" json:"status"`
		ErrMessage string            `serialize:"true" json:"errorMessage,omitempty"`
		Events     stateTypes.Events `serialize:"true" json:"events,omitempty"`
	}

	TxStatus string
)

const (
	TxStatusProcessing TxStatus = "Processing"
	TxStatusDropped    TxStatus = "Dropped"
	TxStatusCommitted  TxStatus = "Committed"
)

func (s TxStatus) Validate() error {
	switch s {
	case TxStatusProcessing, TxStatusDropped, TxStatusCommitted:
		return nil
	default:
		return fmt.Errorf("unknown TxStatus: %s", string(s))
	}
}
