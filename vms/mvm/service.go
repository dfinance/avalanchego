package mvm

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

type Service struct {
	vm *VM
}

type (
	CompileRequest struct {
		SenderAddress string `json:"sender_address"`
		MoveCode      string `json:"move_code"`
	}

	CompileResponse struct {
		CompiledItems []types.CompiledItem `json:"compiled_items"`
	}
)

type (
	DeployRequest struct {
		SenderAddress   string `json:"sender_address"`
		CompiledContent string `json:"compiled_content"`
	}

	DeployResponse struct{}
)

type (
	ExecuteRequest struct {
		SenderAddress   string   `json:"sender_address"`
		CompiledContent string   `json:"compiled_content"`
		Args            []string `json:"args"`
	}

	ExecuteResponse struct{}
)

func (s *Service) Compile(_ *http.Request, args *CompileRequest, reply *CompileResponse) error {
	resp, err := s.vm.state.Compile([]byte(args.SenderAddress), []byte(args.MoveCode))
	if err != nil {
		return err
	}

	reply.CompiledItems = resp

	return nil
}

func (s *Service) Deploy(_ *http.Request, args *DeployRequest, reply *DeployResponse) error {
	compItems, err := s.parseCompiledContent(args.CompiledContent, true)
	if err != nil {
		return fmt.Errorf("compiled_content: %v: %w", err, types.ErrInvalidInput)
	}

	contractsCode := make([][]byte, 0, len(compItems.CompiledItems))
	for _, item := range compItems.CompiledItems {
		contractsCode = append(contractsCode, item.ByteCode)
	}
	tx := types.TxDeployModule{
		Signer:  args.SenderAddress,
		Modules: contractsCode,
	}

	s.vm.proposeBlock([]types.Tx{tx})

	return nil
}

func (s *Service) Execute(_ *http.Request, args *ExecuteRequest, reply *ExecuteResponse) error {
	compItems, err := s.parseCompiledContent(args.CompiledContent, true)
	if err != nil {
		return fmt.Errorf("compiled_content: %v: %w", err, types.ErrInvalidInput)
	}

	meta, err := s.vm.state.GetMetadata(compItems.CompiledItems[0].ByteCode)
	if err != nil {
		return fmt.Errorf("extracting script arguments meta: %w", err)
	}
	if meta.Metadata.GetScript() == nil {
		return fmt.Errorf("extracting script arguments meta: requested byteCode is not a script")
	}
	typedArgs := meta.Metadata.GetScript().Arguments

	// Build msg
	scriptArgs, err := s.convertScriptArgs(args.Args, typedArgs)
	if err != nil {
		return fmt.Errorf("converting input args to typed args: %w", err)
	}
	tx := types.TxExecuteScript{
		Signer: args.SenderAddress,
		Script: compItems.CompiledItems[0].ByteCode,
		Args:   scriptArgs,
	}

	s.vm.proposeBlock([]types.Tx{tx})

	return nil
}

func (s *Service) parseCompiledContent(content string, oneItem bool) (*CompileResponse, error) {
	compItems := CompileResponse{}
	if err := json.Unmarshal([]byte(content), &compItems); err != nil {
		return nil, fmt.Errorf("content JSON unmarshal: %w", err)
	}

	if len(compItems.CompiledItems) == 0 || (oneItem && len(compItems.CompiledItems) != 1) {
		return nil, fmt.Errorf("content has wrong number of items (%d)", len(compItems.CompiledItems))
	}

	itemsCodeType := compItems.CompiledItems[0].CodeType
	for _, item := range compItems.CompiledItems {
		if itemsCodeType != item.CodeType {
			return nil, fmt.Errorf("content has different code types (only simmilar types are allowed)")
		}
	}

	return &compItems, nil
}

func (s *Service) convertScriptArgs(argStrs []string, argTypes []dvm.VMTypeTag) ([]types.ScriptArg, error) {
	if len(argStrs) != len(argTypes) {
		return nil, fmt.Errorf("strArgs / typedArgs length mismatch: %d / %d", len(argStrs), len(argTypes))
	}

	scriptArgs := make([]types.ScriptArg, len(argStrs))
	for argIdx, argStr := range argStrs {
		argType := argTypes[argIdx]
		var scriptArg types.ScriptArg
		var err error

		switch argType {
		case dvm.VMTypeTag_Address:
			scriptArg, err = newAddressScriptArg(argStr)
		case dvm.VMTypeTag_U8:
			scriptArg, err = newU8ScriptArg(argStr)
		case dvm.VMTypeTag_U64:
			scriptArg, err = newU64ScriptArg(argStr)
		case dvm.VMTypeTag_U128:
			scriptArg, err = newU128ScriptArg(argStr)
		case dvm.VMTypeTag_Bool:
			scriptArg, err = newBoolScriptArg(argStr)
		case dvm.VMTypeTag_Vector:
			scriptArg, err = newVectorScriptArg(argStr)
		default:
			return nil, fmt.Errorf("argument [%d]: parsing argument (%s): unsupported argType code: %v", argIdx, argStr, argType)
		}

		if err != nil {
			return nil, fmt.Errorf("argument [%d]: %w", argIdx, err)
		}
		scriptArgs[argIdx] = scriptArg
	}

	return scriptArgs, nil
}

// newAddressScriptArg convert string to address ScriptTag.
func newAddressScriptArg(value string) (types.ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Address
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	if value == "" {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: empty", value, argTypeName)
	}

	return types.ScriptArg{
		Type:  argTypeCode,
		Value: []byte(value),
	}, nil
}

// newU8ScriptArg convert string to U8 ScriptTag.
func newU8ScriptArg(value string) (types.ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_U8
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	hashParsedValue, err := parseXxHashUint(value)
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}

	uintValue, err := strconv.ParseUint(hashParsedValue, 10, 8)
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}

	return types.ScriptArg{
		Type:  argTypeCode,
		Value: []byte{uint8(uintValue)},
	}, nil
}

// newU64ScriptArg convert string to U64 ScriptTag.
func newU64ScriptArg(value string) (types.ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_U64
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	hashParsedValue, err := parseXxHashUint(value)
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}

	uintValue, err := strconv.ParseUint(hashParsedValue, 10, 64)
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}
	argValue := make([]byte, 8)
	binary.LittleEndian.PutUint64(argValue, uintValue)

	return types.ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

// newU128ScriptArg convert string to U128 ScriptTag.
func newU128ScriptArg(value string) (retTag types.ScriptArg, retErr error) {
	argTypeCode := dvm.VMTypeTag_U128
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	defer func() {
		if recover() != nil {
			retErr = fmt.Errorf("parsing argument %q of type %q: failed", value, argTypeName)
		}
	}()

	hashParsedValue, err := parseXxHashUint(value)
	if err != nil {
		retErr = fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
		return
	}

	bigValue, ok := new(big.Int).SetString(hashParsedValue, 0)
	if !ok {
		retErr = fmt.Errorf("parsing argument %q of type %q: invalid BigInt value", value, argTypeName)
		return
	}
	if bigValue.Sign() < 0 {
		retErr = fmt.Errorf("parsing argument %q of type %q: non-posititve BigInt value", value, argTypeName)
		return
	}
	if bigValue.BitLen() > 128 {
		retErr = fmt.Errorf("parsing argument %q of type %q: invalid bitLen %d", value, argTypeName, bigValue.BitLen())
		return
	}

	// BigInt().Bytes() returns BigEndian format, reverse it
	argValue := bigValue.Bytes()
	for left, right := 0, len(argValue)-1; left < right; left, right = left+1, right-1 {
		argValue[left], argValue[right] = argValue[right], argValue[left]
	}

	// Extend to 16 bytes
	if len(argValue) < 16 {
		zeros := make([]byte, 16-len(argValue))
		argValue = append(argValue, zeros...)
	}

	retTag.Type, retTag.Value = argTypeCode, argValue

	return
}

// newVectorScriptArg convert string to Vector ScriptTag.
func newVectorScriptArg(value string) (types.ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Vector
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	if value == "" {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: empty", value, argTypeName)
	}

	argValue, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}

	return types.ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

// newBoolScriptArg convert string to Bool ScriptTag.
func newBoolScriptArg(value string) (types.ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Bool
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	valueBool, err := strconv.ParseBool(value)
	if err != nil {
		return types.ScriptArg{}, fmt.Errorf("parsing argument %q of type %q: %w", value, argTypeName, err)
	}

	argValue := []byte{0}
	if valueBool {
		argValue[0] = 1
	}

	return types.ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

// parseXxHashUint converts (or skips) xxHash integer format.
func parseXxHashUint(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("xxHash parsing: empty")
	}

	if value[0] == '#' {
		seed := xxhash.NewS64(0)
		if len(value) < 2 {
			return "", fmt.Errorf("xxHash parsing: invalid length")
		}

		if _, err := seed.WriteString(strings.ToLower(value[1:])); err != nil {
			return "", fmt.Errorf("xxHash parsing: %w", err)
		}
		value = strconv.FormatUint(seed.Sum64(), 10)
	}

	return value, nil
}
