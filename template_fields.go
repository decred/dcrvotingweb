// Copyright (c) 2017-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/decred/dcrd/rpc/jsonrpc/types/v3"
)

// Overall data structure given to the template to render.
type templateFields struct {

	// Network
	Network string

	// Basic information
	BlockHeight int64
	// Base URL for the block explorer
	BlockExplorerURL string
	// BlockVersion Information
	//
	// BlockVersions is the data after it has been prepared for graphing.
	BlockVersions map[int32]*blockVersions
	// BlockVersionHeights is an array of Block heights for graph's x axis.
	BlockVersionsHeights []int64
	// BlockVersionSuccess is a bool whether or not BlockVersion has
	// successfully tripped over to the new version.
	BlockVersionSuccess bool
	// BlockVersionWindowLength is the activeNetParams of BlockUpgradeNumToCheck
	// rolling window length.
	BlockVersionWindowLength int64
	// BlockVersionRejectThreshold is the activeNetParams of BlockRejectNumRequired.
	BlockVersionRejectThreshold int
	// BlockVersionCurrent is the currently calculated block version based on the rolling window.
	BlockVersionCurrent int32
	// BlockVersionNext is the next block version.
	BlockVersionNext int32
	// BlockVersionNextPercentage is the share of the next block version in the current rolling window.
	BlockVersionNextPercentage float64

	// StakeVersion Information
	//
	// StakeVersionThreshold is the activeNetParams of StakeVersion threshold made into a float for display
	StakeVersionThreshold float64
	// StakeVersionWindowLength is the activeNetParams of StakeVersionInterval
	StakeVersionWindowLength int64
	// CurrentSVIStartHeight is when the current SVI started
	CurrentSVIStartHeight int64
	// CurrentSVIStartHeight is when the current SVI ends
	CurrentSVIEndHeight int64
	// StakeVersionIntervalLabels are labels for the bar graph for each of the past 4 fixed stake version intervals.
	StakeVersionIntervalLabels []string
	// StakeVersionsIntervals  is the data received from GetStakeVersionInfo json-rpc call to dcrd.
	StakeVersionsIntervals []types.VersionInterval
	// StakeVersionIntervalResults is the data after being analyzed for graph displaying.
	StakeVersionIntervalResults []intervalVersionCounts
	// PosUpgrade contains fields describing the stake version upgrade.
	PosUpgrade posUpgrade
	// StakeVersionCurrent is the StakeVersion that has been seen in the recent block header.
	StakeVersionCurrent uint32
	// StakeVersionMostPopular is the most popular stake version that is NOT the current stake version.
	StakeVersionMostPopular uint32
	// StakeVersionMostPopularPercentage is the percentage of most popular stake versions out of possible votes.
	StakeVersionMostPopularPercentage float64
	// StakeVersionTimeRemaining is a string to show how much estimated time is remaining in the stake version interval.
	StakeVersionTimeRemaining string

	// Length of the static rule change interval
	RuleChangeActivationInterval int64
	// Agendas contains all the agendas and their statuses
	Agendas []Agenda
	// Phase Upgrading or Voting
	IsUpgrading bool
	// Pending Activation to show that voting has ceased and activation will begin shortly
	PendingActivation bool
	// Rules Activated to show that all rules have activated
	RulesActivated bool
}
