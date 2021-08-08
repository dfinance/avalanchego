package mvm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/database/encdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// addressesKey defines ID to store the list of addresses this user controls.
var addressesKey = ids.Empty[:]

type userSvc struct {
	codec codec.Manager
	db    *encdb.Database
}

// Close closes keystore database.
func (svc *userSvc) Close() error {
	if err := svc.db.Close(); err != nil {
		return fmt.Errorf("closing keystore db")
	}

	return nil
}

// GetAddresses returns addresses controlled by this user.
func (svc *userSvc) GetAddresses() ([]ids.ShortID, error) {
	hasAddresses, err := svc.db.Has(addressesKey)
	if err != nil {
		return nil, fmt.Errorf("checking address list exists: %w", err)
	}
	if !hasAddresses {
		return nil, nil
	}

	addressesBz, err := svc.db.Get(addressesKey)
	if err != nil {
		return nil, fmt.Errorf("getting address list: %w", err)
	}

	addresses := make([]ids.ShortID, 0)
	if _, err := svc.codec.Unmarshal(addressesBz, &addresses); err != nil {
		return nil, fmt.Errorf("address list unmarshal: %w", err)
	}

	return addresses, nil
}

// PutAddress persists that this user controls address controlled by [privKey].
func (svc *userSvc) PutAddress(privKey *crypto.PrivateKeySECP256K1R) error {
	if privKey == nil {
		return fmt.Errorf("pirvate key: nil")
	}
	address := privKey.PublicKey().Address()

	controlsAddress, err := svc.hasAddress(address)
	if err != nil {
		return fmt.Errorf("checking if user controls address: %w", err)
	}
	if controlsAddress {
		return nil
	}

	if err := svc.db.Put(address.Bytes(), privKey.Bytes()); err != nil {
		return fmt.Errorf("saving user address and privKey pair: %w", err)
	}

	addresses, err := svc.GetAddresses()
	if err != nil {
		return err
	}
	addresses = append(addresses, address)

	addressesBz, err := svc.codec.Marshal(types.CodecVersion, addresses)
	if err != nil {
		return fmt.Errorf("address list marshal: %w", err)
	}
	if err := svc.db.Put(addressesKey, addressesBz); err != nil {
		return fmt.Errorf("saving user address list: %w", err)
	}

	return nil
}

// GetKey returns the private key that controls the given address.
func (svc *userSvc) GetKey(address ids.ShortID) (*crypto.PrivateKeySECP256K1R, error) {
	factory := crypto.FactorySECP256K1R{}
	keyBz, err := svc.db.Get(address.Bytes())
	if err != nil {
		return nil, fmt.Errorf("getting private key data: %w", err)
	}

	key, err := factory.ToPrivateKey(keyBz)
	if err != nil {
		return nil, fmt.Errorf("converting key data to private key: %w", err)
	}

	sk, ok := key.(*crypto.PrivateKeySECP256K1R)
	if !ok {
		return nil, fmt.Errorf("expected private key to be type *crypto.PrivateKeySECP256K1R but is type %T", key)
	}

	return sk, nil
}

// GetKeys return all private keys controlled by this user.
func (svc *userSvc) GetKeys() ([]*crypto.PrivateKeySECP256K1R, error) {
	addresses, err := svc.GetAddresses()
	if err != nil {
		return nil, err
	}

	keys := make([]*crypto.PrivateKeySECP256K1R, 0, len(addresses))
	for i, address := range addresses {
		key, err := svc.GetKey(address)
		if err != nil {
			return nil, fmt.Errorf("key [%d]: %w", i, err)
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// hasAddress returns true if this user controls the given address.
func (svc *userSvc) hasAddress(address ids.ShortID) (bool, error) {
	controls, err := svc.db.Has(address.Bytes())
	if err != nil {
		return false, fmt.Errorf("checking address exists: %w", err)
	}

	return controls, nil
}
