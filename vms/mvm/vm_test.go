package mvm

import (
	"testing"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/stretchr/testify/assert"
)

func TestPrivKeyToBech32(t *testing.T) {
	privKeyBytes, err := formatting.Decode(formatting.CB58, "117pLajkLYsgQjQgXS7UsyfAvm1tmpbj2sE1W694smgM2otgfeqx2MfJPJghk3zYkKbYEsKDfzteeRP")
	assert.NoError(t, err)

	factory := crypto.FactorySECP256K1R{}
	skIntf, err := factory.ToPrivateKey(privKeyBytes)
	assert.NoError(t, err)
	sk := skIntf.(*crypto.PrivateKeySECP256K1R)

	addr := sk.PublicKey().Address()

	addrStr, err := formatting.FormatBech32(constants.FallbackHRP, addr.Bytes())
	assert.NoError(t, err)

	t.Log(addrStr)
}
