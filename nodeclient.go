package main

import (
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
)

// StakeInfoer is the inferface a type must satisfy for hardforkdemo's
// updatetemplateInformation()
type StakeInfoer interface {
	GetBestBlock() (*chainhash.Hash, int64, error)
	GetBlockVerbose(blockHash *chainhash.Hash, verboseTx bool) (*dcrjson.GetBlockVerboseResult, error)
	GetStakeVersions(hash string, count int32) (*dcrjson.GetStakeVersionsResult, error)
	GetStakeVersionInfo(count int32) (*dcrjson.GetStakeVersionInfoResult, error)
	GetVoteInfo(version uint32) (*dcrjson.GetVoteInfoResult, error)
}
