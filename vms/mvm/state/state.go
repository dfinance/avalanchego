package state

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/mvm/state/types"
)

// State encapsulates DVM related operations for M-chain.
type State struct {
	baseDB      *versiondb.Database
	singletonDB database.Database

	log       logging.Logger
	wsStorage *wsStorage
	dvmClient *dvmClient
	dsServer  *dsServer
}

// NewState creates a new State instance initializing genesis data if needed.
func NewState(logger logging.Logger, config types.DVMConnectionConfig, genesisData []byte, db database.Database) (*State, bool, error) {
	logger.Info("InternalState: using DVM connection (%s) and DS address (%s)", config.DVMAddress, config.DataServerAddress)

	dsListener, err := GetGRpcNetListener(config.DataServerAddress)
	if err != nil {
		return nil, false, fmt.Errorf("creating DS server listener: %w", err)
	}

	dvmConnection, err := GetGRpcClientConnection(config.DVMAddress, 1*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("creating DVM connection: %w", err)
	}

	baseDB := versiondb.New(db)
	singletonDB := prefixdb.New(singletonDBPrefix, baseDB)
	wsStorage := newWSStorage(logger, prefixdb.New(writeSetsDBPrefix, baseDB))
	dvmClient := newDVMClient(logger, config.MaxAttempts, config.ReqTimeoutInMs, dvmConnection)
	dsSever := newDSServer(logger, wsStorage, dsListener)

	s := &State{
		baseDB:      baseDB,
		singletonDB: singletonDB,
		log:         logger,
		wsStorage:   wsStorage,
		dvmClient:   dvmClient,
		dsServer:    dsSever,
	}

	genesisBlockInitialized, err := s.sync(genesisData)
	if err != nil {
		return nil, false, fmt.Errorf("genesisState sync: %w", err)
	}

	s.dsServer.Start()

	return s, genesisBlockInitialized, nil
}

// Close stops all inner services.
func (s *State) Close() error {
	s.dsServer.Stop()

	errs := wrappers.Errs{}
	errs.Add(
		s.dvmClient.Close(),
		s.wsStorage.Close(),
		s.singletonDB.Close(),
		s.baseDB.Close(),
	)

	return errs.Err
}

// isInitialized checks if genesis has been synced.
func (s *State) isInitialized() (bool, error) {
	found, err := s.singletonDB.Has(initializedKey)
	if err != nil {
		return false, err
	}

	return !found, nil
}

// setInitialized sets genesis initialized flag.
func (s *State) setInitialized() error {
	return s.singletonDB.Put(initializedKey, nil)
}

// sync initializes genesis data.
func (s *State) sync(genesisData []byte) (bool, error) {
	shouldInit, err := s.isInitialized()
	if err != nil {
		return false, fmt.Errorf("checking if DB is initialized: %w", err)
	}
	if !shouldInit {
		return false, nil
	}

	genesisState, err := types.NewGenesisState(genesisData)
	if err != nil {
		return false, fmt.Errorf("building genesisState: %w", err)
	}

	for idx, writeSet := range genesisState.WriteSets {
		path, data, err := writeSet.ToBytes()
		if err != nil {
			return false, fmt.Errorf("converting writeSet [%d]: %w", idx, err)
		}

		if err := s.wsStorage.PutWriteSet(path, data); err != nil {
			return false, err
		}
	}

	if err := s.setInitialized(); err != nil {
		return false, fmt.Errorf("setting DB initialized flag: %w", err)
	}

	return true, nil
}
