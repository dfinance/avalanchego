package state

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/mvm/types"
)

type State struct {
	baseDB      *versiondb.Database
	writeSetsDB database.Database
	singletonDB database.Database

	log       logging.Logger
	dsServer  *DSServer
	dvmClient *DVMClient
}

func NewState(logger logging.Logger, config types.DVMConnection, genesisData []byte, db database.Database) (*State, bool, error) {
	dsListener, err := GetGRpcNetListener(config.DataServerAddress)
	if err != nil {
		return nil, false, fmt.Errorf("creating DS server listener: %w", err)
	}

	dvmConnection, err := GetGRpcClientConnection(config.DVMAddress, 1*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("creating DVM connection: %w", err)
	}

	baseDB := versiondb.New(db)
	s := &State{
		baseDB:      baseDB,
		writeSetsDB: prefixdb.New(writeSetsDBPrefix, baseDB),
		singletonDB: prefixdb.New(singletonDBPrefix, baseDB),
		log:         logger,
		dvmClient:   NewDVMClient(logger, config.MaxAttempts, config.ReqTimeoutInMs, dvmConnection),
	}
	s.dsServer = NewDSServer(logger, s, dsListener)

	genesisBlockInitialized, err := s.sync(genesisData)
	if err != nil {
		return nil, false, fmt.Errorf("genesisState sync: %w", err)
	}

	s.dsServer.Start()

	return s, genesisBlockInitialized, nil
}

func (s *State) Close() error {
	s.dsServer.Stop()

	errs := wrappers.Errs{}
	errs.Add(
		s.dvmClient.Close(),
		s.writeSetsDB.Close(),
		s.singletonDB.Close(),
		s.baseDB.Close(),
	)

	return errs.Err
}

func (s *State) isInitialized() (bool, error) {
	found, err := s.singletonDB.Has(initializedKey)
	if err != nil {
		return false, err
	}

	return !found, nil
}

func (s *State) setInitialized() error {
	return s.singletonDB.Put(initializedKey, nil)
}

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
	if err := genesisState.Validate(); err != nil {
		return false, fmt.Errorf("validating genesisState: %w", err)
	}

	for idx, writeSet := range genesisState.WriteSets {
		path, data, err := writeSet.ToBytes()
		if err != nil {
			return false, fmt.Errorf("converting writeSet [%d]: %w", idx, err)
		}

		if err := s.PutWriteSet(path, data); err != nil {
			return false, err
		}
	}

	if err := s.setInitialized(); err != nil {
		return false, fmt.Errorf("setting DB initialized flag: %w", err)
	}

	return true, nil
}
