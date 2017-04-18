package main

import (
	"fmt"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrrpcclient"
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

// MakeClient creates a StakeInfoer and a cleanup function for the caller to
// defer. The StakeInfoer will either be a *dcrrpcclient.Client or a
// *StakeInfoStub, depending on the stub input argument.
func MakeClient(config *dcrrpcclient.ConnConfig,
	ntfnHandlers *dcrrpcclient.NotificationHandlers, stub bool,
	stubHeightOpt ...uint32) (StakeInfoer, func(), error) {
	if stub {
		return NewStakeInfoStub(stubHeightOpt...), func() {}, nil
	}

	fmt.Printf("Attempting to connect to dcrd RPC %s as user %s\n",
		config.Host, config.User)
	// Attempt to connect rpcclient and daemon
	dcrdClient, err := dcrrpcclient.New(config, ntfnHandlers)
	if err != nil {
		fmt.Printf("Failed to start dcrd rpcclient: %s\n", err.Error())
		return dcrdClient, func() {}, err
	}
	cleanup := func() {
		fmt.Printf("Disconnecting from dcrd.\n")
		dcrdClient.Disconnect()
	}

	// Subscribe to block notifications
	if err = dcrdClient.NotifyBlocks(); err != nil {
		fmt.Printf("Failed to start register daemon rpc client for  "+
			"block notifications: %s\n", err.Error())
	}

	return dcrdClient, cleanup, err
}
