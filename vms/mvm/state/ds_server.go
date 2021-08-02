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

type DSServer struct {
	sync.Mutex

	log      logging.Logger
	listener net.Listener
	state    *State

	server *grpc.Server
}

func NewDSServer(logger logging.Logger, state *State, listener net.Listener) *DSServer {
	return &DSServer{
		log:      logger,
		listener: listener,
		state:    state,
	}
}

func (srv *DSServer) Start() {
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
	srv.log.Info("DS server started")
}

func (srv *DSServer) Stop() {
	srv.Lock()
	defer srv.Unlock()

	if srv.server == nil {
		return
	}

	srv.server.Stop()
	srv.log.Info("DS server stopped")
}

func (srv *DSServer) GetRaw(_ context.Context, req *dvm.DSAccessPath) (*dvm.DSRawResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	path := &dvm.VMAccessPath{
		Address: req.Address,
		Path:    req.Path,
	}

	wsFound, err := srv.state.HasWriteSet(path)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !wsFound {
		return &dvm.DSRawResponse{
			ErrorCode:    dvm.ErrorCode_NO_DATA,
			ErrorMessage: fmt.Sprintf("data not found for access path: %s", path.String()),
		}, nil
	}

	wsData, err := srv.state.GetWriteSet(path)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &dvm.DSRawResponse{Blob: wsData}, nil
}

func (srv *DSServer) GetOraclePrice(_ context.Context, req *dvm.OraclePriceRequest) (*dvm.OraclePriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetOraclePrice unimplemented")
}

func (srv *DSServer) GetNativeBalance(_ context.Context, req *dvm.NativeBalanceRequest) (*dvm.NativeBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetNativeBalance unimplemented")
}

func (srv *DSServer) GetCurrencyInfo(_ context.Context, req *dvm.CurrencyInfoRequest) (*dvm.CurrencyInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetCurrencyInfo unimplemented")
}

func (srv *DSServer) MultiGetRaw(_ context.Context, req *dvm.DSAccessPaths) (*dvm.DSRawResponses, error) {
	return nil, status.Errorf(codes.Unimplemented, "MultiGetRaw unimplemented")
}
