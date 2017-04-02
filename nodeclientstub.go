package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
)

// StakeInfoStub contains static data used to generate outputs of the
// StakeInfoer methods, and it implements those methods.
type StakeInfoStub struct {
	status                  error
	bestBlockHash           *chainhash.Hash
	bestBlockHeigt          int64
	bestBlockStakeVersion   uint32
	blockVerboseResults     map[chainhash.Hash]*dcrjson.GetBlockVerboseResult
	stakeVersionsResults    []*dcrjson.GetStakeVersionsResult
	stakeVersionInfoResults []*dcrjson.GetStakeVersionInfoResult
	voteInfoResults         map[uint32]*dcrjson.GetVoteInfoResult
}

// NewStakeInfoStub is the constructor for StakeInfoStub. It reads JSON data
// from strings or disk and unmarshals the data into the required dcrjson types
// returned by the StakeInfoer methods.
func NewStakeInfoStub() *StakeInfoStub {
	voteInfoFake, err := ioutil.ReadFile("voteInfo4Fake.json")
	if err != nil {
		panic(fmt.Sprint("Unable to read voteInfo4Fake.json: ", err))
	}
	voteInfoFakeResult := new(dcrjson.GetVoteInfoResult)
	if err = json.Unmarshal(voteInfoFake, voteInfoFakeResult); err != nil {
		panic(fmt.Sprint("Unable to unmarshal voteInfo4Fake.json: ", err))
	}

	voteInfoResults := map[uint32]*dcrjson.GetVoteInfoResult{
		voteInfoFakeResult.VoteVersion: voteInfoFakeResult,
	}

	bbHash, err := chainhash.NewHashFromStr(voteInfoFakeResult.Hash)
	if err != nil {
		panic("invalid block hash in getvoteinfo results")
	}

	// TODO: get the rest of the data

	stakeInfoStub := &StakeInfoStub{
		status:                  nil,
		bestBlockHash:           bbHash,
		bestBlockHeigt:          289984,
		bestBlockStakeVersion:   4,
		blockVerboseResults:     make(map[chainhash.Hash]*dcrjson.GetBlockVerboseResult),
		stakeVersionsResults:    nil,
		stakeVersionInfoResults: nil,
		voteInfoResults:         voteInfoResults,
	}

	return stakeInfoStub
}

// GetBestBlock fakes dcrrpcclient's (*Client).GetBestBlock
func (s *StakeInfoStub) GetBestBlock() (*chainhash.Hash, int64, error) {
	return s.bestBlockHash, s.bestBlockHeigt, nil
}

// GetBlockVerbose fakes dcrrpcclient's (*Client).GetBlockVerbose, but panics if
// the input hash is not the best block hash
func (s *StakeInfoStub) GetBlockVerbose(blockHash *chainhash.Hash, verboseTx bool) (*dcrjson.GetBlockVerboseResult, error) {
	if !blockHash.IsEqual(s.bestBlockHash) {
		return nil, fmt.Errorf("input block hash is not the best block hash of stub")
	}
	return &dcrjson.GetBlockVerboseResult{
		Height:       s.bestBlockHeigt,
		Hash:         s.bestBlockHash.String(),
		StakeVersion: s.bestBlockStakeVersion,
	}, nil
}

// GetStakeVersions fakes dcrrpcclient's (*Client).GetStakeVersions
func (s *StakeInfoStub) GetStakeVersions(hash string, count int32) (*dcrjson.GetStakeVersionsResult, error) {
	return nil, nil
}

// GetStakeVersionInfo fakes dcrrpcclient's (*Client).GetStakeVersionInfo
func (s *StakeInfoStub) GetStakeVersionInfo(count int32) (*dcrjson.GetStakeVersionInfoResult, error) {
	return nil, nil
}

// GetVoteInfo fakes dcrrpcclient's (*Client).GetVoteInfo
func (s *StakeInfoStub) GetVoteInfo(version uint32) (*dcrjson.GetVoteInfoResult, error) {
	res, ok := s.voteInfoResults[version]
	var err error
	if !ok {
		err = fmt.Errorf("Vote version %d not found", version)
	}
	return res, err
}
