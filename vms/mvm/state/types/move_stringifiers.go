package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

// StringifyVMAccessPath returns dvm.VMAccessPath string representation.
func StringifyVMAccessPath(path *dvm.VMAccessPath) string {
	if path == nil {
		return ""
	}

	return fmt.Sprintf("%s:%s", hex.EncodeToString(path.Address), hex.EncodeToString(path.Path))
}

// StringifyDVMTypeTag returns dvm.VMTypeTag string representation.
func StringifyDVMTypeTag(tag dvm.VMTypeTag) (string, error) {
	val, ok := dvm.VMTypeTag_name[int32(tag)]
	if !ok {
		return "", fmt.Errorf("can't find string representation of VMTypeTag %d, check correctness of type value", tag)
	}

	return val, nil
}

// StringifySenderAddress converts ВVM address to string (0x1 for stdlib and wallet1... otherwise).
func StringifySenderAddress(addr []byte) string {
	if bytes.Equal(addr, DVMStdLibAddress) {
		return DVMStdLibAddressShortStr
	} else {
		return hex.EncodeToString(addr)
	}
}

// StringifyEventType returns dvm.LcsTag Move serialization.
// Func is similar to StringifyVMLCSTag, but result is one lined Move representation.
func StringifyEventType(gasMeter *GasMeter, tag *dvm.LcsTag) (string, error) {
	// Start with initial gas for first event, and then go in progression based on depth.
	eventType, err := processEventType(gasMeter, tag, EventTypeProcessingGas, 1)
	if err != nil {
		debugMsg := ""
		if tagStr, err := StringifyVMLCSTag(tag); err != nil {
			debugMsg = fmt.Sprintf("StringifyVMLCSTag failed (%v)", err)
		} else {
			debugMsg = tagStr
		}

		return "", fmt.Errorf("EventType serialization failed: %s: %w", debugMsg, err)
	}

	return eventType, nil
}

// StringifyVMLCSTag converts dvm.LcsTag to string representation (recursive).
// <indentCount> defines number of prefixed indent string for each line.
func StringifyVMLCSTag(tag *dvm.LcsTag, indentCount ...int) (string, error) {
	const strIndent = "  "

	curIndentCount := 0
	if len(indentCount) > 1 {
		return "", fmt.Errorf("invalid indentCount length")
	}
	if len(indentCount) == 1 {
		curIndentCount = indentCount[0]
	}
	if curIndentCount < 0 {
		return "", fmt.Errorf("invalid indentCount")
	}

	strBuilder := strings.Builder{}

	// Helper funcs
	buildStrIndent := func() string {
		str := ""
		for i := 0; i < curIndentCount; i++ {
			str += strIndent
		}
		return str
	}

	buildErr := func(comment string, err error) error {
		return fmt.Errorf("indent %d: %s: %w", curIndentCount, comment, err)
	}

	buildLcsTypeStr := func(t dvm.LcsType) (string, error) {
		val, ok := dvm.LcsType_name[int32(t)]
		if !ok {
			return "", fmt.Errorf("can't find string representation of LcsTag %d, check correctness of type value", t)
		}
		return val, nil
	}

	// Print current tag with recursive func call for fields
	if tag == nil {
		strBuilder.WriteString("nil")
		return strBuilder.String(), nil
	}

	indentStr := buildStrIndent()
	strBuilder.WriteString("LcsTag:\n")

	// Field: TypeTag
	typeTagStr, err := buildLcsTypeStr(tag.TypeTag)
	if err != nil {
		return "", buildErr("TypeTag", err)
	}
	strBuilder.WriteString(fmt.Sprintf("%sTypeTag: %s\n", indentStr, typeTagStr))

	// Field: VectorType
	vectorTypeStr, err := StringifyVMLCSTag(tag.VectorType, curIndentCount+1)
	if err != nil {
		return "", buildErr("VectorType", err)
	}
	strBuilder.WriteString(fmt.Sprintf("%sVectorType: %s\n", indentStr, vectorTypeStr))

	// Field: StructIdent
	if tag.StructIdent != nil {
		strBuilder.WriteString(fmt.Sprintf("%sStructIdent.Address: %s\n", indentStr, hex.EncodeToString(tag.StructIdent.Address)))
		strBuilder.WriteString(fmt.Sprintf("%sStructIdent.Module: %s\n", indentStr, tag.StructIdent.Module))
		strBuilder.WriteString(fmt.Sprintf("%sStructIdent.Name: %s\n", indentStr, tag.StructIdent.Name))
		if len(tag.StructIdent.TypeParams) > 0 {
			for structParamIdx, structParamTag := range tag.StructIdent.TypeParams {
				structParamTagStr, err := StringifyVMLCSTag(structParamTag, curIndentCount+1)
				if err != nil {
					return "", buildErr(fmt.Sprintf("StructIdent.TypeParams[%d]", structParamIdx), err)
				}
				strBuilder.WriteString(fmt.Sprintf("%sStructIdent.TypeParams[%d]: %s", indentStr, structParamIdx, structParamTagStr))
				if structParamIdx < len(tag.StructIdent.TypeParams)-1 {
					strBuilder.WriteString("\n")
				}
			}
		} else {
			strBuilder.WriteString(fmt.Sprintf("%sStructIdent.TypeParams: empty", indentStr))
		}
	} else {
		strBuilder.WriteString(fmt.Sprintf("%sStructIdent: nil", indentStr))
	}

	return strBuilder.String(), nil
}

// StringifyVMStatusMajorCode returns dvm.VMStatus majorCode string representation.
func StringifyVMStatusMajorCode(majorCode string) string {
	if v, ok := dvmErrCodes[majorCode]; ok {
		return v
	}

	return VMErrUnknown
}

// processEventType recursively processes event type and returns result event type as a string.
// If {depth} < 0 we do not charge gas as some nesting levels might be "free".
func processEventType(gasMeter *GasMeter, tag *dvm.LcsTag, gas, depth uint64) (string, error) {
	// We can't consume gas later (after recognizing the type), because it open doors for security holes.
	// Let's say dev will create type with a lot of generics, so transaction will take much more time to process.
	// In result it could be a situation when validator doesn't have enough time to process transaction.
	// Charging gas amount is geometry increased from depth to depth.

	if depth > EventTypeNoGasLevels {
		gas += EventTypeProcessingGas * (depth - EventTypeNoGasLevels - 1)
		if err := gasMeter.ConsumeGas(gas, "event type processing"); err != nil {
			return "", err
		}
	}

	if tag == nil {
		return "", nil
	}

	// Helper function: lcsTypeToString returns dvmTypes.LcsType Move representation
	lcsTypeToString := func(lcsType dvm.LcsType) string {
		switch lcsType {
		case dvm.LcsType_LcsBool:
			return "bool"
		case dvm.LcsType_LcsU8:
			return "u8"
		case dvm.LcsType_LcsU64:
			return "u64"
		case dvm.LcsType_LcsU128:
			return "u128"
		case dvm.LcsType_LcsSigner:
			return "signer"
		case dvm.LcsType_LcsVector:
			return "vector"
		case dvm.LcsType_LcsStruct:
			return "struct"
		default:
			return dvm.LcsType_name[int32(lcsType)]
		}
	}

	// Check data consistency
	if tag.TypeTag == dvm.LcsType_LcsVector && tag.VectorType == nil {
		return "", fmt.Errorf("TypeTag of type %q, but VectorType is nil", lcsTypeToString(tag.TypeTag))
	}
	if tag.TypeTag == dvm.LcsType_LcsStruct && tag.StructIdent == nil {
		return "", fmt.Errorf("TypeTag of type %q, but StructIdent is nil", lcsTypeToString(tag.TypeTag))
	}

	// Vector tag
	if tag.VectorType != nil {
		vectorType, err := processEventType(gasMeter, tag.VectorType, gas, depth+1)
		if err != nil {
			return "", fmt.Errorf("VectorType serialization: %w", err)
		}
		return fmt.Sprintf("%s<%s>", lcsTypeToString(dvm.LcsType_LcsVector), vectorType), nil
	}

	// Struct tag
	if tag.StructIdent != nil {
		structType := fmt.Sprintf("%s::%s::%s", StringifySenderAddress(tag.StructIdent.Address), tag.StructIdent.Module, tag.StructIdent.Name)
		if len(tag.StructIdent.TypeParams) == 0 {
			return structType, nil
		}

		structParams := make([]string, 0, len(tag.StructIdent.TypeParams))
		for paramIdx, paramTag := range tag.StructIdent.TypeParams {
			structParam, err := processEventType(gasMeter, paramTag, gas, depth+1)
			if err != nil {
				return "", fmt.Errorf("StructIdent serialization: TypeParam[%d]: %w", paramIdx, err)
			}
			structParams = append(structParams, structParam)
		}
		return fmt.Sprintf("%s<%s>", structType, strings.Join(structParams, ", ")), nil
	}

	// Single tag
	return lcsTypeToString(tag.TypeTag), nil
}
