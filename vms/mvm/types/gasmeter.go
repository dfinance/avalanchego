package types

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrGasOverflow = errors.New("gas overflow")
	ErrGasLimit    = errors.New("gas reached its limit")
	ErrGasNegative = errors.New("negative gas consumed")
)

// GasMeter track gas consumption.
type GasMeter struct {
	limit    uint64
	consumed uint64
}

// NewGasMeter returns a reference to a new basicGasMeter.
func NewGasMeter(limit uint64) *GasMeter {
	return &GasMeter{
		limit:    limit,
		consumed: 0,
	}
}

// GasConsumed returns current amount of consumed gas.
func (g *GasMeter) GasConsumed() uint64 {
	return g.consumed
}

// Limit returns the gas limit.
func (g *GasMeter) Limit() uint64 {
	return g.limit
}

// ConsumeGas consumes gas checking for overflow or reaching the limit.
func (g *GasMeter) ConsumeGas(amount uint64, description string) error {
	consumed, overflow := addUint64WithOverflowCheck(g.consumed, amount)
	if overflow {
		return fmt.Errorf("%s: %w", description, ErrGasOverflow)
	}
	g.consumed = consumed

	if g.consumed > g.limit {
		return fmt.Errorf("%s: %w", description, ErrGasLimit)
	}

	return nil
}

// RefundGas deducts the given amount from the gas consumed checking if it can be refunded.
func (g *GasMeter) RefundGas(amount uint64, descriptor string) error {
	if g.consumed < amount {
		return fmt.Errorf("%s: %w", descriptor, ErrGasNegative)
	}
	g.consumed -= amount

	return nil
}

// addUint64WithOverflowCheck sums two uint64 values checking if overflow occurred.
func addUint64WithOverflowCheck(a, b uint64) (uint64, bool) {
	if math.MaxUint64-a < b {
		return 0, true
	}

	return a + b, false
}
