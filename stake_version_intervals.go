// Copyright (c) 2017-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/rpc/jsonrpc/types/v3"
	"github.com/decred/dcrd/rpcclient/v7"
)

// StakeVersionIntervals wraps a set of types.VersionIntervals
type StakeVersionIntervals struct {
	Intervals      []types.VersionInterval
	MaxVoteVersion uint32
}

// GetStakeVersionUpgradeSVI will search through every stake version interval
// to find the first SVI which meets the upgrade threshold for the provided version.
func (s *StakeVersionIntervals) GetStakeVersionUpgradeSVI(version uint32) (upgradeOccurred bool, upgradeSVI types.VersionInterval) {

	// This site is not able to cope with the PoS upgrade threshold being met
	// before the PoW threshold.
	// That happened on both testnet and mainnet during the version 8 upgrade,
	// and on mainnet during version 9.
	// Hardcoding those upgrade SVIs rather than detecting them
	// programmatically.
	if version == 8 {
		switch activeNetParams.Name {
		case chaincfg.MainNetParams().Name:
			return true, s.Intervals[261]
		case chaincfg.TestNet3Params().Name:
			return true, s.Intervals[152]
		default:
			panic("unsupported network")
		}
	}
	if version == 9 && activeNetParams.Name == chaincfg.MainNetParams().Name {
		return true, s.Intervals[312]
	}

	for i, svi := range s.Intervals {
		// If this is an incomplete SVI, then the upgrade has not happened.
		if svi.EndHeight-svi.StartHeight < activeNetParams.StakeVersionInterval {
			continue
		}

		// Count the votes in this SVI to see if the upgrade threshold has been met
		var totalVotes int32
		var versionVotes int32
		for _, voteVersion := range svi.VoteVersions {
			totalVotes += int32(voteVersion.Count)
			if voteVersion.Version == version {
				versionVotes += int32(voteVersion.Count)
			}
		}
		upgradeThreshold := totalVotes * activeNetParams.StakeMajorityMultiplier / activeNetParams.StakeMajorityDivisor
		if versionVotes > upgradeThreshold {
			log.Printf("v%d upgrade threshold was met during SVI %d (blocks %d-%d). Total votes: %d, v%d votes: %d, threshold: %d",
				version, i+1, svi.StartHeight, svi.EndHeight, totalVotes, version, versionVotes, upgradeThreshold)
			return true, svi
		}
	}
	return false, types.VersionInterval{}
}

// AllStakeVersionIntervals uses the dcrd client to create an ordered
// set of objects representing every Stake Version Interval up to the
// provided block height
func AllStakeVersionIntervals(ctx context.Context, dcrdClient *rpcclient.Client, height int64) (StakeVersionIntervals, error) {
	// Use current height to calculate the number of the current SVI
	totalSVIs := 1 + int32((height-activeNetParams.StakeValidationHeight)/activeNetParams.StakeVersionInterval)

	// Get SVIs details from dcrd
	stakeVersionInfoResult, err := dcrdClient.GetStakeVersionInfo(ctx, totalSVIs)
	if err != nil {
		return StakeVersionIntervals{}, err
	}

	svis := StakeVersionIntervals{
		Intervals: stakeVersionInfoResult.Intervals}

	// Reverse the slice of SVIs
	// This makes traversing the set easier later on,
	// because the first element is the first SVI, etc.
	for i := len(svis.Intervals)/2 - 1; i >= 0; i-- {
		opp := len(svis.Intervals) - 1 - i
		svis.Intervals[i], svis.Intervals[opp] = svis.Intervals[opp], svis.Intervals[i]
	}

	// Get max vote version
	var max uint32
	for version := range activeNetParams.Deployments {
		if version > max {
			max = version
		}
	}

	svis.MaxVoteVersion = max

	return svis, nil
}
