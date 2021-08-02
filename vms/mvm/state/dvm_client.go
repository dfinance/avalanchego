package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/mvm/dvm"
	"google.golang.org/grpc"
)

// VMExecRetryReq contains VM "execution" request meta (request details and retry settings).
type VMExecRetryReq struct {
	// Request to retry (module publish).
	RawModule *dvm.VMPublishModule
	// Request to retry (script execution)
	RawScript *dvm.VMExecuteScript
	// Max number of request attempts (0 - infinite)
	MaxAttempts uint
	// Request timeout per attempt (0 - infinite) [ms]
	ReqTimeoutInMs time.Duration
}

// DVMClient is an aggregated gRPC DVM services client.
type DVMClient struct {
	sync.Mutex
	dvm.DvmCompilerClient
	dvm.DVMBytecodeMetadataClient
	dvm.VMModulePublisherClient
	dvm.VMScriptExecutorClient

	log        logging.Logger
	connection *grpc.ClientConn

	maxAttempts uint
	reqTimeout  time.Duration
}

// NewDVMClient creates VMClient using gRPC connection.
func NewDVMClient(logger logging.Logger, maxAttempts, reqTimeoutMs uint, conn *grpc.ClientConn) *DVMClient {
	return &DVMClient{
		DvmCompilerClient:         dvm.NewDvmCompilerClient(conn),
		DVMBytecodeMetadataClient: dvm.NewDVMBytecodeMetadataClient(conn),
		VMModulePublisherClient:   dvm.NewVMModulePublisherClient(conn),
		VMScriptExecutorClient:    dvm.NewVMScriptExecutorClient(conn),
		log:                       logger,
		connection:                conn,
		maxAttempts:               maxAttempts,
		reqTimeout:                time.Duration(reqTimeoutMs) * time.Millisecond,
	}
}

// Close closes DVM connection.
func (c *DVMClient) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.connection == nil {
		return nil
	}

	return c.connection.Close()
}

// SendExecuteReq sends request with retry mechanism.
func (c *DVMClient) SendExecuteReq(moduleReq *dvm.VMPublishModule, scriptReq *dvm.VMExecuteScript) (*dvm.VMExecuteResponse, error) {
	if moduleReq == nil && scriptReq == nil {
		return nil, fmt.Errorf("request (module / script) not specified")
	}
	if moduleReq != nil && scriptReq != nil {
		return nil, fmt.Errorf("only single request (module / script) is supported")
	}

	retryReq := VMExecRetryReq{
		RawModule:      moduleReq,
		RawScript:      scriptReq,
		MaxAttempts:    c.maxAttempts,
		ReqTimeoutInMs: c.reqTimeout,
	}

	return c.retryExecReq(retryReq)
}

// retryExecReq sends request with retry mechanism and waits for connection and execution.
// Contract: either RawModule or RawScript must be specified for RetryExecReq.
func (c *DVMClient) retryExecReq(req VMExecRetryReq) (retResp *dvm.VMExecuteResponse, retErr error) {
	const failedRetryLogPeriod = 100

	doneCh := make(chan bool)
	curAttempt := uint(0)
	reqTimeout := req.ReqTimeoutInMs
	reqStartedAt := time.Now()

	go func() {
		defer func() {
			close(doneCh)
		}()

		for {
			var connCtx context.Context
			var connCancel context.CancelFunc
			var resp *dvm.VMExecuteResponse
			var err error

			curAttempt++

			connCtx = context.Background()
			if reqTimeout > 0 {
				connCtx, connCancel = context.WithTimeout(context.Background(), reqTimeout)
			}

			curReqStartedAt := time.Now()
			if req.RawModule != nil {
				resp, err = c.VMModulePublisherClient.PublishModule(connCtx, req.RawModule)
			} else if req.RawScript != nil {
				resp, err = c.VMScriptExecutorClient.ExecuteScript(connCtx, req.RawScript)
			}
			if connCancel != nil {
				connCancel()
			}
			curReqDur := time.Since(curReqStartedAt)

			if err == nil {
				retResp, retErr = resp, nil
				return
			}

			if req.MaxAttempts != 0 && curAttempt == req.MaxAttempts {
				retResp, retErr = nil, err
				return
			}

			if curReqDur < reqTimeout {
				time.Sleep(reqTimeout - curReqDur)
			}

			if curAttempt%failedRetryLogPeriod == 0 {
				c.log.Info("Failing VM request: attempt %d / %d with %v timeout: %v", curAttempt, req.MaxAttempts, reqTimeout, time.Since(reqStartedAt))
			}
		}
	}()
	<-doneCh

	reqDur := time.Since(reqStartedAt)
	msg := fmt.Sprintf("in %d attempt(s) with %v timeout (%v)", curAttempt, reqTimeout, reqDur)
	if retErr == nil {
		c.log.Info(fmt.Sprintf("Successfull VM request (%s)", msg))
	} else {
		c.log.Error(fmt.Sprintf("Failed VM request (%s): %v", msg, retErr))
		retErr = fmt.Errorf("%s: %w", msg, retErr)
	}

	return
}
