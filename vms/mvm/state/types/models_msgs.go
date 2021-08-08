package types

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

// Msg defines interface for DVM request messages.
type Msg interface {
	// Validate performs a basic message validation.
	Validate() error
}

// MsgExecuteScript defines a message to execute a script with args using DVM.
type (
	MsgExecuteScript struct {
		Sender []byte      `serialize:"true" json:"sender"` // Sender address
		Script []byte      `serialize:"true" json:"script"` // Script Byte code
		Args   []ScriptArg `serialize:"true" json:"args"`   // Script arguments
	}

	ScriptArg struct {
		Type  dvm.VMTypeTag `serialize:"true" json:"type"`
		Value []byte        `serialize:"true" json:"value"`
	}
)

// MsgDeployModule defines a message to deploy a module (contract) using DVM.
type MsgDeployModule struct {
	Sender  []byte   `serialize:"true" json:"sender"`  // Sender address
	Modules [][]byte `serialize:"true" json:"modules"` // Modules byte code
}

// MsgCompile defines a message to compile Move code.
type MsgCompile struct {
	Sender []byte `serialize:"true" json:"sender"` // Sender address
	Code   []byte `serialize:"true" json:"code"`   // Source code
}

// MsgGetMetadata defines a message to get Move code metadata.
type MsgGetMetadata struct {
	Code []byte `serialize:"true" json:"code"` // Source code
}

// Validate implements Msg interface.
func (m MsgExecuteScript) Validate() error {
	if len(m.Sender) != DVMAddressLength {
		return fmt.Errorf("sender: invalid length (should be %d)", DVMAddressLength)
	}

	if len(m.Script) == 0 {
		return fmt.Errorf("script: empty")
	}

	for i, arg := range m.Args {
		if _, err := StringifyDVMTypeTag(arg.Type); err != nil {
			return fmt.Errorf("args [%d]: type: %w", i, err)
		}
		if len(arg.Value) == 0 {
			return fmt.Errorf("args [%d]: value: empty", i)
		}
	}

	return nil
}

// Validate implements Msg interface.
func (m MsgDeployModule) Validate() error {
	if len(m.Sender) != DVMAddressLength {
		return fmt.Errorf("sender: invalid length (should be %d)", DVMAddressLength)
	}

	if len(m.Modules) == 0 {
		return fmt.Errorf("modules: empty")
	}
	for i, module := range m.Modules {
		if len(module) == 0 {
			return fmt.Errorf("modules [%d]: empty", i)
		}
	}

	return nil
}

// Validate implements Msg interface.
func (m MsgCompile) Validate() error {
	if len(m.Sender) != DVMAddressLength {
		return fmt.Errorf("sender: invalid length (should be %d)", DVMAddressLength)
	}

	if len(m.Code) == 0 {
		return fmt.Errorf("code: empty")
	}

	return nil
}

// Validate implements Msg interface.
func (m MsgGetMetadata) Validate() error {
	if len(m.Code) == 0 {
		return fmt.Errorf("code: empty")
	}

	return nil
}

// MsgBuilder is a Msg build helper.
type MsgBuilder struct {
	msg      Msg
	buildErr error

	getMetadata  func(*MsgGetMetadata) (*Metadata, error)
	parseAddress func(string) (ids.ShortID, error)
}

// NewMsgBuilder creates a new MsgBuilder instance.
func NewMsgBuilder() *MsgBuilder {
	return &MsgBuilder{}
}

// WithMetadataGetter sets script metadate getter.
func (b *MsgBuilder) WithMetadataGetter(getter func(*MsgGetMetadata) (*Metadata, error)) *MsgBuilder {
	b.getMetadata = getter

	return b
}

// WithAddressParser sets address parser.
func (b *MsgBuilder) WithAddressParser(parser func(string) (ids.ShortID, error)) *MsgBuilder {
	b.parseAddress = parser

	return b
}

// ExecuteScript builds the MsgExecuteScript message.
func (b *MsgBuilder) ExecuteScript(senderAddress ids.ShortID, compiledItems CompiledItems, argValues ...string) *MsgBuilder {
	if b.getMetadata == nil {
		b.buildErr = fmt.Errorf("metadata getter not configured")
		return b
	}

	if err := b.validateCompiledContent(compiledItems, true); err != nil {
		b.buildErr = fmt.Errorf("validating compiled items: %w", err)
		return b
	}

	meta, err := b.getMetadata(&MsgGetMetadata{Code: compiledItems[0].ByteCode})
	if err != nil {
		b.buildErr = fmt.Errorf("extracting script arguments meta: %w", err)
		return b
	}
	if meta.Metadata.GetScript() == nil {
		b.buildErr = fmt.Errorf("extracting script arguments meta: byteCode is not a script")
		return b
	}
	argTypes := meta.Metadata.GetScript().Arguments

	if len(argValues) != len(argTypes) {
		b.buildErr = fmt.Errorf("argValues / argTypes length mismatch: %d / %d", len(argValues), len(argTypes))
		return b
	}

	args := make([]ScriptArg, 0, len(argValues))
	for argIdx, argValue := range argValues {
		arg, err := b.buildScriptArg(argValue, argTypes[argIdx])
		if err != nil {
			b.buildErr = fmt.Errorf("argument [%d] (%s): %w", argIdx, argValue, err)
			return b
		}
		args = append(args, arg)
	}

	b.msg = &MsgExecuteScript{
		Sender: senderAddress.Bytes(),
		Script: compiledItems[0].ByteCode,
		Args:   args,
	}

	return b
}

// DeployModule builds the MsgDeployModule message.
func (b *MsgBuilder) DeployModule(senderAddress ids.ShortID, compiledItems CompiledItems) *MsgBuilder {
	if err := b.validateCompiledContent(compiledItems, true); err != nil {
		b.buildErr = fmt.Errorf("validating compiled items: %w", err)
		return b
	}

	contractsCode := make([][]byte, 0, len(compiledItems))
	for _, item := range compiledItems {
		contractsCode = append(contractsCode, item.ByteCode)
	}

	b.msg = &MsgDeployModule{
		Sender:  senderAddress.Bytes(),
		Modules: contractsCode,
	}

	return b
}

// Compile builds the MsgCompile message.
func (b *MsgBuilder) Compile(senderAddress ids.ShortID, srcCode string) *MsgBuilder {
	b.msg = &MsgCompile{
		Sender: senderAddress.Bytes(),
		Code:   []byte(srcCode),
	}

	return b
}

// GetMetadata builds the MsgGetMetadata message.
func (b *MsgBuilder) GetMetadata(byteCode []byte) *MsgBuilder {
	b.msg = &MsgGetMetadata{
		Code: byteCode,
	}

	return b
}

func (b *MsgBuilder) Build() (Msg, error) {
	if b.buildErr != nil {
		return nil, fmt.Errorf("building message (%T): %w", b.msg, b.buildErr)
	}

	if b.msg == nil {
		return nil, fmt.Errorf("message: nil")
	}

	if err := b.msg.Validate(); err != nil {
		return nil, fmt.Errorf("validating message (%T): %w", b.msg, b.buildErr)
	}

	return b.msg, nil
}

func (b *MsgBuilder) validateCompiledContent(items CompiledItems, oneItem bool) error {
	if len(items) == 0 || (oneItem && len(items) != 1) {
		return fmt.Errorf("wrong number of items: %d", len(items))
	}

	itemsCodeType := items[0].CodeType
	for _, item := range items {
		if itemsCodeType != item.CodeType {
			return fmt.Errorf("items have different code types (only simmilar types are allowed)")
		}
	}

	return nil
}

func (b *MsgBuilder) buildScriptArg(argValue string, argType dvm.VMTypeTag) (ScriptArg, error) {
	var arg ScriptArg
	var err error

	switch argType {
	case dvm.VMTypeTag_Address:
		arg, err = b.parseAddressScriptArg(argValue)
	case dvm.VMTypeTag_U8:
		arg, err = b.parseU8ScriptArg(argValue)
	case dvm.VMTypeTag_U64:
		arg, err = b.parseU64ScriptArg(argValue)
	case dvm.VMTypeTag_U128:
		arg, err = b.parseU128ScriptArg(argValue)
	case dvm.VMTypeTag_Bool:
		arg, err = b.parseBoolScriptArg(argValue)
	case dvm.VMTypeTag_Vector:
		arg, err = b.parseVectorScriptArg(argValue)
	default:
		return ScriptArg{}, fmt.Errorf("unsupported argType code: %v", argType)
	}

	if err != nil {
		return ScriptArg{}, err
	}

	return arg, nil
}

func (b *MsgBuilder) parseAddressScriptArg(value string) (ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Address
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	if b.parseAddress == nil {
		return ScriptArg{}, fmt.Errorf("type (%s): address parses not configured", argTypeName)
	}

	if value == "" {
		return ScriptArg{}, fmt.Errorf("type (%s): empty", argTypeName)
	}

	return ScriptArg{
		Type:  argTypeCode,
		Value: []byte(value),
	}, nil
}

func (b *MsgBuilder) parseU8ScriptArg(value string) (ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_U8
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	hashParsedValue, err := b.parseXxHashUint(value)
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}

	uintValue, err := strconv.ParseUint(hashParsedValue, 10, 8)
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}

	return ScriptArg{
		Type:  argTypeCode,
		Value: []byte{uint8(uintValue)},
	}, nil
}

func (b *MsgBuilder) parseU64ScriptArg(value string) (ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_U64
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	hashParsedValue, err := b.parseXxHashUint(value)
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}

	uintValue, err := strconv.ParseUint(hashParsedValue, 10, 64)
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}
	argValue := make([]byte, 8)
	binary.LittleEndian.PutUint64(argValue, uintValue)

	return ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

func (b *MsgBuilder) parseU128ScriptArg(value string) (retArg ScriptArg, retErr error) {
	argTypeCode := dvm.VMTypeTag_U128
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	defer func() {
		if recover() != nil {
			retErr = fmt.Errorf("type (%s): failed", argTypeName)
		}
	}()

	hashParsedValue, err := b.parseXxHashUint(value)
	if err != nil {
		retErr = fmt.Errorf("type (%s): %w", argTypeName, err)
		return
	}

	bigValue, ok := new(big.Int).SetString(hashParsedValue, 0)
	if !ok {
		retErr = fmt.Errorf("type (%s): invalid BigInt value", argTypeName)
		return
	}
	if bigValue.Sign() < 0 {
		retErr = fmt.Errorf("type (%s): non-posititve BigInt value", argTypeName)
		return
	}
	if bigValue.BitLen() > 128 {
		retErr = fmt.Errorf("type (%s): invalid bitLen %d", argTypeName, bigValue.BitLen())
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

	retArg.Type, retArg.Value = argTypeCode, argValue

	return
}

func (b *MsgBuilder) parseVectorScriptArg(value string) (ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Vector
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	if value == "" {
		return ScriptArg{}, fmt.Errorf("type (%s): empty", argTypeName)
	}

	argValue, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}

	return ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

func (b *MsgBuilder) parseBoolScriptArg(value string) (ScriptArg, error) {
	argTypeCode := dvm.VMTypeTag_Bool
	argTypeName := dvm.VMTypeTag_name[int32(argTypeCode)]

	valueBool, err := strconv.ParseBool(value)
	if err != nil {
		return ScriptArg{}, fmt.Errorf("type (%s): %w", argTypeName, err)
	}

	argValue := []byte{0}
	if valueBool {
		argValue[0] = 1
	}

	return ScriptArg{
		Type:  argTypeCode,
		Value: argValue,
	}, nil
}

func (b *MsgBuilder) parseXxHashUint(value string) (string, error) {
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
