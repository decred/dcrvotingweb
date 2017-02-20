// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
)

// Set some high value to check version number
var maxVersion = 10000

// Settings for daemon
var dcrdCertPath = ("/home/user/.dcrd/rpc.cert")
var dcrdServer = "127.0.0.1:19109"
var dcrdUser = "USER"
var dcrdPass = "PASSWORD"

// Daemon Params to use
var activeNetParams = &chaincfg.TestNetParams

// Webserver settings
var listeningPort = ":8000"

// Overall Data structure given to the template to render.
type templateFields struct {

	// Basic information
	BlockHeight int64

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
	BlockVersionWindowLength uint64
	// BlockVersionEnforceThreshold is the activeNetParams of BlockEnforceNumRequired.
	BlockVersionEnforceThreshold int
	// BlockVersionRejectThreshold is the activeNetParams of BlockRejectNumRequired.
	BlockVersionRejectThreshold int
	// BlockVersionCurrent is the currently calculated block version based on the rolling window.
	BlockVersionCurrent int32
	// BlockVersionMostPopular
	BlockVersionMostPopular int32
	// BlockVersionMostPopularPercentage
	BlockVersionMostPopularPercentage float64

	// StakeVersion Information

	// StakeVersionThreshold
	StakeVersionThreshold float64
	// StakeVersionWindowLength
	StakeVersionWindowLength int64
	//StakeVersionWindowVoteTotal
	StakeVersionWindowVoteTotal int64
	//StakeVersionWindowStartHeight
	StakeVersionWindowStartHeight int64
	//StakeVersionWindowEndHeight
	StakeVersionWindowEndHeight int
	// StakeVersionIntervalLabels
	StakeVersionIntervalLabels []string
	//StakeVersionVotesRemaining
	StakeVersionVotesRemaining int64
	//StakeVersionsIntervals
	StakeVersionsIntervals []dcrjson.VersionInterval
	//StakeVersionIntervalResults
	StakeVersionIntervalResults []intervalVersionCounts
	//StakeVersionHeights
	StakeVersionHeights []int64
	// StakeVersionSuccess
	StakeVersionSuccess bool
	// StakeVersionCurrent
	StakeVersionCurrent uint32
	// StakeVersionMostPopular
	StakeVersionMostPopular uint32
	// StakeVersionMostPopularCount
	StakeVersionMostPopularCount uint32
	// StakeVersionMostPopularPercentage
	StakeVersionMostPopularPercentage float64
	// StakeVersionRequiredVotes
	StakeVersionRequiredVotes int32

	// Quorum and Rule Change Information

	// RuleChangeActivationThreshold
	RuleChangeActivationThreshold int32
	// RuleChangeActivationQuorum
	RuleChangeActivationQuorum uint32
	// RuleChangeActivationMultiplier
	RuleChangeActivationMultiplier uint32
	// RuleChangeActivationDivisor
	RuleChangeActivationDivisor uint32
	// RuleChangeActivationWindow
	RuleChangeActivationWindow uint32
	// RuleChangeActivationWindowVotes
	RuleChangeActivationWindowVotes uint32

	// Quorum is a bool that is true if needed number of yes/nos were
	// received (>10%)
	Quorum bool
	// QuorumPercentage
	QuorumPercentage float64
	// QuorumVotes
	QuorumVotes int
	// QuorumVotedPercentage
	QuorumVotedPercentage float64
	// QuorumAbstainedPercentage
	QuorumAbstainedPercentage float64
	// QuorumExpirationDate
	QuorumExpirationDate string

	AgendaID                 string
	AgendaDescription        string
	AgendaChoice1Id          string
	AgendaChoice1Description string
	AgendaChoice1Count       uint32
	AgendaChoice1IsIgnore    bool
	AgendaChoice1Bits        uint16
	AgendaChoice1Progress    float64
	AgendaChoice2Id          string
	AgendaChoice2Description string
	AgendaChoice2Count       uint32
	AgendaChoice2IsIgnore    bool
	AgendaChoice2Bits        uint16
	AgendaChoice2Progress    float64
	AgendaChoice3Id          string
	AgendaChoice3Description string
	AgendaChoice3Count       uint32
	AgendaChoice3IsIgnore    bool
	AgendaChoice3Bits        uint16
	AgendaChoice3Progress    float64
	VotingStarted            bool
	VotingDefined            bool
	VotingLockedin           bool
	VotingFailed             bool
	VoteStartHeight          int64
	VoteEndHeight            int64
	VoteBlockLeft            int64
	TotalVotes               uint32
	VoteExpirationBlock      int64
	ChoiceIds                []string
	ChoicePercentages        []float64
}

// Contains a certain block version's count of blocks in the
// rolling window (which has a length of activeNetParams.BlockUpgradeNumToCheck)
type blockVersions struct {
	RollingWindowLookBacks []int
}

type intervalVersionCounts struct {
	Version uint32
	Count   []uint32
}

var templateInformation = &templateFields{}

var funcMap = template.FuncMap{
	"minus": minus,
}

func minus(a, b int) int {
	return a - b
}

func demoPage(w http.ResponseWriter, r *http.Request) {

	fp := filepath.Join("public", "views", "design_sketch.html")
	tmpl, err := template.New("home").Funcs(funcMap).ParseFiles(fp)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, templateInformation)
	if err != nil {
		panic(err)
	}

}
func updatetemplateInformation(dcrdClient *dcrrpcclient.Client) {

	fmt.Println("updating hard fork information")
	hash, height, err := dcrdClient.GetBestBlock()
	if err != nil {
		fmt.Println(err)
		return
	}
	// Request twice as many, so we can populate the rolling block version window's first
	stakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(activeNetParams.BlockUpgradeNumToCheck*2))
	if err != nil {
		fmt.Println(err)
		return
	}
	block, err := dcrdClient.GetBlockVerbose(hash, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	blockVersionsFound := make(map[int32]*blockVersions)
	blockVersionsHeights := make([]int64, activeNetParams.BlockUpgradeNumToCheck)
	elementNum := 0
	for i := len(stakeVersionResults.StakeVersions)/2 - 1; i >= 0; i-- {
		windowEnd := i + int(activeNetParams.BlockUpgradeNumToCheck)
		blockVersionsHeights[elementNum] = stakeVersionResults.StakeVersions[i].Height
		stakeVersionsWindow := stakeVersionResults.StakeVersions[i:windowEnd]
		for _, stakeVersion := range stakeVersionsWindow {
			_, ok := blockVersionsFound[stakeVersion.BlockVersion]
			if !ok {
				// Had not found this block version yet
				blockVersionsFound[stakeVersion.BlockVersion] = &blockVersions{}
				blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks = make([]int, activeNetParams.BlockUpgradeNumToCheck)
				// Need to populate "back" to fill in values for previously missed window
				for k := 0; k < elementNum; k++ {
					blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks[k] = 0
				}
				blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks[elementNum] = 1
			} else {
				// Already had that block version, so increment
				blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks[elementNum] =
					blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks[elementNum] + 1
			}
		}
		elementNum++
	}
	templateInformation.BlockVersionsHeights = blockVersionsHeights
	templateInformation.BlockVersions = blockVersionsFound

	// Calculate current block version and most popular version (and that percentage)
	templateInformation.BlockVersionCurrent = int32(maxVersion)
	MostPopularBlockVersionCount := 0
	for i, blockVersion := range templateInformation.BlockVersions {
		tipBlockVersionCount := blockVersion.RollingWindowLookBacks[len(blockVersion.RollingWindowLookBacks)-1]
		if tipBlockVersionCount >= int(activeNetParams.BlockRejectNumRequired) {
			// Show Green
			templateInformation.BlockVersionCurrent = i
			templateInformation.BlockVersionMostPopular = i
			templateInformation.BlockVersionMostPopularPercentage = toFixed(float64(tipBlockVersionCount)/float64(activeNetParams.BlockUpgradeNumToCheck)*100, 2)
			templateInformation.BlockVersionSuccess = true
			MostPopularBlockVersionCount = tipBlockVersionCount
		}
		if tipBlockVersionCount > MostPopularBlockVersionCount {
			// Show Red
			MostPopularBlockVersionCount = tipBlockVersionCount
			templateInformation.BlockVersionMostPopular = i
			templateInformation.BlockVersionMostPopularPercentage = toFixed(float64(tipBlockVersionCount)/float64(activeNetParams.BlockUpgradeNumToCheck)*100, 2)
		}
	}
	if templateInformation.BlockVersionCurrent == int32(maxVersion) {
		for i := range templateInformation.BlockVersions {
			if i < templateInformation.BlockVersionCurrent {
				templateInformation.BlockVersionCurrent = i
			}
		}
	}

	blocksIntoStakeVersionWindow := (height - activeNetParams.StakeValidationHeight) % activeNetParams.StakeVersionInterval
	// Request twice as many, so we can populate the rolling block version window's first
	heightstakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(blocksIntoStakeVersionWindow))
	if err != nil {
		fmt.Println(err)
	}
	missedVotesStakeInterval := 0
	for _, stakeVersionResult := range heightstakeVersionResults.StakeVersions {
		missedVotesStakeInterval += int(activeNetParams.TicketsPerBlock) - len(stakeVersionResult.VoterVersions)
	}

	//templateInformation.StakeVersionHeights = dataTickHeights
	//templateInformation.StakeVersions = stakeVersionsFound
	numberOfIntervals := 4
	stakeVersionInfo, err := dcrdClient.GetStakeVersionInfo(int32(numberOfIntervals))
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(stakeVersionInfo.Intervals) == 0 {
		fmt.Println("StakeVersion info did not return usable information, intervals empty")
		return
	}
	templateInformation.StakeVersionsIntervals = stakeVersionInfo.Intervals
	minimumNeededVoteVersions := uint32(100)
	// Hacky way of populating the Vote Version bar graph
	// Each element in each dataset needs counts for each interval
	// For example:
	// version 1: [100, 200, 300, 400]
	voteVersionIntervalResults := make([]intervalVersionCounts, 0)
	voteVersionLabels := make([]string, len(stakeVersionInfo.Intervals))
	for i := len(stakeVersionInfo.Intervals) - 1; i >= 0; i-- {
		interval := stakeVersionInfo.Intervals[i]
		voteVersionLabels[len(stakeVersionInfo.Intervals)-1-i] = fmt.Sprintf("%v - %v", interval.StartHeight, interval.EndHeight)
		for _, voteVersion := range interval.VoteVersions {
			found := false
			for k, result := range voteVersionIntervalResults {
				if result.Version == voteVersion.Version {
					voteVersionIntervalResults[k].Count[len(stakeVersionInfo.Intervals)-1-i] = voteVersion.Count
					voteVersionIntervalResults[k].Version = voteVersion.Version
					found = true
				}
			}
			if !found && voteVersion.Count > minimumNeededVoteVersions {
				voteVersionIntervalResult := intervalVersionCounts{}
				voteVersionIntervalResult.Count = make([]uint32, len(stakeVersionInfo.Intervals))
				voteVersionIntervalResult.Count[len(stakeVersionInfo.Intervals)-1-i] = voteVersion.Count
				voteVersionIntervalResult.Version = voteVersion.Version
				voteVersionIntervalResults = append(voteVersionIntervalResults, voteVersionIntervalResult)
			}
		}
	}
	voteVersionLabels[len(stakeVersionInfo.Intervals)-1] = "Current Interval"
	templateInformation.StakeVersionIntervalResults = voteVersionIntervalResults
	templateInformation.BlockHeight = height
	templateInformation.BlockVersionEnforceThreshold = int(float64(activeNetParams.BlockEnforceNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100)
	templateInformation.BlockVersionRejectThreshold = int(float64(activeNetParams.BlockRejectNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100)
	templateInformation.BlockVersionWindowLength = activeNetParams.BlockUpgradeNumToCheck
	templateInformation.StakeVersionWindowLength = activeNetParams.StakeVersionInterval
	templateInformation.StakeVersionWindowVoteTotal = activeNetParams.StakeVersionInterval*5 - int64(missedVotesStakeInterval)
	templateInformation.StakeVersionThreshold = toFixed(float64(activeNetParams.StakeMajorityMultiplier)/float64(activeNetParams.StakeMajorityDivisor)*100, 0)
	templateInformation.StakeVersionIntervalLabels = voteVersionLabels

	templateInformation.StakeVersionCurrent = block.StakeVersion

	mostPopularVersion := uint32(0)
	mostPopularVersionCount := uint32(0)
	for _, voteVersion := range stakeVersionInfo.Intervals[0].VoteVersions {
		if voteVersion.Version != templateInformation.StakeVersionCurrent &&
			voteVersion.Count > mostPopularVersionCount {
			mostPopularVersion = voteVersion.Version
			mostPopularVersionCount = voteVersion.Count
		}
	}
	if mostPopularVersion <= templateInformation.StakeVersionCurrent {
		templateInformation.StakeVersionSuccess = true
	}
	templateInformation.StakeVersionMostPopularCount = mostPopularVersionCount
	templateInformation.StakeVersionMostPopularPercentage = toFixed(float64(mostPopularVersionCount)/float64(templateInformation.StakeVersionWindowVoteTotal)*100, 2)
	templateInformation.StakeVersionMostPopular = mostPopularVersion
	templateInformation.StakeVersionRequiredVotes = int32(templateInformation.StakeVersionWindowVoteTotal) * activeNetParams.StakeMajorityMultiplier / activeNetParams.StakeMajorityDivisor

	blocksIntoInterval := stakeVersionInfo.Intervals[0].EndHeight - stakeVersionInfo.Intervals[0].StartHeight
	templateInformation.StakeVersionVotesRemaining = (activeNetParams.StakeVersionInterval - blocksIntoInterval) * 5
	// Quorum/vote information
	getVoteInfo, err := dcrdClient.GetVoteInfo(4)
	if err != nil {
		fmt.Println("Get vote info err", err)
		templateInformation.Quorum = false
		return
	}
	templateInformation.Quorum = true
	templateInformation.RuleChangeActivationQuorum = activeNetParams.RuleChangeActivationQuorum
	templateInformation.RuleChangeActivationMultiplier = activeNetParams.RuleChangeActivationMultiplier
	templateInformation.RuleChangeActivationDivisor = activeNetParams.RuleChangeActivationDivisor
	templateInformation.RuleChangeActivationWindow = activeNetParams.RuleChangeActivationInterval
	templateInformation.RuleChangeActivationWindowVotes = templateInformation.RuleChangeActivationWindow * 5
	templateInformation.QuorumPercentage = float64(activeNetParams.RuleChangeActivationQuorum) / float64(templateInformation.RuleChangeActivationWindowVotes) * 100
	templateInformation.QuorumExpirationDate = time.Unix(int64(getVoteInfo.Agendas[0].ExpireTime), int64(0)).Format(time.RFC850)
	templateInformation.QuorumVotedPercentage = toFixed(float64(getVoteInfo.Agendas[0].QuorumProgress*100), 2)
	templateInformation.QuorumAbstainedPercentage = toFixed(float64(getVoteInfo.Agendas[0].Choices[0].Progress*100), 2)
	templateInformation.AgendaID = getVoteInfo.Agendas[0].Id
	templateInformation.AgendaDescription = getVoteInfo.Agendas[0].Description
	// XX instread of static linking there should be itteration trough the Choices array
	templateInformation.AgendaChoice1Id = getVoteInfo.Agendas[0].Choices[0].Id
	templateInformation.AgendaChoice1Description = getVoteInfo.Agendas[0].Choices[0].Description
	templateInformation.AgendaChoice1Count = getVoteInfo.Agendas[0].Choices[0].Count
	templateInformation.AgendaChoice1IsIgnore = getVoteInfo.Agendas[0].Choices[0].IsIgnore
	templateInformation.AgendaChoice1Bits = getVoteInfo.Agendas[0].Choices[0].Bits
	templateInformation.AgendaChoice1Progress = toFixed(float64(getVoteInfo.Agendas[0].Choices[0].Progress*100), 2)
	templateInformation.AgendaChoice2Id = getVoteInfo.Agendas[0].Choices[1].Id
	templateInformation.AgendaChoice2Description = getVoteInfo.Agendas[0].Choices[1].Description
	templateInformation.AgendaChoice2Count = getVoteInfo.Agendas[0].Choices[1].Count
	templateInformation.AgendaChoice2IsIgnore = getVoteInfo.Agendas[0].Choices[1].IsIgnore
	templateInformation.AgendaChoice2Bits = getVoteInfo.Agendas[0].Choices[1].Bits
	templateInformation.AgendaChoice2Progress = toFixed(float64(getVoteInfo.Agendas[0].Choices[1].Progress*100), 2)
	templateInformation.AgendaChoice3Id = getVoteInfo.Agendas[0].Choices[2].Id
	templateInformation.AgendaChoice3Description = getVoteInfo.Agendas[0].Choices[2].Description
	templateInformation.AgendaChoice3Count = getVoteInfo.Agendas[0].Choices[2].Count
	templateInformation.AgendaChoice3IsIgnore = getVoteInfo.Agendas[0].Choices[2].IsIgnore
	templateInformation.AgendaChoice3Bits = getVoteInfo.Agendas[0].Choices[2].Bits
	templateInformation.AgendaChoice3Progress = toFixed(float64(getVoteInfo.Agendas[0].Choices[2].Progress*100), 2)
	templateInformation.VoteStartHeight = getVoteInfo.StartHeight
	templateInformation.VoteEndHeight = getVoteInfo.EndHeight
	templateInformation.VoteBlockLeft = getVoteInfo.EndHeight - getVoteInfo.CurrentHeight
	templateInformation.TotalVotes = getVoteInfo.TotalVotes
	templateInformation.VotingStarted = getVoteInfo.Agendas[0].Status == "started"
	templateInformation.VotingDefined = getVoteInfo.Agendas[0].Status == "defined"
	templateInformation.VotingLockedin = getVoteInfo.Agendas[0].Status == "lockedin"
	templateInformation.VotingFailed = getVoteInfo.Agendas[0].Status == "failed"

	/// XXX need to calculate expiration block
	templateInformation.VoteExpirationBlock = int64(210001)

	choiceIds := make([]string, len(getVoteInfo.Agendas[0].Choices))
	choicePercentages := make([]float64, len(getVoteInfo.Agendas[0].Choices))
	for i, choice := range getVoteInfo.Agendas[0].Choices {
		choiceIds[i] = choice.Id
		choicePercentages[i] = choice.Progress
	}
	templateInformation.ChoiceIds = choiceIds
	templateInformation.ChoicePercentages = choicePercentages

}

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/"] = demoPage

	connectChan := make(chan int64, 100)
	quit := make(chan struct{})
	ntfnHandlersDaemon := dcrrpcclient.NotificationHandlers{
		OnBlockConnected: func(serializedBlockHeader []byte, transactions [][]byte) {
			var blockHeader wire.BlockHeader
			err := blockHeader.Deserialize(bytes.NewReader(serializedBlockHeader))
			if err != nil {
				fmt.Printf("Failed to deserialize block header: %v\n", err.Error())
				return
			}
			fmt.Println("got a new block passing it", blockHeader.Height)
			connectChan <- int64(blockHeader.Height)
		},
	}
	var dcrdCerts []byte
	dcrdCerts, err := ioutil.ReadFile(dcrdCertPath)
	if err != nil {
		fmt.Printf("Failed to read dcrd cert file at %s: %s\n", dcrdCertPath,
			err.Error())
		os.Exit(1)
	}
	fmt.Printf("Attempting to connect to dcrd RPC %s as user %s "+
		"using certificate located in %s\n",
		dcrdServer, dcrdUser, dcrdCertPath)
	connCfgDaemon := &dcrrpcclient.ConnConfig{
		Host:         dcrdServer,
		Endpoint:     "ws",
		User:         dcrdUser,
		Pass:         dcrdPass,
		Certificates: dcrdCerts,
		DisableTLS:   false,
	}
	dcrdClient, err := dcrrpcclient.New(connCfgDaemon, &ntfnHandlersDaemon)
	if err != nil {
		fmt.Printf("Failed to start dcrd rpcclient: %s\n", err.Error())
		os.Exit(1)
	}

	if err := dcrdClient.NotifyBlocks(); err != nil {
		fmt.Printf("Failed to start register daemon rpc client for  "+
			"block notifications: %s\n", err.Error())
		os.Exit(1)
	}
	updatetemplateInformation(dcrdClient)
	go func() {
		for {
			select {
			case height := <-connectChan:
				fmt.Printf("Block height %v connected\n", height)
				updatetemplateInformation(dcrdClient)
			case <-quit:
				close(quit)
				dcrdClient.Disconnect()
				fmt.Printf("\nClosing hardfork demo.\n")
				os.Exit(1)
				break
			}
		}
	}()
	http.HandleFunc("/", demoPage)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("public/js/"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("public/css/"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("public/fonts/"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("public/images/"))))
	err = http.ListenAndServe(listeningPort, nil)
	if err != nil {
		fmt.Printf("Failed to bind http server: %s\n", err.Error())
	}
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
