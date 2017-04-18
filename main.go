// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
)

// Set some high value to check version number
var maxVersion = 10000

// Settings for daemon
var host = flag.String("host", "127.0.0.1:19109", "node RPC host:port")
var user = flag.String("user", "dcrd", "node RPC username")
var pass = flag.String("pass", "bananas", "node RPC password")
var cert = flag.String("cert", "dcrd.cert", "node RPC TLS certificate (when notls=false)")
var notls = flag.Bool("notls", false, "Disable use of TLS for node connection")
var listenPort = flag.String("listen", ":8000", "web app listening port")

// Daemon Params to use
var activeNetParams = &chaincfg.TestNet2Params

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
	// BlockVersionMostPopular is the calculated most popular block version that is NOT current version.
	BlockVersionMostPopular int32
	// BlockVersionMostPopularPercentage is the percentage of the most popular block version
	BlockVersionMostPopularPercentage float64

	// StakeVersion Information
	//
	// StakeVersionThreshold is the activeNetParams of StakeVersion threshold made into a float for display
	StakeVersionThreshold float64
	// StakeVersionWindowLength is the activeNetParams of StakeVersionInterval
	StakeVersionWindowLength int64
	// StakeVersionWindowVoteTotal is the number of total possible votes in the windows.
	// It is reduced by number of observed missed votes thus far in the window.
	StakeVersionWindowVoteTotal int64
	// StakeVersionIntervalLabels are labels for the bar graph for each of the past 4 fixed stake version intervals.
	StakeVersionIntervalLabels []string
	// StakeVersionVotesRemaining is the calculated number of votes possibly remaining in the current stake version interval.
	StakeVersionVotesRemaining int64
	// StakeVersionsIntervals  is the data received from GetStakeVersionInfo json-rpc call to dcrd.
	StakeVersionsIntervals []dcrjson.VersionInterval
	// StakeVersionIntervalResults is the data after being analyzed for graph displaying.
	StakeVersionIntervalResults []intervalVersionCounts
	// StakeVersionSuccess is a bool for whether or not the StakeVersion has rolled over in this window.
	StakeVersionSuccess bool
	// StakeVersionCurrent is the StakeVersion that has been seen in the recent block header.
	StakeVersionCurrent uint32
	// StakeVersionMostPopular is the most popular stake version that is NOT the current stake version.
	StakeVersionMostPopular uint32
	// StakeVersionMostPopularCount is the count of most popular stake versions.
	StakeVersionMostPopularCount uint32
	// StakeVersionMostPopularPercentage is the percentage of most popular stake versions out of possible votes.
	StakeVersionMostPopularPercentage float64
	// StakeVersionRequiredVotes is the number of stake version votes required for the stake version to change.
	StakeVersionRequiredVotes int32

	// Quorum and Rule Change Information
	// RuleChangeActivationQuorum is the activeNetParams of RuleChangeActivationQuorum
	RuleChangeActivationQuorum uint32
	// Quorum is a bool that is true if needed number of yes/nos were
	// received (>10%).
	Quorum bool
	// QuorumThreshold is the percentage required for the RuleChange to become active.
	QuorumThreshold float64
	// QuorumVotedPercentage is the percentage of progress toward quorum XXX needs to be fixed.
	QuorumVotedPercentage float64
	// QuorumAbstainedPercentage is the abstain percentage.
	QuorumAbstainedPercentage float64
	// QuorumExpirationDate is the date in which the agenda is scheduled to expire.
	QuorumExpirationDate string
	// All of these are already contained in GetVoteInfoResult, so we need to refactor the html
	// to properly use these.
	AgendaLockedinPercentage float64
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
	// These are bools to determine what state a given agenda is at.  These need to be refactored with stuff above.
	VotingStarted  bool
	VotingDefined  bool
	VotingLockedin bool
	VotingFailed   bool
	VotingActive   bool
	QuorumAchieved bool
	// GetVoteInfoResult has all the raw data returned from getvoteinfo json-rpc command.
	GetVoteInfoResult *dcrjson.GetVoteInfoResult
	// Choice Ids and percentages that have been scrubbed for graphing.
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

// Set all activeNetParams fields since they don't change at runtime
var templateInformation = &templateFields{
	// BlockVersion params
	BlockVersionEnforceThreshold: int(float64(activeNetParams.BlockEnforceNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100),
	BlockVersionRejectThreshold:  int(float64(activeNetParams.BlockRejectNumRequired) / float64(activeNetParams.BlockUpgradeNumToCheck) * 100),
	BlockVersionWindowLength:     activeNetParams.BlockUpgradeNumToCheck,
	// StakeVersion params
	StakeVersionWindowLength: activeNetParams.StakeVersionInterval,
	StakeVersionThreshold:    toFixed(float64(activeNetParams.StakeMajorityMultiplier)/float64(activeNetParams.StakeMajorityDivisor)*100, 0),
	// RuleChange params
	RuleChangeActivationQuorum: activeNetParams.RuleChangeActivationQuorum,
	QuorumThreshold:            float64(activeNetParams.RuleChangeActivationQuorum) / float64(activeNetParams.RuleChangeActivationInterval*uint32(activeNetParams.TicketsPerBlock)) * 100,
}

var funcMap = template.FuncMap{
	"minus":   minus,
	"minus64": minus64,
}

func minus(a, b int) int {
	return a - b
}
func minus64(a, b int64) int64 {
	return a - b
}

// renders the 'home' template that is current located at "design_sketch.html".
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

// updatetemplateInformation is called on startup and upon every block connected notification received.
func updatetemplateInformation(dcrdClient *dcrrpcclient.Client) {
	fmt.Println("updating hard fork information")

	// Get the current best block (height and hash)
	hash, height, err := dcrdClient.GetBestBlock()
	if err != nil {
		fmt.Println(err)
		return
	}
	// Set Current block height
	templateInformation.BlockHeight = height

	// Request the current block to parse its blockHeader
	block, err := dcrdClient.GetBlockVerbose(hash, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Request GetStakeVersions to receive information about past block versions.
	//
	// Request twice as many, so we can populate the rolling block version window's first
	stakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(templateInformation.BlockVersionWindowLength*2))
	if err != nil {
		fmt.Println(err)
		return
	}
	blockVersionsFound := make(map[int32]*blockVersions)
	blockVersionsHeights := make([]int64, templateInformation.BlockVersionWindowLength)
	elementNum := 0

	// The algorithm starts at the middle of the GetStakeVersionResults and decrements backwards toward
	// the beginning of the list.  This is due to GetStakeVersionResults.StakeVersions being ordered
	// from most recent blocks to oldest. (ie [0] == current, [len] == oldest).  So by starting in the middle
	// we then can calculate that first blocks rolling window result then become one block 'more recent'
	// and calculate that blocks rolling window results.
	for i := len(stakeVersionResults.StakeVersions)/2 - 1; i >= 0; i-- {
		// Calculate the last block element in the window
		windowEnd := i + int(templateInformation.BlockVersionWindowLength)
		// blockVersionsHeights lets us have a correctly ordered list of blockheights for xaxis label
		blockVersionsHeights[elementNum] = stakeVersionResults.StakeVersions[i].Height
		// Define rolling window range for this current block (i)
		stakeVersionsWindow := stakeVersionResults.StakeVersions[i:windowEnd]
		for _, stakeVersion := range stakeVersionsWindow {
			_, ok := blockVersionsFound[stakeVersion.BlockVersion]
			if !ok {
				// Had not found this block version yet
				blockVersionsFound[stakeVersion.BlockVersion] = &blockVersions{}
				blockVersionsFound[stakeVersion.BlockVersion].RollingWindowLookBacks = make([]int, templateInformation.BlockVersionWindowLength)
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
	mostPopularBlockVersionCount := 0
	// Range across rolling window block version results.  If any of the rolling window look back
	// counts are greater than the required threshold then that is assured to be the current block
	// version at point in the chain.
	for i, blockVersion := range templateInformation.BlockVersions {
		tipBlockVersionCount := blockVersion.RollingWindowLookBacks[len(blockVersion.RollingWindowLookBacks)-1]
		if tipBlockVersionCount >= int(activeNetParams.BlockRejectNumRequired) {
			// Show Green
			templateInformation.BlockVersionCurrent = i
			templateInformation.BlockVersionMostPopular = i
			templateInformation.BlockVersionMostPopularPercentage = toFixed(float64(tipBlockVersionCount)/float64(templateInformation.BlockVersionWindowLength)*100, 2)
			templateInformation.BlockVersionSuccess = true
			mostPopularBlockVersionCount = tipBlockVersionCount
		}
		if tipBlockVersionCount > mostPopularBlockVersionCount {
			// Show Red
			mostPopularBlockVersionCount = tipBlockVersionCount
			templateInformation.BlockVersionMostPopular = i
			templateInformation.BlockVersionMostPopularPercentage = toFixed(float64(tipBlockVersionCount)/float64(templateInformation.BlockVersionWindowLength)*100, 2)
		}
	}
	if templateInformation.BlockVersionCurrent == int32(maxVersion) {
		for i := range templateInformation.BlockVersions {
			if i < templateInformation.BlockVersionCurrent {
				templateInformation.BlockVersionCurrent = i
			}
		}
	}

	blocksIntoStakeVersionWindow := (templateInformation.BlockHeight - activeNetParams.StakeValidationHeight) % activeNetParams.StakeVersionInterval
	// Request twice as many, so we can populate the rolling block version window's first
	heightstakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(blocksIntoStakeVersionWindow))
	if err != nil {
		fmt.Println(err)
	}
	missedVotesStakeInterval := 0
	for _, stakeVersionResult := range heightstakeVersionResults.StakeVersions {
		missedVotesStakeInterval += int(activeNetParams.TicketsPerBlock) - len(stakeVersionResult.Votes)
	}

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
	stakeVersionIntervalResults := make([]intervalVersionCounts, 0)
	stakeVersionLabels := make([]string, len(stakeVersionInfo.Intervals))
	for i := len(stakeVersionInfo.Intervals) - 1; i >= 0; i-- {
		interval := stakeVersionInfo.Intervals[i]
		stakeVersionLabels[len(stakeVersionInfo.Intervals)-1-i] = fmt.Sprintf("%v - %v", interval.StartHeight, interval.EndHeight)
		for _, stakeVersion := range interval.VoteVersions {
			found := false
			for k, result := range stakeVersionIntervalResults {
				if result.Version == stakeVersion.Version {
					stakeVersionIntervalResults[k].Count[len(stakeVersionInfo.Intervals)-1-i] = stakeVersion.Count
					stakeVersionIntervalResults[k].Version = stakeVersion.Version
					found = true
				}
			}
			if !found && stakeVersion.Count > minimumNeededVoteVersions {
				stakeVersionIntervalResult := intervalVersionCounts{}
				stakeVersionIntervalResult.Count = make([]uint32, len(stakeVersionInfo.Intervals))
				stakeVersionIntervalResult.Count[len(stakeVersionInfo.Intervals)-1-i] = stakeVersion.Count
				stakeVersionIntervalResult.Version = stakeVersion.Version
				stakeVersionIntervalResults = append(stakeVersionIntervalResults, stakeVersionIntervalResult)
			}
		}
	}
	stakeVersionLabels[len(stakeVersionInfo.Intervals)-1] = "Current Interval"
	templateInformation.StakeVersionIntervalResults = stakeVersionIntervalResults
	templateInformation.StakeVersionWindowVoteTotal = activeNetParams.StakeVersionInterval*5 - int64(missedVotesStakeInterval)
	templateInformation.StakeVersionIntervalLabels = stakeVersionLabels
	templateInformation.StakeVersionCurrent = block.StakeVersion

	mostPopularVersion := uint32(0)
	mostPopularVersionCount := uint32(0)
	for _, stakeVersion := range stakeVersionInfo.Intervals[0].VoteVersions {
		if stakeVersion.Version != templateInformation.StakeVersionCurrent &&
			stakeVersion.Count > mostPopularVersionCount {
			mostPopularVersion = stakeVersion.Version
			mostPopularVersionCount = stakeVersion.Count
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
	templateInformation.GetVoteInfoResult = getVoteInfo

	// Set Quorum to true since we got a valid response back from GetVoteInfoResult
	templateInformation.Quorum = true

	// These fields can be refactored out of GetVoteInfoResults
	templateInformation.QuorumExpirationDate = time.Unix(int64(getVoteInfo.Agendas[0].ExpireTime), int64(0)).Format(time.RFC850)
	templateInformation.QuorumVotedPercentage = toFixed(getVoteInfo.Agendas[0].QuorumProgress*100, 2)
	templateInformation.QuorumAbstainedPercentage = toFixed(getVoteInfo.Agendas[0].Choices[0].Progress*100, 2)
	templateInformation.AgendaID = getVoteInfo.Agendas[0].Id
	templateInformation.AgendaDescription = getVoteInfo.Agendas[0].Description

	// Status LockedIn Circle3 Ring Indicates BlocksLeft until old versions gets denied
	lockedinBlocksleft := float64(getVoteInfo.EndHeight) - float64(getVoteInfo.CurrentHeight)
	lockedinWindowsize := float64(getVoteInfo.EndHeight) - float64(getVoteInfo.StartHeight)
	lockedinPercentage := lockedinWindowsize / 100
	templateInformation.AgendaLockedinPercentage = toFixed(lockedinBlocksleft/lockedinPercentage, 2)

	// Recalculating Vote Percentages for Donut Chart

	// XX instread of static linking there should be itteration trough the Choices array
	templateInformation.AgendaChoice1Id = getVoteInfo.Agendas[0].Choices[0].Id
	templateInformation.AgendaChoice1Description = getVoteInfo.Agendas[0].Choices[0].Description
	templateInformation.AgendaChoice1Count = getVoteInfo.Agendas[0].Choices[0].Count
	templateInformation.AgendaChoice1IsIgnore = getVoteInfo.Agendas[0].Choices[0].IsIgnore
	templateInformation.AgendaChoice1Bits = getVoteInfo.Agendas[0].Choices[0].Bits
	templateInformation.AgendaChoice1Progress = toFixed(getVoteInfo.Agendas[0].Choices[0].Progress*100, 2)
	templateInformation.AgendaChoice2Id = getVoteInfo.Agendas[0].Choices[1].Id
	templateInformation.AgendaChoice2Description = getVoteInfo.Agendas[0].Choices[1].Description
	templateInformation.AgendaChoice2Count = getVoteInfo.Agendas[0].Choices[1].Count
	templateInformation.AgendaChoice2IsIgnore = getVoteInfo.Agendas[0].Choices[1].IsIgnore
	templateInformation.AgendaChoice2Bits = getVoteInfo.Agendas[0].Choices[1].Bits
	templateInformation.AgendaChoice2Progress = toFixed(getVoteInfo.Agendas[0].Choices[1].Progress*100, 2)
	templateInformation.AgendaChoice3Id = getVoteInfo.Agendas[0].Choices[2].Id
	templateInformation.AgendaChoice3Description = getVoteInfo.Agendas[0].Choices[2].Description
	templateInformation.AgendaChoice3Count = getVoteInfo.Agendas[0].Choices[2].Count
	templateInformation.AgendaChoice3IsIgnore = getVoteInfo.Agendas[0].Choices[2].IsIgnore
	templateInformation.AgendaChoice3Bits = getVoteInfo.Agendas[0].Choices[2].Bits
	templateInformation.AgendaChoice3Progress = toFixed(getVoteInfo.Agendas[0].Choices[2].Progress*100, 2)

	templateInformation.VotingStarted = getVoteInfo.Agendas[0].Status == "started"
	templateInformation.VotingDefined = getVoteInfo.Agendas[0].Status == "defined"
	templateInformation.VotingLockedin = getVoteInfo.Agendas[0].Status == "lockedin"
	templateInformation.VotingFailed = getVoteInfo.Agendas[0].Status == "failed"
	templateInformation.VotingActive = getVoteInfo.Agendas[0].Status == "active"
	templateInformation.QuorumAchieved = getVoteInfo.Agendas[0].QuorumProgress == 1

	choiceIds := make([]string, len(getVoteInfo.Agendas[0].Choices))
	choicePercentages := make([]float64, len(getVoteInfo.Agendas[0].Choices))
	for i, choice := range getVoteInfo.Agendas[0].Choices {
		if choice.IsIgnore {

		} else {
			choiceIds[i] = choice.Id
			choicePercentages[i] = toFixed(choice.Progress*100, 2)
		}
	}
	templateInformation.ChoiceIds = choiceIds
	templateInformation.ChoicePercentages = choicePercentages

}

func main() {
	os.Exit(mainCore())
}

func mainCore() int {
	flag.Parse()

	// Chans for rpccclient notification handlers
	connectChan := make(chan int64, 100)
	quit := make(chan struct{})

	// Read in current dcrd cert
	var dcrdCerts []byte
	var err error
	if !*notls {
		dcrdCerts, err = ioutil.ReadFile(*cert)
		if err != nil {
			fmt.Printf("Failed to read dcrd cert file at %s: %s\n", *cert,
				err.Error())
			os.Exit(1)
		}
	}

	// Set up notification handler that will release ntfns when new blocks connect
	ntfnHandlersDaemon := dcrrpcclient.NotificationHandlers{
		OnBlockConnected: func(serializedBlockHeader []byte, transactions [][]byte) {
			var blockHeader wire.BlockHeader
			errLocal := blockHeader.Deserialize(bytes.NewReader(serializedBlockHeader))
			if errLocal != nil {
				fmt.Printf("Failed to deserialize block header: %v\n", errLocal.Error())
				return
			}
			fmt.Println("got a new block passing it", blockHeader.Height)
			connectChan <- int64(blockHeader.Height)
		},
	}

	// dcrrpclient configuration
	connCfgDaemon := &dcrrpcclient.ConnConfig{
		Host:         *host,
		Endpoint:     "ws",
		User:         *user,
		Pass:         *pass,
		Certificates: dcrdCerts,
		DisableTLS:   *notls,
	}

	fmt.Printf("Attempting to connect to dcrd RPC %s as user %s "+
		"using certificate located in %s\n", *host, *user, *cert)
	// Attempt to connect rpcclient and daemon
	dcrdClient, err := dcrrpcclient.New(connCfgDaemon, &ntfnHandlersDaemon)
	if err != nil {
		fmt.Printf("Failed to start dcrd rpcclient: %s\n", err.Error())
		return 1
	}
	defer func() {
		fmt.Printf("Disconnecting from dcrd.\n")
		dcrdClient.Disconnect()
	}()

	// Subscribe to block notifications
	if err = dcrdClient.NotifyBlocks(); err != nil {
		fmt.Printf("Failed to start register daemon rpc client for  "+
			"block notifications: %s\n", err.Error())
		return 1
	}

	// Only accept a single CTRL+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Start waiting for the interrupt signal
	go func() {
		<-c
		signal.Stop(c)
		// Close the channel so multiple goroutines can get the message
		fmt.Println("CTRL+C hit.  Closing.")
		close(quit)
		return
	}()

	// Run an initial templateInforation update based on current change
	updatetemplateInformation(dcrdClient)

	// Run goroutine for notifications
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case height := <-connectChan:
				fmt.Printf("Block height %v connected\n", height)
				updatetemplateInformation(dcrdClient)
			case <-quit:
				fmt.Printf("Closing hardfork demo.\n")
				wg.Done()
				return
			}
		}
	}()

	// Various url handlers for js/css/fonts/images
	http.HandleFunc("/", demoPage)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("public/js/"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("public/css/"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("public/fonts/"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("public/images/"))))

	// Start http server listening and serving, but no way to signal to quit
	go func() {
		err = http.ListenAndServe(*listenPort, nil)
		if err != nil {
			fmt.Printf("Failed to bind http server: %s\n", err.Error())
			close(quit)
		}
	}()

	wg.Wait()

	return 0
}

// Some various helper math helper funcs
func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
