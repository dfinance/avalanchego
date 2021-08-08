package types

const (
	// DVMAddressLength defines default Move address length.
	DVMAddressLength = 20

	// DVMGasPrice is a gas unit price for DVM execution.
	DVMGasPrice = 1
	// DVMGasLimit defines the max gas value for DVM execution.
	DVMGasLimit = ^uint64(0)/1000 - 1
)

// DVM event to Event conversion params.
const (
	// EventTypeProcessingGas is the initial gas for processing event type.
	EventTypeProcessingGas = 10000
	// EventTypeNoGasLevels defines number of nesting levels that do not charge gas.
	EventTypeNoGasLevels = 2
)

var (
	// DVMStdLibAddress is the Move stdlib address.
	DVMStdLibAddress = make([]byte, DVMAddressLength)

	// DVMStdLibAddressShortStr is the Move stdlib address string representation.
	DVMStdLibAddressShortStr = "0x1"
)

func init() {
	DVMStdLibAddress[DVMAddressLength-1] = 1
}
