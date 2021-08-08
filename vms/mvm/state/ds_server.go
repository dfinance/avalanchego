package state

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/mvm/dvm"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ dvm.DSServiceServer = (*dsServer)(nil)

// dsServer implements dvm.DSServiceServer interface.
// DataSource gRPC server is used by the DVM instance to retrieve chain data.
type dsServer struct {
	sync.Mutex

	log       logging.Logger
	listener  net.Listener
	wsStorage *wsStorage

	server *grpc.Server
}

// newDSServer creates a new dsServer instance.
func newDSServer(logger logging.Logger, wsStorage *wsStorage, listener net.Listener) *dsServer {
	return &dsServer{
		log:       logger,
		listener:  listener,
		wsStorage: wsStorage,
	}
}

// Start starts DS gRPC server.
func (srv *dsServer) Start() {
	srv.Lock()
	defer srv.Unlock()

	if srv.server != nil {
		return
	}

	srv.server = grpc.NewServer()
	dvm.RegisterDSServiceServer(srv.server, srv)

	go func() {
		if err := srv.server.Serve(srv.listener); err != nil {
			panic(err) // should not happen
		}
	}()
	srv.log.Info("DS server: started")
}

// Stop stops DS gRPC server.
func (srv *dsServer) Stop() {
	srv.Lock()
	defer srv.Unlock()

	if srv.server == nil {
		return
	}

	srv.server.Stop()
}

// GetRaw handles gRPC service request: reads writeSet data.
func (srv *dsServer) GetRaw(_ context.Context, req *dvm.DSAccessPath) (*dvm.DSRawResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	path := &dvm.VMAccessPath{
		Address: req.Address,
		Path:    req.Path,
	}

	wsFound, err := srv.wsStorage.HasWriteSet(path)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !wsFound {
		return &dvm.DSRawResponse{
			ErrorCode:    dvm.ErrorCode_NO_DATA,
			ErrorMessage: fmt.Sprintf("data not found for access path: %s", path.String()),
		}, nil
	}

	wsData, err := srv.wsStorage.GetWriteSet(path)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &dvm.DSRawResponse{Blob: wsData}, nil
}

func (srv *dsServer) GetOraclePrice(_ context.Context, req *dvm.OraclePriceRequest) (*dvm.OraclePriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetOraclePrice unimplemented")
}

func (srv *dsServer) GetNativeBalance(_ context.Context, req *dvm.NativeBalanceRequest) (*dvm.NativeBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetNativeBalance unimplemented")
}

func (srv *dsServer) GetCurrencyInfo(_ context.Context, req *dvm.CurrencyInfoRequest) (*dvm.CurrencyInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetCurrencyInfo unimplemented")
}

func (srv *dsServer) MultiGetRaw(_ context.Context, req *dvm.DSAccessPaths) (*dvm.DSRawResponses, error) {
	return nil, status.Errorf(codes.Unimplemented, "MultiGetRaw unimplemented")
}
