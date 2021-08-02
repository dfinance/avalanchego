package state

import (
	"bytes"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

var (
	writeSetsDBPrefix = []byte("writeSets")
	singletonDBPrefix = []byte("singleton")

	initializedKey = []byte("initialized")

	keyDelimiter = []byte(":")
)

func NewWriteSetStorageKey(path *dvm.VMAccessPath) []byte {
	if path == nil {
		return nil
	}

	return bytes.Join(
		[][]byte{
			path.Address,
			path.Path,
		},
		keyDelimiter,
	)
}
