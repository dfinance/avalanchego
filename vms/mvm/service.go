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

type (
	DeployRequest struct {
		api.UserPass

		CompiledContent string `json:"compiledContent"`
	}
)

type (
	ExecuteRequest struct {
		api.UserPass

		CompiledContent string   `json:"compiledContent"`
		Args            []string `json:"args"`
	}
)

type (
	ImportKeyArgs struct {
		api.UserPass

		PrivateKey string `json:"privateKey"`
	}
)

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

type ExecutionResponse struct {
	Executed bool              `json:"executed"`
	Message  string            `json:"message,omitempty"`
	Events   stateTypes.Events `json:"events,omitempty"`
}

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

	s.vm.issueTx(tx)

	return nil
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

	s.vm.issueTx(tx)

	return nil
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
