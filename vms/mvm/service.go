package mvm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
	stateTypes "github.com/ava-labs/avalanchego/vms/mvm/state/types"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

// Service provides VM's REST API handlers.
type Service struct {
	vm *VM
}

type (
	CompileRequest struct {
		api.UserPass

		MoveCode string `json:"moveCode"`
	}

	CompileResponse struct {
		CompiledItems stateTypes.CompiledItems `json:"compiledItems"`
	}
)

// Compile compiles Move code and returns byte code with DVM meta.
func (s *Service) Compile(_ *http.Request, args *CompileRequest, reply *CompileResponse) error {
	s.vm.Ctx.Log.Debug("MVM: Compile called")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	address, _, err := s.getUserCreds(args.UserPass)
	if err != nil {
		return err
	}

	msg, err := stateTypes.NewMsgBuilder().
		Compile(address, args.MoveCode).
		Build()
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidInput)
	}

	resp, err := s.vm.state.Compile(msg.(*stateTypes.MsgCompile))
	if err != nil {
		return err
	}

	reply.CompiledItems = resp

	return nil
}

type DeployRequest struct {
	api.UserPass

	CompiledContent string `json:"compiledContent"`
}

// Deploy issues contract deploy Tx.
func (s *Service) Deploy(_ *http.Request, args *DeployRequest, reply *api.JSONTxID) error {
	s.vm.Ctx.Log.Debug("MVM: Deploy called")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	address, privKeys, err := s.getUserCreds(args.UserPass)
	if err != nil {
		return err
	}

	compItems, err := s.parseCompiledContent(args.CompiledContent)
	if err != nil {
		return err
	}

	msg, err := stateTypes.NewMsgBuilder().
		DeployModule(address, compItems).
		Build()
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidInput)
	}

	tx, err := s.vm.newMoveTx(msg, privKeys)
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidInput)
	}

	if err := s.vm.issueTx(tx); err != nil {
		return fmt.Errorf("issuing Tx: %w", err)
	}
	reply.TxID = tx.ID()

	return nil
}

type ExecuteRequest struct {
	api.UserPass

	CompiledContent string   `json:"compiledContent"`
	Args            []string `json:"args"`
}

// Execute issues contract execute Tx.
func (s *Service) Execute(_ *http.Request, args *ExecuteRequest, reply *api.JSONTxID) error {
	s.vm.Ctx.Log.Debug("MVM: Execute called")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	address, privKeys, err := s.getUserCreds(args.UserPass)
	if err != nil {
		return err
	}

	compItems, err := s.parseCompiledContent(args.CompiledContent)
	if err != nil {
		return err
	}

	msg, err := stateTypes.NewMsgBuilder().
		WithMetadataGetter(s.vm.state.GetMetadata).
		WithAddressParser(s.vm.ParseLocalAddress).
		ExecuteScript(address, compItems, args.Args...).
		Build()
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidInput)
	}

	tx, err := s.vm.newMoveTx(msg, privKeys)
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidInput)
	}

	if err := s.vm.issueTx(tx); err != nil {
		return fmt.Errorf("issuing Tx: %w", err)
	}
	reply.TxID = tx.ID()

	return nil
}

type (
	GetDataRequest struct {
		DVMAddress string `json:"dvmAddress"`
		DVMPath    string `json:"dvmPath"`
	}

	GetDataResponse struct {
		Data []byte `json:"data,omitempty"`
	}
)

// GetData returns stored writeSet data.
func (s *Service) GetData(_ *http.Request, args *GetDataRequest, reply *GetDataResponse) error {
	s.vm.Ctx.Log.Debug("MVM: Get data")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	parseHexString := func(v string) ([]byte, error) {
		v = strings.TrimPrefix(v, "0x")
		bytes, err := hex.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("parsing HEX string: %w", err)
		}

		return bytes, nil
	}

	address, err := parseHexString(args.DVMAddress)
	if err != nil {
		return fmt.Errorf("parsing dvmAddress: %v: %w", err, ErrInvalidInput)
	}

	path, err := parseHexString(args.DVMPath)
	if err != nil {
		return fmt.Errorf("parsing dvmPath: %v: %w", err, ErrInvalidInput)
	}

	data, err := s.vm.state.GetWriteSetData(address, path)
	if err != nil {
		return err
	}
	reply.Data = data

	return nil
}

type ImportKeyArgs struct {
	api.UserPass

	PrivateKey string `json:"privateKey"`
}

// ImportKey imports user private key, creating address.
func (s *Service) ImportKey(_ *http.Request, args *ImportKeyArgs, reply *api.JSONAddress) error {
	s.vm.Ctx.Log.Debug("MVM: ImportKey called for user %s", args.Username)
	if err := s.checkInitialized(); err != nil {
		return err
	}

	if !strings.HasPrefix(args.PrivateKey, constants.SecretKeyPrefix) {
		return fmt.Errorf("private key missing %s prefix: %w", constants.SecretKeyPrefix, ErrInvalidInput)
	}

	prvKeyBytes, err := formatting.Decode(formatting.CB58, strings.TrimPrefix(args.PrivateKey, constants.SecretKeyPrefix))
	if err != nil {
		return fmt.Errorf("parsing private key: %v: %w", err, ErrInvalidInput)
	}

	prvKey, err := s.vm.factory.ToPrivateKey(prvKeyBytes)
	if err != nil {
		return fmt.Errorf("parsing private key: %v: %w", err, ErrInvalidInput)
	}

	user, err := s.vm.getUserSvc(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("retrieving keystore data: %w", err)
	}
	defer user.Close()

	addresses, err := user.GetAddresses()
	if err != nil {
		return err
	}
	if len(addresses) > 0 {
		return fmt.Errorf("keystore user can have only one address")
	}

	sk := prvKey.(*crypto.PrivateKeySECP256K1R)

	address, err := s.vm.FormatLocalAddress(sk.PublicKey().Address())
	if err != nil {
		return fmt.Errorf("formatting address: %w", err)
	}
	if err := user.PutAddress(sk); err != nil {
		return err
	}

	reply.Address = address

	return user.Close()
}

type (
	AddressListResponse struct {
		Addresses []AddressResponse `json:"addresses"`
	}

	AddressResponse struct {
		CB58  string `json:"cb58_format"`
		Local string `json:"local_format"`
		Hex   string `json:"hex_format"`
	}
)

// ListAddresses returns addresses controlled by user.
func (s *Service) ListAddresses(_ *http.Request, args *api.UserPass, reply *AddressListResponse) error {
	s.vm.Ctx.Log.Debug("MVM: ListAddresses called")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	user, err := s.vm.getUserSvc(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("retrieving keystore data: %w", err)
	}
	defer user.Close()

	prvKeys, err := user.GetKeys()
	if err != nil {
		return err
	}

	for idx, prvKey := range prvKeys {
		address := prvKey.PublicKey().Address()
		addressLocal, err := s.vm.FormatLocalAddress(address)
		if err != nil {
			return fmt.Errorf("private key [%d]: formatting to local address: %w", idx, err)
		}

		reply.Addresses = append(reply.Addresses, AddressResponse{
			CB58:  address.String(),
			Local: addressLocal,
			Hex:   hex.EncodeToString(address.Bytes()),
		})
	}

	return user.Close()
}

type (
	GetTxStatusRequest struct {
		TxID ids.ID `json:"txID"`
	}

	GetTxStatusResponse struct {
		TxState *types.TxState `json:"txState"`
	}
)

// GetTxStatus gets a Tx state.
func (s *Service) GetTxStatus(_ *http.Request, args *GetTxStatusRequest, reply *GetTxStatusResponse) error {
	s.vm.Ctx.Log.Debug("MVM: GetTxStatus called")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	txState, err := s.vm.txStorage.GetTxState(args.TxID)
	if err != nil {
		return err
	}
	if txState != nil {
		reply.TxState = txState
		return nil
	}

	preferredBlockRaw, err := s.vm.GetBlock(s.vm.Preferred())
	if err != nil {
		return fmt.Errorf("reading preferred block: %w", err)
	}
	preferredBlock, ok := preferredBlockRaw.(*Block)
	if !ok {
		return fmt.Errorf("reading preferred block: invalid type: %T", preferredBlockRaw)
	}

	for _, tx := range preferredBlock.Txs {
		if tx.ID() == args.TxID {
			reply.TxState = &types.TxState{
				Tx:       *tx,
				TxStatus: types.TxStatusProcessing,
				Events:   nil,
			}

			return nil
		}
	}

	return nil
}

type GetHeightResponse struct {
	Height uint64 `json:"height"`
}

// GetHeight returns the height of the last accepted block
func (s *Service) GetHeight(_ *http.Request, _ *struct{}, response *GetHeightResponse) error {
	s.vm.Ctx.Log.Debug("MVM: Get height")
	if err := s.checkInitialized(); err != nil {
		return err
	}

	lastAcceptedBlockID, err := s.vm.LastAccepted()
	if err != nil {
		return fmt.Errorf("reading last accepted block ID: %w", err)
	}

	lastAcceptedBlock, err := s.vm.GetBlock(lastAcceptedBlockID)
	if err != nil {
		return fmt.Errorf("reading last accepted block: %w", err)
	}
	response.Height = lastAcceptedBlock.Height()

	return nil
}

func (s *Service) checkInitialized() error {
	if err := s.vm.CheckInitialized(); err != nil {
		return err
	}

	return nil
}

func (s *Service) getUserCreds(args api.UserPass) (ids.ShortID, []*crypto.PrivateKeySECP256K1R, error) {
	user, err := s.vm.getUserSvc(args.Username, args.Password)
	if err != nil {
		return ids.ShortID{}, nil, fmt.Errorf("retrieving keystore data: %w", err)
	}
	defer user.Close()

	addresses, err := user.GetAddresses()
	if err != nil {
		return ids.ShortID{}, nil, err
	}
	if len(addresses) == 0 {
		return ids.ShortID{}, nil, fmt.Errorf("user has no addresses in the keystore: %w", ErrInvalidInput)
	}

	address := addresses[0]
	keys, err := user.GetKeys()
	if err != nil {
		return ids.ShortID{}, nil, err
	}

	return address, keys, nil
}

func (s *Service) parseSenderAddress(addressStr string) (ids.ShortID, error) {
	id, err := s.vm.ParseLocalAddress(addressStr)
	if err != nil {
		return ids.ShortID{}, fmt.Errorf("parsing address: %v: %w", err, ErrInvalidInput)
	}

	return id, nil
}

func (s *Service) parseCompiledContent(content string) (stateTypes.CompiledItems, error) {
	resp := CompileResponse{}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("compiledContent: JSON unmarshal: %v: %w", err, ErrInvalidInput)
	}

	return resp.CompiledItems, nil
}
