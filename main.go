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

// Overall Data structure given to the template to render
type hardForkInfo struct {
	BlockHeight                       int64
	BlockVersions                     map[int32]*blockVersions
	BlockVersionSuccess               bool
	BlockVersionsHeights              []int64
	BlockVersionWindowLength          uint64
	BlockVersionEnforceThreshold      int
	BlockVersionRejectThreshold       int
	CurrentCalculatedBlockVersion     int32
	BlockCountAtLatestVersion         int
	StakeVersionThreshold             float64
	StakeVersionWindowLength          int64
	StakeVersionWindowVoteTotal       int64
	StakeVersionWindowStartHeight     int64
	StakeVersionWindowEndHeight       int
	MostPopularBlockVersion           int32
	MostPopularBlockVersionPercentage float64
	VoteVersionThreshold              int32
	StakeVersionIntervalLabels        []string
	StakeVersionVotesRemaining        int64
	StakeVersionsIntervals            []dcrjson.VersionInterval
	StakeVersionIntervalResults       []intervalVersionCounts
	StakeVersionHeights               []int64
	StakeVersionSuccess               bool
	CurrentCalculatedStakeVersion     uint32
	MostPopularStakeVersion           uint32
	MostPopularStakeVersionCount      uint32
	RuleChangeActivationQuorum        uint32
	RuleChangeActivationMultiplier    uint32
	RuleChangeActivationDivisor       uint32
	RuleChangeActivationWindow        uint32
	RuleChangeActivationWindowVotes   uint32

	Quorum                    bool
	QuorumPercentage          float64
	QuorumVotes               int
	QuorumVotedPercentage     float64
	QuorumAbstainedPercentage float64
	QuorumExpirationDate      string
	AgendaID                  string
	AgendaDescription         string
	VoteStartHeight           int64
	VoteEndHeight             int64
	VoteBlockLeft             int64
	VoteExpirationBlock       int64

	ChoiceIds         []string
	ChoicePercentages []float64
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

var hardForkInformation = &hardForkInfo{}

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
	err = tmpl.Execute(w, hardForkInformation)
	if err != nil {
		panic(err)
	}

}
func updateHardForkInformation(dcrdClient *dcrrpcclient.Client) {

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
	hardForkInformation.BlockVersionsHeights = blockVersionsHeights
	hardForkInformation.BlockVersions = blockVersionsFound

	// Calculate current block version and most popular version (and that percentage)
	hardForkInformation.CurrentCalculatedBlockVersion = int32(maxVersion)
	MostPopularBlockVersionCount := 0
	for i, blockVersion := range hardForkInformation.BlockVersions {
		tipBlockVersionCount := blockVersion.RollingWindowLookBacks[len(blockVersion.RollingWindowLookBacks)-1]
		if tipBlockVersionCount >= int(activeNetParams.BlockRejectNumRequired) {
			// Show Green
			hardForkInformation.CurrentCalculatedBlockVersion = i
			hardForkInformation.MostPopularBlockVersion = i
			hardForkInformation.MostPopularBlockVersionPercentage = toFixed(float64(tipBlockVersionCount)/float64(activeNetParams.BlockUpgradeNumToCheck)*100, 2)
			hardForkInformation.BlockVersionSuccess = true
			MostPopularBlockVersionCount = tipBlockVersionCount
		}
		if tipBlockVersionCount > MostPopularBlockVersionCount {
			// Show Red
			MostPopularBlockVersionCount = tipBlockVersionCount
			hardForkInformation.MostPopularBlockVersion = i
			hardForkInformation.MostPopularBlockVersionPercentage = toFixed(float64(tipBlockVersionCount)/float64(activeNetParams.BlockUpgradeNumToCheck)*100, 2)
		}
	}
	if hardForkInformation.CurrentCalculatedBlockVersion == int32(maxVersion) {
		for i := range hardForkInformation.BlockVersions {
			if i < hardForkInformation.CurrentCalculatedBlockVersion {
				hardForkInformation.CurrentCalculatedBlockVersion = i
			}
		}
	}
	/*
		blocksIntoStakeVersionWindow := (height - activeNetParams.StakeValidationHeight) % activeNetParams.StakeVersionInterval
		// Request twice as many, so we can populate the rolling block version window's first
		heightstakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
			int32(blocksIntoStakeVersionWindow))
		if err != nil {
			fmt.Println(err)
		}
			stakeVersionsFound := make(map[uint32]*stakeVersions)
			stakeVersionsHeights := make([]int64, len(heightstakeVersionResults.StakeVersions))
			elementNum = 0
			for i := len(heightstakeVersionResults.StakeVersions) - 1; i >= 0; i-- {
				stakeVersion := heightstakeVersionResults.StakeVersions[i]
				stakeVersionsHeights[elementNum] = stakeVersion.Height
				for _, vote := range stakeVersion.VoterVersions {
					_, ok := stakeVersionsFound[vote]
					if !ok {
						// Had not found this block version yet
						stakeVersionsFound[vote] = &stakeVersions{}
						stakeVersionsFound[vote].StaticWindowVoteCounts = make([]int, len(heightstakeVersionResults.StakeVersions))
						// Need to populate "back" to fill in values for previously missed window
						for k := 0; k < elementNum; k++ {
							stakeVersionsFound[vote].StaticWindowVoteCounts[k] = 0
						}
						stakeVersionsFound[vote].StaticWindowVoteCounts[elementNum] = 1
					} else {
						if elementNum == 0 {
							stakeVersionsFound[vote].StaticWindowVoteCounts[elementNum] = 1
						} else {
							stakeVersionsFound[vote].StaticWindowVoteCounts[elementNum] =
								stakeVersionsFound[vote].StaticWindowVoteCounts[elementNum] + 1
						}
					}
				}
				for voteVersion := range stakeVersionsFound {
					if elementNum > 0 {
						stakeVersionsFound[voteVersion].StaticWindowVoteCounts[elementNum] +=
							stakeVersionsFound[voteVersion].StaticWindowVoteCounts[elementNum-1]
					}
				}
				elementNum++
			}
			numDataPoints := 24
			dataTickLength := int(activeNetParams.StakeVersionInterval) / numDataPoints
			dataTickHeights := make([]int64, numDataPoints)
			for vote := range stakeVersionsFound {
				dataTickedVoteCounts := make([]int, numDataPoints)
				dataTicketNumber := 0
				for elementNum, counts := range stakeVersionsFound[vote].StaticWindowVoteCounts {
					if elementNum%dataTickLength == 0 {
						dataTickedVoteCounts[dataTicketNumber] = counts
						stakeVersionsFound[vote].CurrentTotalVotes = counts
						dataTickHeights[dataTicketNumber] = stakeVersionsHeights[elementNum]
						dataTicketNumber++
					}
				}
				stakeVersionsFound[vote].StaticWindowVoteCounts = dataTickedVoteCounts

			}

			// Fill in heights for any that weren't populated
			for i := range dataTickHeights {
				if dataTickHeights[i] == 0 && i != 0 {
					dataTickHeights[i] = dataTickHeights[i-1] + (int64(activeNetParams.StakeVersionInterval) / int64(numDataPoints))
				} else if dataTickHeights[i] == 0 && i == 0 {
					dataTickHeights[i] = height
				}
			}

			// Add end of window height dataTick
			dataTickHeights = append(dataTickHeights, dataTickHeights[len(dataTickHeights)-1]+int64(activeNetParams.StakeVersionInterval)/int64(numDataPoints))
	*/
	//hardForkInformation.StakeVersionHeights = dataTickHeights
	//hardForkInformation.StakeVersions = stakeVersionsFound
	numberOfIntervals := 4
	stakeVersionInfo, err := dcrdClient.GetStakeVersionInfo(int32(numberOfIntervals))
	if err != nil {
		fmt.Println(err)
		return
	}

	hardForkInformation.StakeVersionsIntervals = stakeVersionInfo.Intervals
	minimumNeededVoteVersions := uint32(100)
	// Hacky way of populating the Vote Version bar graph
	// Each element in each dataset needs counts for each interval
	// For example:
	// version 1: [100, 200, 300, 400]
	voteVersionIntervalResults := make([]intervalVersionCounts, 0)
	for i, interval := range stakeVersionInfo.Intervals {
		for _, voteVersion := range interval.VoteVersions {
			found := false
			for k, result := range voteVersionIntervalResults {
				if result.Version == voteVersion.Version {
					voteVersionIntervalResults[k].Count[i] = voteVersion.Count
					voteVersionIntervalResults[k].Version = voteVersion.Version
					found = true
				}
			}
			if !found && voteVersion.Count > minimumNeededVoteVersions {
				voteVersionIntervalResult := intervalVersionCounts{}
				voteVersionIntervalResult.Count = make([]uint32, len(stakeVersionInfo.Intervals))
				voteVersionIntervalResult.Count[i] = voteVersion.Count
				voteVersionIntervalResult.Version = voteVersion.Version
				voteVersionIntervalResults = append(voteVersionIntervalResults, voteVersionIntervalResult)
			}
		}
	}
	hardForkInformation.StakeVersionIntervalResults = voteVersionIntervalResults
	hardForkInformation.BlockHeight = height
	hardForkInformation.BlockVersionEnforceThreshold = int(float64(activeNetParams.BlockEnforceNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100)
	hardForkInformation.BlockVersionRejectThreshold = int(float64(activeNetParams.BlockRejectNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100)
	hardForkInformation.BlockVersionWindowLength = activeNetParams.BlockUpgradeNumToCheck
	hardForkInformation.StakeVersionWindowLength = activeNetParams.StakeVersionInterval
	hardForkInformation.StakeVersionWindowVoteTotal = activeNetParams.StakeVersionInterval * 5

	hardForkInformation.StakeVersionIntervalLabels = []string{"Current Window", "2nd", "3rd", "4th"}

	if len(stakeVersionInfo.Intervals) > 2 {
		// Get the stakeversion from that most recent full window
		// XXX THIS IS NOT QUITE RIGHT
		hardForkInformation.CurrentCalculatedStakeVersion = stakeVersionInfo.Intervals[1].PoSVersions[0].Version
	}

	mostPopularVersion := uint32(0)
	mostPopularVersionCount := uint32(0)
	for _, voteVersion := range stakeVersionInfo.Intervals[0].VoteVersions {
		if voteVersion.Version != hardForkInformation.CurrentCalculatedStakeVersion &&
			voteVersion.Count > mostPopularVersionCount {
			mostPopularVersion = voteVersion.Version
			mostPopularVersionCount = voteVersion.Count
		}
	}
	voteVersionThreshold := int32(hardForkInformation.StakeVersionWindowVoteTotal) * activeNetParams.StakeMajorityMultiplier / activeNetParams.StakeMajorityDivisor
	if mostPopularVersionCount > uint32(voteVersionThreshold) {
		hardForkInformation.StakeVersionSuccess = true
	}
	hardForkInformation.MostPopularStakeVersionCount = mostPopularVersionCount
	hardForkInformation.MostPopularStakeVersion = mostPopularVersion
	hardForkInformation.VoteVersionThreshold = voteVersionThreshold

	blocksIntoInterval := stakeVersionInfo.Intervals[0].EndHeight - stakeVersionInfo.Intervals[0].StartHeight
	hardForkInformation.StakeVersionVotesRemaining = (activeNetParams.StakeVersionInterval - blocksIntoInterval) * 5
	// Quorum/vote information
	getVoteInfo, err := dcrdClient.GetVoteInfo(4)
	if err != nil {
		fmt.Println("Get vote info err", err)
		hardForkInformation.Quorum = false
		return
	}
	hardForkInformation.Quorum = true
	hardForkInformation.RuleChangeActivationQuorum = activeNetParams.RuleChangeActivationQuorum
	hardForkInformation.RuleChangeActivationMultiplier = activeNetParams.RuleChangeActivationMultiplier
	hardForkInformation.RuleChangeActivationDivisor = activeNetParams.RuleChangeActivationDivisor
	hardForkInformation.RuleChangeActivationWindow = activeNetParams.RuleChangeActivationInterval
	hardForkInformation.RuleChangeActivationWindowVotes = hardForkInformation.RuleChangeActivationWindow * 5
	hardForkInformation.QuorumPercentage = float64(activeNetParams.RuleChangeActivationQuorum) / float64(hardForkInformation.RuleChangeActivationWindowVotes) * 100
	hardForkInformation.QuorumExpirationDate = time.Unix(int64(getVoteInfo.Agendas[0].ExpireTime), int64(0)).Format(time.RFC850)
	hardForkInformation.QuorumVotedPercentage = getVoteInfo.Agendas[0].QuorumProgress
	hardForkInformation.QuorumAbstainedPercentage = (float64(1) - getVoteInfo.Agendas[0].QuorumProgress) * 100
	hardForkInformation.AgendaID = getVoteInfo.Agendas[0].Id
	hardForkInformation.AgendaDescription = getVoteInfo.Agendas[0].Description
	hardForkInformation.VoteStartHeight = getVoteInfo.StartHeight
	hardForkInformation.VoteEndHeight = getVoteInfo.EndHeight
	hardForkInformation.VoteBlockLeft = getVoteInfo.EndHeight - getVoteInfo.CurrentHeight

	/// XXX need to calculate expiration block
	hardForkInformation.VoteExpirationBlock = int64(210001)

	choiceIds := make([]string, len(getVoteInfo.Agendas[0].Choices))
	choicePercentages := make([]float64, len(getVoteInfo.Agendas[0].Choices))
	for i, choice := range getVoteInfo.Agendas[0].Choices {
		choiceIds[i] = choice.Id
		choicePercentages[i] = choice.Progress
	}
	hardForkInformation.ChoiceIds = choiceIds
	hardForkInformation.ChoicePercentages = choicePercentages

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
	updateHardForkInformation(dcrdClient)
	go func() {
		for {
			select {
			case height := <-connectChan:
				fmt.Printf("Block height %v connected\n", height)
				updateHardForkInformation(dcrdClient)
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
