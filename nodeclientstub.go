package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
)

// StakeInfoStub contains static data used to generate outputs of the
// StakeInfoer methods, and it implements those methods.
type StakeInfoStub struct {
	status                  error
	bestBlockHash           *chainhash.Hash
	bestBlockHeight         int64
	bestBlockStakeVersion   uint32
	blockVerboseResults     map[chainhash.Hash]*dcrjson.GetBlockVerboseResult
	stakeVersionsResults    *dcrjson.GetStakeVersionsResult
	stakeVersionInfoResults *dcrjson.GetStakeVersionInfoResult
	voteInfoResults         map[uint32]*dcrjson.GetVoteInfoResult
}

// NewStakeInfoStub is the constructor for StakeInfoStub. It reads JSON data
// from strings or disk and unmarshals the data into the required dcrjson types
// returned by the StakeInfoer methods.
func NewStakeInfoStub(bestBlockOpt ...uint32) *StakeInfoStub {
	bestBlock := uint32(289984)
	if len(bestBlockOpt) > 0 {
		bestBlock = bestBlockOpt[0]
	}

	stubPath := filepath.Join("stub_data", fmt.Sprintf("bestBlock%d", bestBlock))
	_, err := os.Stat(stubPath)
	if err != nil {
		fmt.Printf("Unable to find stub path %s: %v", stubPath, err)
	}

	voteInfo4, err := ioutil.ReadFile(filepath.Join(stubPath, "getVoteInfo4.json"))
	if err != nil {
		panic(fmt.Sprint("Unable to read voteInfo4Fake.json: ", err))
	}
	voteInfo4Result := new(dcrjson.GetVoteInfoResult)
	if err = json.Unmarshal(voteInfo4, voteInfo4Result); err != nil {
		panic(fmt.Sprint("Unable to unmarshal voteInfo4Fake.json: ", err))
	}

	voteInfoResults := map[uint32]*dcrjson.GetVoteInfoResult{
		voteInfo4Result.VoteVersion: voteInfo4Result,
	}

	bbHash, err := chainhash.NewHashFromStr(voteInfo4Result.Hash)
	if err != nil {
		panic("invalid block hash in getvoteinfo results")
	}

	// getstakeversions
	stakeVersions, err := ioutil.ReadFile(filepath.Join(stubPath, "getStakeVersions2016.json"))
	if err != nil {
		panic(fmt.Sprint("Unable to read getStakeVersions2016.json: ", err))
	}
	stakeVersionsResult := new(dcrjson.GetStakeVersionsResult)
	if err = json.Unmarshal(stakeVersions, stakeVersionsResult); err != nil {
		panic(fmt.Sprint("Unable to unmarshal getStakeVersions2016.json: ", err))
	}

	// getstakeversioninfo
	stakeVersionInfo, err := ioutil.ReadFile(filepath.Join(stubPath, "getStakeVersionInfo144.json"))
	if err != nil {
		panic(fmt.Sprint("Unable to read getStakeVersionInfo144.json: ", err))
	}
	stakeVersionInfoResult := new(dcrjson.GetStakeVersionInfoResult)
	if err = json.Unmarshal(stakeVersionInfo, stakeVersionInfoResult); err != nil {
		panic(fmt.Sprint("Unable to unmarshal getStakeVersionInfo144.json: ", err))
	}

	// TODO: get the rest of the data

	stakeInfoStub := &StakeInfoStub{
		status:                  nil,
		bestBlockHash:           bbHash,
		bestBlockHeight:         int64(bestBlock),
		bestBlockStakeVersion:   stakeVersionsResult.StakeVersions[0].StakeVersion,
		blockVerboseResults:     make(map[chainhash.Hash]*dcrjson.GetBlockVerboseResult),
		stakeVersionsResults:    stakeVersionsResult,
		stakeVersionInfoResults: stakeVersionInfoResult,
		voteInfoResults:         voteInfoResults,
	}

	return stakeInfoStub
}

// GetBestBlock fakes dcrrpcclient's (*Client).GetBestBlock
func (s *StakeInfoStub) GetBestBlock() (*chainhash.Hash, int64, error) {
	return s.bestBlockHash, s.bestBlockHeight, nil
}

// GetBlockVerbose fakes dcrrpcclient's (*Client).GetBlockVerbose, but panics if
// the input hash is not the best block hash
func (s *StakeInfoStub) GetBlockVerbose(blockHash *chainhash.Hash, verboseTx bool) (*dcrjson.GetBlockVerboseResult, error) {
	if !blockHash.IsEqual(s.bestBlockHash) {
		return nil, fmt.Errorf("input block hash is not the best block hash of stub")
	}
	return &dcrjson.GetBlockVerboseResult{
		Height:       s.bestBlockHeight,
		Hash:         s.bestBlockHash.String(),
		StakeVersion: s.bestBlockStakeVersion,
	}, nil
}

// GetStakeVersions fakes dcrrpcclient's (*Client).GetStakeVersions
func (s *StakeInfoStub) GetStakeVersions(hash string, count int32) (*dcrjson.GetStakeVersionsResult, error) {
	if hash != s.bestBlockHash.String() {
		return nil, fmt.Errorf("input block hash is not the best block hash of stub")
	}
	if int(count) > len(s.stakeVersionsResults.StakeVersions) {
		return nil, fmt.Errorf("insufficient number of getstakeversions results")
	}

	stakeVersionsResult := new(dcrjson.GetStakeVersionsResult)
	stakeVersionsResult.StakeVersions = s.stakeVersionsResults.StakeVersions[:count]

	return stakeVersionsResult, nil
}

// GetStakeVersionInfo fakes dcrrpcclient's (*Client).GetStakeVersionInfo
func (s *StakeInfoStub) GetStakeVersionInfo(count int32) (*dcrjson.GetStakeVersionInfoResult, error) {
	if int(count) > len(s.stakeVersionInfoResults.Intervals) {
		return nil, fmt.Errorf("insufficient number of getstakeversioninfo intervals")
	}

	stakeVersionInfoResult := &dcrjson.GetStakeVersionInfoResult{
		CurrentHeight: s.bestBlockHeight,
		Hash:          s.bestBlockHash.String(),
		Intervals:     s.stakeVersionInfoResults.Intervals[:count],
	}

	return stakeVersionInfoResult, nil
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
