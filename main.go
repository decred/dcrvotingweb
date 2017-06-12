// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
	"github.com/decred/hardforkdemo/agendadb"
)

// Settings for daemon
var network = flag.String("network", "mainnet", "current network being used")
var host = flag.String("host", "127.0.0.1:9109", "node RPC host:port")
var user = flag.String("user", "USER", "node RPC username")
var pass = flag.String("pass", "PASSWORD", "node RPC password")
var cert = flag.String("cert", "/home/user/.dcrd/rpc.cert", "node RPC TLS certificate (when notls=false)")
var notls = flag.Bool("notls", false, "Disable use of TLS for node connection")
var listenPort = flag.String("listen", ":8000", "web app listening port")

// Daemon Params to use
var activeNetParams = &chaincfg.MainNetParams

// Latest BlockHeader
var latestBlockHeader *wire.BlockHeader

// Contains a certain block version's count of blocks in the
// rolling window (which has a length of activeNetParams.BlockUpgradeNumToCheck)
type blockVersions struct {
	RollingWindowLookBacks []int
}

type intervalVersionCounts struct {
	Version        uint32
	Count          []uint32
	StaticInterval string
}

// Set all activeNetParams fields since they don't change at runtime
var templateInformation = &templateFields{
	Network: *network,
	// BlockVersion params
	BlockVersionEnforceThreshold: int(float64(activeNetParams.BlockEnforceNumRequired) /
		float64(activeNetParams.BlockUpgradeNumToCheck) * 100),
	BlockVersionRejectThreshold: int(float64(activeNetParams.BlockRejectNumRequired) /
		float64(activeNetParams.BlockUpgradeNumToCheck) * 100),
	BlockVersionWindowLength: activeNetParams.BlockUpgradeNumToCheck,
	// StakeVersion params
	StakeVersionWindowLength: activeNetParams.StakeVersionInterval,
	StakeVersionThreshold: toFixed(float64(activeNetParams.StakeMajorityMultiplier)/
		float64(activeNetParams.StakeMajorityDivisor)*100, 0),
	// RuleChange params
	RuleChangeActivationQuorum: activeNetParams.RuleChangeActivationQuorum,
	QuorumThreshold: float64(activeNetParams.RuleChangeActivationQuorum) /
		float64(activeNetParams.RuleChangeActivationInterval*uint32(activeNetParams.TicketsPerBlock)) * 100,
	RuleChangeActivationInterval: int64(activeNetParams.RuleChangeActivationInterval),
}

// updatetemplateInformation is called on startup and upon every block connected notification received.
func updatetemplateInformation(dcrdClient *dcrrpcclient.Client, db *agendadb.AgendaDB) {
	fmt.Println("updating hard fork information")

	if latestBlockHeader == nil {
		// Get the current best block (height and hash)
		hash, err := dcrdClient.GetBestBlockHash()
		if err != nil {
			fmt.Println(err)
			return
		}
		// Request the current block header
		latestBlockHeader, err = dcrdClient.GetBlockHeader(hash)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	hash := latestBlockHeader.BlockHash()
	height := latestBlockHeader.Height

	// Set Current block height
	templateInformation.BlockHeight = height
	templateInformation.BlockExplorerLink = fmt.Sprintf("https://%s.decred.org/block/%v",
		*network, hash)

	// Request GetStakeVersions to receive information about past block versions.
	//
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

	// The algorithm starts at the middle of the GetStakeVersionResults and decrements backwards toward
	// the beginning of the list.  This is due to GetStakeVersionResults.StakeVersions being ordered
	// from most recent blocks to oldest. (ie [0] == current, [len] == oldest).  So by starting in the middle
	// we then can calculate that first blocks rolling window result then become one block 'more recent'
	// and calculate that blocks rolling window results.
	for i := len(stakeVersionResults.StakeVersions)/2 - 1; i >= 0; i-- {
		// Calculate the last block element in the window
		windowEnd := i + int(activeNetParams.BlockUpgradeNumToCheck)
		// blockVersionsHeights lets us have a correctly ordered list of blockheights for xaxis label
		blockVersionsHeights[elementNum] = stakeVersionResults.StakeVersions[i].Height
		// Define rolling window range for this current block (i)
		stakeVersionsWindow := stakeVersionResults.StakeVersions[i:windowEnd]
		for _, stakeVersion := range stakeVersionsWindow {
			// Try to get an existing blockVersions struct (pointer)
			theseBlockVersions, ok := blockVersionsFound[stakeVersion.BlockVersion]
			if !ok {
				// Had not found this block version yet
				theseBlockVersions = &blockVersions{}
				blockVersionsFound[stakeVersion.BlockVersion] = theseBlockVersions
				theseBlockVersions.RollingWindowLookBacks =
					make([]int, activeNetParams.BlockUpgradeNumToCheck)
				// Need to populate "back" to fill in values for previously missed window
				for k := 0; k < elementNum; k++ {
					theseBlockVersions.RollingWindowLookBacks[k] = 0
				}
				theseBlockVersions.RollingWindowLookBacks[elementNum] = 1
			} else {
				// Already had that block version, so increment
				theseBlockVersions.RollingWindowLookBacks[elementNum]++
			}
		}
		elementNum++
	}
	templateInformation.BlockVersionsHeights = blockVersionsHeights
	templateInformation.BlockVersions = blockVersionsFound

	// Pick min block version (current version) out of most recent window
	stakeVersionsWindow := stakeVersionResults.StakeVersions[:activeNetParams.BlockUpgradeNumToCheck]
	blockVersionsCounts := make(map[int32]int64)
	for _, sv := range stakeVersionsWindow {
		blockVersionsCounts[sv.BlockVersion] = blockVersionsCounts[sv.BlockVersion] + 1
	}
	var minBlockVersion, maxBlockVersion, popBlockVersion int32 = math.MaxInt32, -1, 0
	for v := range blockVersionsCounts {
		if v < minBlockVersion {
			minBlockVersion = v
		}
		if v > maxBlockVersion {
			maxBlockVersion = v
		}
	}
	popBlockVersionCount := int64(-1)
	for v, c := range blockVersionsCounts {
		if c > popBlockVersionCount && v != minBlockVersion {
			popBlockVersionCount = c
			popBlockVersion = v
		}
	}

	blockWinUpgradePct := func(count int64) float64 {
		return 100 * float64(count) / float64(activeNetParams.BlockUpgradeNumToCheck)
	}

	templateInformation.BlockVersionCurrent = 4

	templateInformation.BlockVersionMostPopular = popBlockVersion
	templateInformation.BlockVersionMostPopularPercentage = toFixed(blockWinUpgradePct(popBlockVersionCount), 2)

	templateInformation.BlockVersionNext = minBlockVersion + 1
	templateInformation.BlockVersionNextPercentage = toFixed(blockWinUpgradePct(blockVersionsCounts[minBlockVersion+1]), 2)

	//if popBlockVersionCount > int64(activeNetParams.BlockRejectNumRequired) {
	templateInformation.BlockVersionSuccess = true
	//}

	// Voting intervals ((height-4096) mod 2016)
	blocksIntoStakeVersionInterval := (int64(height) - activeNetParams.StakeValidationHeight) %
		activeNetParams.StakeVersionInterval
	// Stake versions per block in current voting interval (getstakeversions hash blocksIntoInterval)
	intervalStakeVersions, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(blocksIntoStakeVersionInterval))
	if err != nil {
		fmt.Println(err)
		return
	}
	// Tally missed votes so far in this interval
	missedVotesStakeInterval := 0
	for _, stakeVersionResult := range intervalStakeVersions.StakeVersions {
		missedVotesStakeInterval += int(activeNetParams.TicketsPerBlock) - len(stakeVersionResult.Votes)
	}

	// Vote tallies for previous intervals (getstakeversioninfo 4)
	numberOfIntervalsToCheck := 4
	stakeVersionInfo, err := dcrdClient.GetStakeVersionInfo(int32(numberOfIntervalsToCheck))
	if err != nil {
		fmt.Println(err)
		return
	}
	numIntervals := len(stakeVersionInfo.Intervals)
	if numIntervals == 0 {
		fmt.Println("StakeVersion info did not return usable information, intervals empty")
		return
	}
	templateInformation.StakeVersionsIntervals = stakeVersionInfo.Intervals

	minimumNeededVoteVersions := uint32(100)
	// Hacky way of populating the Vote Version bar graph
	// Each element in each dataset needs counts for each interval
	// For example:
	// version 1: [100, 200, 0, 400]
	var stakeVersionIntervalEndHeight = int64(0)
	var stakeVersionIntervalResults []intervalVersionCounts
	stakeVersionLabels := make([]string, numIntervals)
	// Oldest to newest interval (charts are left to right)
	for i := 0; i < numIntervals; i++ {
		interval := &stakeVersionInfo.Intervals[numIntervals-1-i]
		stakeVersionLabels[i] = fmt.Sprintf("%v - %v", interval.StartHeight, interval.EndHeight-1)
		if i == numIntervals-1 {
			stakeVersionIntervalEndHeight = interval.StartHeight + activeNetParams.StakeVersionInterval - 1
			templateInformation.StakeVersionIntervalBlocks = fmt.Sprintf("%v - %v", interval.StartHeight, stakeVersionIntervalEndHeight)
		}
	versionloop:
		for _, versionCount := range interval.VoteVersions {
			// Is this a vote version we've seen in a previous interval?
			for k, result := range stakeVersionIntervalResults {
				if result.Version == versionCount.Version {
					stakeVersionIntervalResults[k].Count[i] = versionCount.Count
					continue versionloop
				}
			}
			if versionCount.Count > minimumNeededVoteVersions {
				stakeVersionIntervalResult := intervalVersionCounts{
					Version: versionCount.Version,
					Count:   make([]uint32, numIntervals),
				}
				stakeVersionIntervalResult.Count[i] = versionCount.Count
				stakeVersionIntervalResults = append(stakeVersionIntervalResults, stakeVersionIntervalResult)
			}
		}
	}
	blocksRemainingStakeInterval := stakeVersionIntervalEndHeight - int64(height)
	timeLeftDuration := activeNetParams.TargetTimePerBlock * time.Duration(blocksRemainingStakeInterval)
	templateInformation.StakeVersionTimeRemaining = fmt.Sprintf("%s remaining", timeLeftDuration.String())
	stakeVersionLabels[numIntervals-1] = "Current Interval"
	currentInterval := stakeVersionInfo.Intervals[0]

	maxPossibleVotes := activeNetParams.StakeVersionInterval*int64(activeNetParams.TicketsPerBlock) -
		int64(missedVotesStakeInterval)
	templateInformation.StakeVersionIntervalResults = stakeVersionIntervalResults
	templateInformation.StakeVersionWindowVoteTotal = maxPossibleVotes
	templateInformation.StakeVersionIntervalLabels = stakeVersionLabels
	templateInformation.StakeVersionCurrent = latestBlockHeader.StakeVersion

	var mostPopularVersion, mostPopularVersionCount uint32
	for _, stakeVersion := range currentInterval.VoteVersions {
		if stakeVersion.Version > latestBlockHeader.StakeVersion &&
			stakeVersion.Count > mostPopularVersionCount {
			mostPopularVersion = stakeVersion.Version
			mostPopularVersionCount = stakeVersion.Count
		}
	}

	templateInformation.StakeVersionMostPopularCount = mostPopularVersionCount
	templateInformation.StakeVersionMostPopularPercentage = toFixed(float64(100), 2) //float64(mostPopularVersionCount)/float64(maxPossibleVotes)*100, 2)
	templateInformation.StakeVersionMostPopular = 4                                  //mostPopularVersion
	templateInformation.StakeVersionRequiredVotes = int32(maxPossibleVotes) *
		activeNetParams.StakeMajorityMultiplier / activeNetParams.StakeMajorityDivisor
	//if int32(mostPopularVersionCount) > templateInformation.StakeVersionRequiredVotes {
	templateInformation.StakeVersionSuccess = true
	//}

	blocksIntoInterval := currentInterval.EndHeight - currentInterval.StartHeight
	templateInformation.StakeVersionVotesRemaining =
		(activeNetParams.StakeVersionInterval - blocksIntoInterval) * int64(activeNetParams.TicketsPerBlock)

	// Quorum/vote information
	getVoteInfo, err := dcrdClient.GetVoteInfo(latestBlockHeader.StakeVersion)
	if err != nil {
		fmt.Println("Get vote info err", err)
		templateInformation.Quorum = false
		return
	}
	templateInformation.GetVoteInfoResult = getVoteInfo

	// Check if Phase Upgrading or Voting
	if templateInformation.StakeVersionSuccess && templateInformation.BlockVersionSuccess {
		templateInformation.IsUpgrading = false
	} else {
		templateInformation.IsUpgrading = true
	}

	// There may be no agendas for this vote version
	if len(getVoteInfo.Agendas) == 0 {
		fmt.Printf("No agendas for vote version %d\n", mostPopularVersion)
		templateInformation.Agendas = []Agenda{}
		return
	}

	// Set Quorum to true since we got a valid response back from GetVoteInfoResult (?)
	if getVoteInfo.TotalVotes >= getVoteInfo.Quorum {
		templateInformation.Quorum = true
	}

	// Status LockedIn Circle3 Ring Indicates BlocksLeft until old versions gets denied
	lockedinBlocksleft := float64(getVoteInfo.EndHeight) - float64(getVoteInfo.CurrentHeight)
	lockedinWindowsize := float64(getVoteInfo.EndHeight) - float64(getVoteInfo.StartHeight)
	lockedinPercentage := lockedinWindowsize / 100

	templateInformation.LockedinPercentage = toFixed(lockedinBlocksleft/lockedinPercentage, 2)
	templateInformation.Agendas = make([]Agenda, 0, len(getVoteInfo.Agendas))

	for i := range getVoteInfo.Agendas {
		// Direct conversion works with go1.8, but angers Travis b/c "Id"!="ID"
		//agenda := agendadb.AgendaTagged(getVoteInfo.Agendas[i])
		agenda := agendadb.FromDcrJSONAgenda(&getVoteInfo.Agendas[i])
		if err = db.StoreAgenda(agenda); err != nil {
			fmt.Printf("Failed to store agenda %s: %v\n", agenda.ID, err)
		}

		// Acting (non-abstaining) fraction of votes
		actingPct := 1.0
		choiceIds := make([]string, len(agenda.Choices))
		choicePercentages := make([]float64, len(agenda.Choices))
		for i, choice := range agenda.Choices {
			choiceIds[i] = choice.Id
			choicePercentages[i] = toFixed(choice.Progress*100, 2)
			// non-abstain pct = 1 - abstain pct
			if choice.IsAbstain && choice.Progress < 1 {
				actingPct = 1 - choice.Progress
			}
		}

		choiceIdsActing := make([]string, 0, len(agenda.Choices)-1)
		choicePercentagesActing := make([]float64, 0, len(agenda.Choices)-1)
		for _, choice := range agenda.Choices {
			if !choice.IsAbstain {
				choiceIdsActing = append(choiceIdsActing, choice.Id)
				choicePercentagesActing = append(choicePercentagesActing,
					toFixed(choice.Progress/actingPct*100, 2))
			}
		}
		voteCount := uint32(0)
		for _, choice := range agenda.Choices {
			voteCount += choice.Count
		}

		voteCountPercentage := float64(voteCount) / (float64(activeNetParams.RuleChangeActivationInterval) * float64(activeNetParams.TicketsPerBlock))

		templateInformation.Agendas = append(templateInformation.Agendas, Agenda{
			Agenda:                    *agenda.ToDcrJSONAgenda(),
			QuorumExpirationDate:      time.Unix(int64(agenda.ExpireTime), int64(0)).Format(time.RFC850),
			QuorumVotedPercentage:     toFixed(agenda.QuorumProgress*100, 2),
			QuorumAbstainedPercentage: toFixed(agenda.Choices[0].Progress*100, 2),
			ChoiceIDs:                 choiceIds,
			ChoicePercentages:         choicePercentages,
			ChoiceIDsActing:           choiceIdsActing,
			ChoicePercentagesActing:   choicePercentagesActing,
			StartHeight:               getVoteInfo.StartHeight,
			VoteCountPercentage:       toFixed(voteCountPercentage*100, 1),
		})
	}
}

// main wraps mainCore, which does all the work, because deferred functions do
/// not run after os.Exit().
func main() {
	os.Exit(mainCore())
}

func mainCore() int {
	flag.Parse()

	// Chans for rpccclient notification handlers
	connectChan := make(chan wire.BlockHeader, 100)
	quit := make(chan struct{})

	// Read in current dcrd cert
	var dcrdCerts []byte
	var err error
	if !*notls {
		dcrdCerts, err = ioutil.ReadFile(*cert)
		if err != nil {
			fmt.Printf("Failed to read dcrd cert file at %s: %s\n", *cert,
				err.Error())
			return 1
		}
	}

	// Set up notification handler that will release ntfns when new blocks connect
	ntfnHandlersDaemon := dcrrpcclient.NotificationHandlers{
		OnBlockConnected: func(serializedBlockHeader []byte, transactions [][]byte) {
			var blockHeader wire.BlockHeader
			errLocal := blockHeader.Deserialize(bytes.NewReader(serializedBlockHeader))
			if errLocal != nil {
				fmt.Printf("Failed to deserialize block header: %v\n", errLocal)
				return
			}
			fmt.Printf("received new block %v (height %d)\n", blockHeader.BlockHash(),
				blockHeader.Height)
			connectChan <- blockHeader
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
		fmt.Printf("Failed to start dcrd rpcclient: %v\n", err)
		return 1
	}
	defer func() {
		fmt.Printf("Disconnecting from dcrd.\n")
		dcrdClient.Disconnect()
	}()

	// Subscribe to block notifications
	if err = dcrdClient.NotifyBlocks(); err != nil {
		fmt.Printf("Failed to start register daemon rpc client for  "+
			"block notifications: %v\n", err)
		return 1
	}

	// Open DB for past agendas
	dbPath, dbName := "history", "agendas.db"
	err = os.Mkdir(dbPath, os.FileMode(750))
	if err != nil && !os.IsExist(err) {
		fmt.Printf("Unable to create database folder: %v\n", err)
	}
	db, err := agendadb.Open(filepath.Join(dbPath, dbName))
	if err != nil {
		fmt.Printf("Unable to open agendas DB: %v\n", err)
		return 1
	}
	defer db.Close()
	db.ListAgendas()

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
	updatetemplateInformation(dcrdClient, db)

	// Run goroutine for notifications
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case blkHdr := <-connectChan:
				latestBlockHeader = &blkHdr
				fmt.Printf("Block %v (height %v) connected\n",
					blkHdr.BlockHash(), blkHdr.Height)
				updatetemplateInformation(dcrdClient, db)
			case <-quit:
				fmt.Println("Closing hardfork demo.")
				wg.Done()
				return
			}
		}
	}()

	// Create new web UI to deal with HTML templates and provide the
	// http.HandleFunc for the web server
	webUI, err := NewWebUI()
	if err != nil {
		fmt.Printf("NewWebUI failed: %v\n", err)
		os.Exit(1)
	}
	webUI.TemplateData = templateInformation
	// Register OS signal (USR1 on non-Windows platforms) to reload templates
	webUI.UseSIGToReloadTemplates()

	// URL handlers for js/css/fonts/images
	http.HandleFunc("/", webUI.demoPage)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("public/js/"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("public/css/"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("public/fonts/"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("public/images/"))))

	// Start http server listening and serving, but no way to signal to quit
	go func() {
		err = http.ListenAndServe(*listenPort, nil)
		if err != nil {
			fmt.Printf("Failed to bind http server: %v\n", err)
			close(quit)
		}
	}()

	// Wait for goroutines, such as the block connected handler loop
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
