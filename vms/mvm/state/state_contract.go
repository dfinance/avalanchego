package state

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"github.com/ava-labs/avalanchego/vms/mvm/state/types"
)

// ExecuteContract executes Move script and processes execution results (events, writeSets).
func (s *State) ExecuteContract(msg *types.MsgExecuteScript, blockHeight uint64) (types.Events, error) {
	vmArgs := make([]*dvm.VMArgs, 0, len(msg.Args))
	for _, arg := range msg.Args {
		vmArgs = append(vmArgs, &dvm.VMArgs{
			Type:  arg.Type,
			Value: arg.Value,
		})
	}

	req := &dvm.VMExecuteScript{
		Senders:      [][]byte{msg.Sender},
		MaxGasAmount: types.DVMGasLimit,
		GasUnitPrice: types.DVMGasPrice,
		Block:        blockHeight,
		Timestamp:    0,
		Code:         msg.Script,
		TypeParams:   nil,
		Args:         vmArgs,
	}

	exec, err := s.dvmClient.SendExecuteReq(nil, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC error: %w", err)
	}

	return s.processDVMExecution(exec)
}

// DeployContract deploys Move module (contract) and processes execution results (events, writeSets).
func (s *State) DeployContract(msg *types.MsgDeployModule) (types.Events, error) {
	execList := make([]*dvm.VMExecuteResponse, 0, len(msg.Modules))
	for i, code := range msg.Modules {
		req := &dvm.VMPublishModule{
			Sender:       msg.Sender,
			MaxGasAmount: types.DVMGasLimit,
			GasUnitPrice: types.DVMGasPrice,
			Code:         code,
		}

		exec, err := s.dvmClient.SendExecuteReq(req, nil)
		if err != nil {
			return nil, fmt.Errorf("contract [%d]: gRPC error: %w", i, err)
		}
		execList = append(execList, exec)
	}

	var retEvents types.Events
	var errList []string
	for i, exec := range execList {
		events, err := s.processDVMExecution(exec)
		if err != nil {
			errList = append(errList, fmt.Sprintf("execution [%d]: %v", i, err))
		}
		retEvents = append(retEvents, events...)
	}

	var retErr error
	if len(errList) > 0 {
		retErr = fmt.Errorf("%s", strings.Join(errList, ", "))
	}

	return retEvents, retErr
}

// GetMetadata returns contract metadata by its byte code.
func (s *State) GetMetadata(msg *types.MsgGetMetadata) (*types.Metadata, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	res, err := s.dvmClient.GetMetadata(ctx, &dvm.Bytecode{
		Code: msg.Code,
	})
	if err != nil {
		return nil, fmt.Errorf("getting meta information: %w", err)
	}

	return &types.Metadata{Metadata: res}, nil
}

// Compile compiles Move code.
func (s *State) Compile(msg *types.MsgCompile) (types.CompiledItems, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	// Compile request
	resp, err := s.dvmClient.Compile(ctx, &dvm.SourceFiles{
		Units: []*dvm.CompilationUnit{
			{
				Text: string(msg.Code),
				Name: "CompilationUnit",
			},
		},
		Address: msg.Sender,
	})
	if err != nil {
		return nil, fmt.Errorf("DVM connection: %w", err)
	}

	// Check for compilation errors
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("compiler errors: [%s]", strings.Join(resp.Errors, ", "))
	}

	// Build response
	compItems := make(types.CompiledItems, 0, len(resp.Units))
	for _, unit := range resp.Units {
		compItem := types.CompiledItem{
			ByteCode: unit.Bytecode,
			Name:     unit.Name,
		}

		meta, err := s.dvmClient.GetMetadata(ctx, &dvm.Bytecode{Code: unit.Bytecode})
		if err != nil {
			return nil, fmt.Errorf("getting meta information: %w", err)
		}

		if ok := meta.GetScript(); ok != nil {
			compItem.CodeType = types.CompiledItemScript
		}

		if moduleMeta := meta.GetModule(); moduleMeta != nil {
			compItem.CodeType = types.CompiledItemModule
			compItem.Types = moduleMeta.GetTypes()
			compItem.Methods = moduleMeta.GetFunctions()
		}

		compItems = append(compItems, compItem)
	}

	return compItems, nil
}

// processDVMExecution processes DVM execution result: updates writeSets.
func (s *State) processDVMExecution(exec *dvm.VMExecuteResponse) (types.Events, error) {
	// Build events with infinite (almost) gasMeter
	events, err := types.NewContractEvents(exec)
	if err != nil {
		return nil, err
	}

	gasMeter := types.NewGasMeter(math.MaxUint64)
	for _, vmEvent := range exec.Events {
		event, err := types.NewMoveEvent(gasMeter, vmEvent)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	// Process success status
	if exec.GetStatus().GetError() == nil {
		if err := s.processDVMWriteSet(exec.WriteSet); err != nil {
			return events, fmt.Errorf("processing writeSets: %w", err)
		}

		return events, nil
	}

	return events, fmt.Errorf("execution failed (refer to events for details)")
}

// processDVMWriteSet processes VM execution writeSets (set/delete).
func (s *State) processDVMWriteSet(writeSet []*dvm.VMValue) error {
	for i, value := range writeSet {
		if value == nil {
			return fmt.Errorf("writeSet [%d]: nil value received", i)
		}

		switch value.Type {
		case dvm.VmWriteOp_Value:
			if err := s.wsStorage.PutWriteSet(value.Path, value.Value); err != nil {
				return fmt.Errorf("writeSet [%d]: WriteOp: %w", i, err)
			}
		case dvm.VmWriteOp_Deletion:
			if err := s.wsStorage.DeleteWriteSet(value.Path); err != nil {
				return fmt.Errorf("writeSet [%d]: DeleteOp: %w", i, err)
			}
		default:
			panic(fmt.Errorf("processing writeSets: unsupported writeOp.Type: %d", value.Type))
		}
	}

	return nil
}
