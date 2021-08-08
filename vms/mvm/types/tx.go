package types

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

var _ UnsignedTx = (*UnsignedMoveTx)(nil)

// UnsignedTx is an unsigned transaction.
type UnsignedTx interface {
	ID() ids.ID
	Initialize(unsignedBytes, signedBytes []byte)
	UnsignedBytes() []byte
	Bytes() []byte
	Validate(ctx *snow.Context) error
}

type UnsignedGenesisTx struct {
	avax.BaseTx `serialize:"true"`
}

func (tx UnsignedGenesisTx) Validate(ctx *snow.Context) error {
	return nil
}

type UnsignedMoveTx struct {
	avax.BaseTx `serialize:"true"`

	Msg stateTypes.Msg `serialize:"true" json:"msg"`
}

func (tx UnsignedMoveTx) Validate(ctx *snow.Context) error {
	if tx.Msg == nil {
		return fmt.Errorf("msg: nil")
	}

	if err := tx.Msg.Validate(); err != nil {
		return fmt.Errorf("msg: validation: %w", err)
	}

	if err := tx.BaseTx.MetadataVerify(ctx); err != nil {
		return fmt.Errorf("metadate verification: %w", err)
	}

	return nil
}

// Tx wraps UnsignedTx with signature.
type Tx struct {
	// The body of this transaction
	UnsignedTx `serialize:"true"`

	// The credentials of this transaction
	Creds []verify.Verifiable `serialize:"true" json:"credentials"`
}

// Sign this transaction with the provided signers
func (tx *Tx) Sign(c codec.Manager, signers [][]*crypto.PrivateKeySECP256K1R) error {
	unsignedBytes, err := c.Marshal(CodecVersion, &tx.UnsignedTx)
	if err != nil {
		return fmt.Errorf("unsigned Tx marshal: %w", err)
	}

	hash := hashing.ComputeHash256(unsignedBytes)

	for _, keys := range signers {
		cred := &secp256k1fx.Credential{
			Sigs: make([][crypto.SECP256K1RSigLen]byte, len(keys)),
		}
		for i, key := range keys {
			sig, err := key.SignHash(hash)
			if err != nil {
				return fmt.Errorf("problem generating credential: %w", err)
			}
			copy(cred.Sigs[i][:], sig)
		}

		tx.Creds = append(tx.Creds, cred)
	}

	signedBytes, err := c.Marshal(CodecVersion, tx)
	if err != nil {
		return fmt.Errorf("signed Tx marshal: %w", err)
	}
	tx.Initialize(unsignedBytes, signedBytes)

	return nil
}
