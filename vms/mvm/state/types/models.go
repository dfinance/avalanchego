package types

import (
	"fmt"
	"strings"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

// CompiledItem keeps compiled Move code data.
type (
	CompiledItem struct {
		ByteCode []byte               `json:"byteCode,omitempty"`
		Name     string               `json:"name,omitempty"`
		Methods  []*dvm.Function      `json:"methods,omitempty"`
		Types    []*dvm.Struct        `json:"types,omitempty"`
		CodeType CompiledItemCodeType `json:"codeType"`
	}

	CompiledItemCodeType int32

	CompiledItems []CompiledItem
)

const (
	CompiledItemModule CompiledItemCodeType = 0
	CompiledItemScript CompiledItemCodeType = 1
)

// Metadata keeps Move code metadata.
type Metadata struct {
	Metadata *dvm.Metadata `json:"metadata,omitempty"`
}

// Event keeps DVM event.
type (
	Event struct {
		Type       string           `serialize:"true" json:"type"`
		Attributes []EventAttribute `serialize:"true" json:"attributes"`
	}

	EventAttribute struct {
		Key   string `serialize:"true" json:"key"`
		Value string `serialize:"true" json:"value"`
	}

	Events []Event
)

// NewEvent creates a new Event.
func NewEvent(eventType string, eventAttributes ...EventAttribute) Event {
	return Event{
		Type:       eventType,
		Attributes: eventAttributes,
	}
}

// NewEventAttribute creates a new EventAttribute.
func NewEventAttribute(attrKey, attrValue string) EventAttribute {
	return EventAttribute{
		Key:   attrKey,
		Value: attrValue,
	}
}

// String implements fmt.Stringer interface.
func (events Events) String() string {
	str := strings.Builder{}
	str.WriteString("\n")
	for eventIdx, event := range events {
		str.WriteString(fmt.Sprintf("[%d]: %s:\n", eventIdx, event.Type))
		for _, attr := range event.Attributes {
			str.WriteString(fmt.Sprintf("  %s -> %s\n", attr.Key, attr.Value))
		}
	}

	return str.String()
}
