package types

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
)

const (
	EventTypeContractStatus = "contract_status"
	EventTypeMoveEvent      = "contract_events"

	AttributeStatus             = "status"
	AttributeErrMajorStatus     = "major_status"
	AttributeErrSubStatus       = "sub_status"
	AttributeErrMessage         = "message"
	AttributeErrLocationAddress = "location_address"
	AttributeErrLocationModule  = "location_module"
	AttributeVmEventSender      = "sender_address"
	AttributeVmEventSource      = "source"
	AttributeVmEventType        = "type"
	AttributeVmEventData        = "data"

	AttributeValueStatusKeep      = "keep"
	AttributeValueStatusDiscard   = "discard"
	AttributeValueStatusError     = "error"
	AttributeValueSourceScript    = "script"
	AttributeValueSourceModuleFmt = "%s::%s"
)

// NewContractEvents creates Events on successful / failed VM execution.
// "keep" status emits two events, "discard" status emits one event.
// panic if dvmTypes.VMExecuteResponse or dvmTypes.VMExecuteResponse.Status == nil
func NewContractEvents(exec *dvm.VMExecuteResponse) (Events, error) {
	if exec == nil {
		return nil, fmt.Errorf("building contract Events: exec is nil")
	}

	status := exec.GetStatus()
	if status == nil {
		return nil, fmt.Errorf("building contract Events: exec.Status is nil")
	}

	if status.GetError() == nil {
		return Events{
			NewEvent(
				EventTypeContractStatus,
				NewEventAttribute(AttributeStatus, AttributeValueStatusKeep),
			),
		}, nil
	}

	// Allocate memory for 5 possible attributes: status, abort location 2 attributes, major and sub codes
	attributes := make([]EventAttribute, 1, 5)
	attributes[0] = NewEventAttribute(AttributeStatus, AttributeValueStatusDiscard)

	if sErr := status.GetError(); sErr != nil {
		majorStatus, subStatus, abortLocation, err := GetStatusCodesFromVMStatus(status)
		if err != nil {
			return nil, err
		}

		if abortLocation != nil {
			if abortLocation.GetAddress() != nil {
				address := abortLocation.GetAddress()
				attributes = append(attributes, NewEventAttribute(AttributeErrLocationAddress, string(address)))
			}

			if abortLocation.GetModule() != "" {
				attributes = append(attributes, NewEventAttribute(AttributeErrLocationModule, abortLocation.GetModule()))
			}
		}

		attributes = append(
			attributes,
			NewEventAttribute(AttributeErrMajorStatus, strconv.FormatUint(majorStatus, 10)),
			NewEventAttribute(AttributeErrSubStatus, strconv.FormatUint(subStatus, 10)),
		)

		if status.GetMessage() != nil {
			attributes = append(attributes, NewEventAttribute(AttributeErrMessage, status.GetMessage().GetText()))
		}
	}

	return Events{NewEvent(EventTypeContractStatus, attributes...)}, nil
}

// NewMoveEvent converts VM event to SDK event.
// GasMeter is used to prevent long parsing (lots of nested structs).
func NewMoveEvent(gasMeter *GasMeter, vmEvent *dvm.VMEvent) (Event, error) {
	if vmEvent == nil {
		return Event{}, fmt.Errorf("building Move sdk.Event: event is nil")
	}

	eventType, err := StringifyEventType(gasMeter, vmEvent.EventType)
	if err != nil {
		return Event{}, err
	}

	// eventData: not parsed as it doesn't make sense
	return NewEvent(EventTypeMoveEvent,
		NewEventAttribute(AttributeVmEventSender, StringifySenderAddress(vmEvent.SenderAddress)),
		NewEventAttribute(AttributeVmEventSource, GetEventSourceAttribute(vmEvent.SenderModule)),
		NewEventAttribute(AttributeVmEventType, eventType),
		NewEventAttribute(AttributeVmEventData, hex.EncodeToString(vmEvent.EventData)),
	), nil
}

// GetStatusCodesFromVMStatus extracts majorStatus, subStatus and abortLocation from dvmTypes.VMStatus
// panic if error exist but error object == nil
func GetStatusCodesFromVMStatus(status *dvm.VMStatus) (majorStatus, subStatus uint64, location *dvm.AbortLocation, retErr error) {
	switch sErr := status.GetError().(type) {
	case *dvm.VMStatus_Abort:
		majorStatus = VMAbortedCode
		if sErr.Abort == nil {
			retErr = fmt.Errorf("getting status codes: VMStatus_Abort.Abort is nil")
			return
		}
		subStatus = sErr.Abort.GetAbortCode()
		if l := sErr.Abort.GetAbortLocation(); l != nil {
			location = l
		}
	case *dvm.VMStatus_ExecutionFailure:
		if sErr.ExecutionFailure == nil {
			retErr = fmt.Errorf("getting status codes: VMStatus_ExecutionFailure.ExecutionFailure is nil")
			return
		}
		majorStatus = sErr.ExecutionFailure.GetStatusCode()
		if l := sErr.ExecutionFailure.GetAbortLocation(); l != nil {
			location = l
		}
	case *dvm.VMStatus_MoveError:
		if sErr.MoveError == nil {
			panic(fmt.Errorf("getting status codes: VMStatus_MoveError.MoveError is nil"))
		}
		majorStatus = sErr.MoveError.GetStatusCode()
	case nil:
		majorStatus = VMExecutedCode
	}

	return
}

// GetEventSourceAttribute returns SDK event attribute for VM event source (script / module) serialized to string.
func GetEventSourceAttribute(senderModule *dvm.ModuleIdent) string {
	if senderModule == nil {
		return AttributeValueSourceScript
	}

	return fmt.Sprintf(AttributeValueSourceModuleFmt, StringifySenderAddress(senderModule.Address), senderModule.Name)
}
