package state

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

// ExecuteContract executes Move script and processes execution results (events, writeSets).
func (s *State) ExecuteContract(msg types.TxExecuteScript, blockHeight uint64) (types.Events, error) {
	s.log.Info("Execute contract")

	req := types.NewVMExecuteScriptRequest(msg.Signer, msg.Script, blockHeight, msg.Args...)

	exec, err := s.dvmClient.SendExecuteReq(nil, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC error: %w", err)
	}

	return s.processDVMExecution(exec)
}

// DeployContract deploys Move module (contract) and processes execution results (events, writeSets).
func (s *State) DeployContract(msg types.TxDeployModule) (types.Events, error) {
	s.log.Info("Deploy contract")

	execList := make([]*dvm.VMExecuteResponse, 0, len(msg.Modules))
	for i, code := range msg.Modules {
		req := types.NewVMPublishModuleRequests(msg.Signer, code)

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

func (s *State) GetMetadata(code []byte) (*types.Metadata, error) {
	if len(code) == 0 {
		return nil, fmt.Errorf("code: empty: %w", types.ErrInvalidInput)
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	res, err := s.dvmClient.GetMetadata(ctx, &dvm.Bytecode{
		Code: code,
	})
	if err != nil {
		return nil, fmt.Errorf("getting meta information: %w", err)
	}

	return &types.Metadata{Metadata: res}, nil
}

func (s *State) Compile(senderAddress, code []byte) ([]types.CompiledItem, error) {
	if len(senderAddress) != types.DVMAddressLength {
		return nil, fmt.Errorf("address: invalid length (should be %d): %w", types.DVMAddressLength, types.ErrInvalidInput)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("code: empty: %w", types.ErrInvalidInput)
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	// Compile request
	resp, err := s.dvmClient.Compile(ctx, &dvm.SourceFiles{
		Units: []*dvm.CompilationUnit{
			{
				Text: string(code),
				Name: "CompilationUnit",
			},
		},
		Address: senderAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("DVM connection: %w", err)
	}

	// Check for compilation errors
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("compiler errors: [%s]", strings.Join(resp.Errors, ", "))
	}

	// Build response
	compItems := make([]types.CompiledItem, 0, len(resp.Units))
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

// processDVMExecution processes DVM execution result: eupdates writeSets.
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
			if err := s.PutWriteSet(value.Path, value.Value); err != nil {
				return fmt.Errorf("writeSet [%d]: WriteOp: %w", i, err)
			}
		case dvm.VmWriteOp_Deletion:
			if err := s.DeleteWriteSet(value.Path); err != nil {
				return fmt.Errorf("writeSet [%d]: DeleteOp: %w", i, err)
			}
		default:
			panic(fmt.Errorf("processing writeSets: unsupported writeOp.Type: %d", value.Type))
		}
	}

	return nil
}
