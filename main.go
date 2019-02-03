// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/decred/dcrd/rpcclient"
	"github.com/decred/dcrd/wire"
)

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

const (
	// numberOfIntervals is the number of intervals to use when calling
	// getstakeversioninfo
	numberOfIntervals = 4

	// stakeVersionMain is the version of the block being generated for
	// the main network.
	stakeVersionMain = 6

	// stakeVersionTest is the version of the block being generated
	// for the testnet network.
	stakeVersionTest = 7
)

var (
	// stakeVersion is the stake version we call getvoteinfo with.
	stakeVersion uint32 = stakeVersionMain

	// latestBlockHeader is the latest block header.
	latestBlockHeader *wire.BlockHeader

	// templateInformation is the template holding the active network
	// parameters.
	templateInformation *templateFields

	friendlyAgendaLabels = map[string]string{
		"sdiffalgorithm": "Change PoS Staking Algorithm",
		"lnsupport":      "Start Lightning Network Support",
		"lnfeatures":     "Enable Lightning Network Features",
		"fixlnseqlocks":  "Update Sequence Lock Rules"}

	longAgendaDescriptions = map[string]template.HTML{
		"sdiffalgorithm": template.HTML("Specifies a proposed replacement algorithm for determining the stake difficulty (commonly called the ticket price). This proposal resolves all issues with a new algorithm that adheres to the referenced ideals."),
		"lnsupport":      template.HTML("The <a href='https://lightning.network/' target='_blank' rel='noopener noreferrer'>Lightning Network</a> is the most directly useful application of smart contracts to date since it allows for off-chain transactions that optionally settle on-chain. This infrastructure has clear benefits for both scaling and privacy. Decred is optimally positioned for this integration."),
		"lnfeatures":     template.HTML("The <a href='https://lightning.network/' target='_blank' rel='noopener noreferrer'>Lightning Network</a> is the most directly useful application of smart contracts to date since it allows for off-chain transactions that optionally settle on-chain. This infrastructure has clear benefits for both scaling and privacy. Decred is optimally positioned for this integration."),
		"fixlnseqlocks":  template.HTML("In order to fully support the <a href='https://lightning.network/' target='_blank' rel='Noopener noreferrer'>Lightning Network</a>, the current sequence lock consensus rules need to be modified.")}
)

// updatetemplateInformation is called on startup and upon every block connected notification received.
func updatetemplateInformation(dcrdClient *rpcclient.Client) {
	log.Println("updating hard fork information")

	hash := latestBlockHeader.BlockHash()
	height := latestBlockHeader.Height

	// Set Current block height
	templateInformation.BlockHeight = height
	templateInformation.BlockExplorerLink = fmt.Sprintf("https://%s.dcrdata.org/block/%v",
		activeNetParams.Name, hash)

	templateInformation.FriendlyAgendaLabels = friendlyAgendaLabels
	templateInformation.LongAgendaDescriptions = longAgendaDescriptions

	// Request GetStakeVersions to receive information about past block versions.
	//
	// Request twice as many, so we can populate the rolling block version window's first
	stakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(activeNetParams.BlockUpgradeNumToCheck*2))
	if err != nil {
		log.Println(err)
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
		blockVersionsCounts[sv.BlockVersion]++
	}
	var minBlockVersion, maxBlockVersion int32 = math.MaxInt32, -1
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
		}
	}

	blockWinUpgradePct := func(count int64) float64 {
		return 100 * float64(count) / float64(activeNetParams.BlockUpgradeNumToCheck)
	}

	templateInformation.BlockVersionCurrent = int32(stakeVersion) - 1

	templateInformation.BlockVersionNext = minBlockVersion + 1
	templateInformation.BlockVersionNextPercentage = toFixed(blockWinUpgradePct(blockVersionsCounts[minBlockVersion+1]), 2)

	if popBlockVersionCount > int64(activeNetParams.BlockRejectNumRequired) {
		templateInformation.BlockVersionSuccess = true
	}

	// Voting intervals ((height-4096) mod 2016)
	blocksIntoStakeVersionInterval := (int64(height) - activeNetParams.StakeValidationHeight) %
		activeNetParams.StakeVersionInterval
	// Stake versions per block in current voting interval (getstakeversions hash blocksIntoInterval)
	intervalStakeVersions, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(blocksIntoStakeVersionInterval))
	if err != nil {
		log.Println(err)
		return
	}
	// Tally missed votes so far in this interval
	missedVotesStakeInterval := 0
	for _, stakeVersionResult := range intervalStakeVersions.StakeVersions {
		missedVotesStakeInterval += int(activeNetParams.TicketsPerBlock) - len(stakeVersionResult.Votes)
	}

	// Vote tallies for previous intervals
	stakeVersionInfo, err := dcrdClient.GetStakeVersionInfo(numberOfIntervals)
	if err != nil {
		log.Println(err)
		return
	}
	numIntervals := len(stakeVersionInfo.Intervals)
	if numIntervals == 0 {
		log.Println("StakeVersion info did not return usable information, intervals empty")
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
	templateInformation.StakeVersionTimeRemaining = fmt.Sprintf("%s remaining", fmtDuration(timeLeftDuration))
	stakeVersionLabels[numIntervals-1] = "Current Interval"
	currentInterval := stakeVersionInfo.Intervals[0]

	maxPossibleVotes := activeNetParams.StakeVersionInterval*int64(activeNetParams.TicketsPerBlock) -
		int64(missedVotesStakeInterval)
	templateInformation.StakeVersionIntervalResults = stakeVersionIntervalResults
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

	templateInformation.StakeVersionMostPopularPercentage = toFixed(float64(mostPopularVersionCount)/float64(maxPossibleVotes)*100, 2)
	templateInformation.StakeVersionMostPopular = mostPopularVersion
	var stakeVersionRequiredVotes = int32(maxPossibleVotes) *
		activeNetParams.StakeMajorityMultiplier / activeNetParams.StakeMajorityDivisor
	if int32(mostPopularVersionCount) > stakeVersionRequiredVotes {
		templateInformation.StakeVersionSuccess = true
	}

	// Quorum/vote information
	getVoteInfo, err := dcrdClient.GetVoteInfo(stakeVersion)
	if err != nil {
		log.Printf("Get vote info error: %v", err)
		templateInformation.Quorum = false
		return
	}
	templateInformation.GetVoteInfoResult = getVoteInfo
	templateInformation.TimeLeftString = blocksToTimeEstimate(int(getVoteInfo.EndHeight - getVoteInfo.CurrentHeight))
	// Check if Phase Upgrading or Voting
	if templateInformation.StakeVersionSuccess && templateInformation.BlockVersionSuccess {
		templateInformation.IsUpgrading = false
	} else {
		templateInformation.IsUpgrading = true
	}

	// Assume all agendas have been voted and are pending activation
	templateInformation.PendingActivation = true

	templateInformation.RulesActivated = true
	// There may be no agendas for this vote version
	if len(getVoteInfo.Agendas) == 0 {
		log.Printf("No agendas for vote version %d", mostPopularVersion)
		templateInformation.Agendas = []Agenda{}
		return
	}

	// Set Quorum to true since we got a valid response back from GetVoteInfoResult (?)
	if getVoteInfo.TotalVotes >= getVoteInfo.Quorum {
		templateInformation.Quorum = true
	}

	templateInformation.Agendas = make([]Agenda, 0, len(getVoteInfo.Agendas))

	for _, agenda := range getVoteInfo.Agendas {
		log.Printf("getvoteinfo id: %#v", agenda)

		// Check to see if all agendas are pending activation
		if agenda.Status != "lockedin" {
			templateInformation.PendingActivation = false
		}
		if agenda.Status != "active" {
			log.Println(agenda.Status)
			templateInformation.RulesActivated = false
		}

		// Acting (non-abstaining) fraction of votes
		actingPct := 1.0
		choiceIds := make([]string, len(agenda.Choices))
		choicePercentages := make([]float64, len(agenda.Choices))
		for i, choice := range agenda.Choices {
			choiceIds[i] = choice.ID
			choicePercentages[i] = toFixed(choice.Progress*100, 2)
			// non-abstain pct = 1 - abstain pct

			if choice.IsAbstain && choice.Progress < 1 {
				actingPct = 1 - choice.Progress
			}

		}
		// hardcode in values until we can properly revamp the agenda db to save this information on an
		// ongoing basis.
		var blockLockedIn int64
		var blockActivated int64
		var blockForked int64

		choiceIdsActing := make([]string, 0, len(agenda.Choices)-1)
		choicePercentagesActing := make([]float64, 0, len(agenda.Choices)-1)

		for _, choice := range agenda.Choices {
			if !choice.IsAbstain {
				choiceIdsActing = append(choiceIdsActing, choice.ID)
				/*
					if agenda.ID == "lnsupport" {
						if choice.Id == "yes" {
							choicePercentagesActing = append(choicePercentagesActing,
								98.61)
						} else if choice.Id == "no" {
							choicePercentagesActing = append(choicePercentagesActing,
								1.38)
						}
						blockLockedIn = 141184
						blockActivated = 149248
					} else if agenda.ID == "sdiffalgorithm" {
						if choice.Id == "yes" {
							choicePercentagesActing = append(choicePercentagesActing,
								97.92)
						} else if choice.Id == "no" {
							choicePercentagesActing = append(choicePercentagesActing,
								2.07)
						}
						blockLockedIn = 141184
						blockActivated = 149248
						blockForked = 149328
					} else {
				*/
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
			Agenda:                    agenda,
			QuorumExpirationDate:      time.Unix(int64(agenda.ExpireTime), int64(0)).Format(time.RFC850),
			QuorumVotedPercentage:     toFixed(agenda.QuorumProgress*100, 2),
			QuorumAbstainedPercentage: toFixed(agenda.Choices[0].Progress*100, 2),
			ChoiceIDs:                 choiceIds,
			ChoicePercentages:         choicePercentages,
			ChoiceIDsActing:           choiceIdsActing,
			ChoicePercentagesActing:   choicePercentagesActing,
			StartHeight:               getVoteInfo.StartHeight,
			VoteCountPercentage:       toFixed(voteCountPercentage*100, 1),
			BlockLockedIn:             blockLockedIn,
			BlockForked:               blockForked,
			BlockActivated:            blockActivated,
		})
	}
}

// main wraps mainCore, which does all the work, because deferred functions do
/// not run after os.Exit().
func main() {
	os.Exit(mainCore())
}

func mainCore() int {
	cfg, err := loadConfig()
	if err != nil {
		return 1
	}

	// Chans for rpccclient notification handlers
	connectChan := make(chan wire.BlockHeader, 100)
	quit := make(chan struct{})

	// Read in current dcrd cert
	var dcrdCerts []byte
	if !cfg.DisableTLS {
		dcrdCerts, err = ioutil.ReadFile(cfg.RPCCert)
		if err != nil {
			log.Printf("Failed to read dcrd cert file at %v: %v",
				cfg.RPCCert, err)
			return 1
		}
	}

	// Set up notification handler that will release ntfns when new blocks connect
	ntfnHandlersDaemon := rpcclient.NotificationHandlers{
		OnBlockConnected: func(serializedBlockHeader []byte, _ [][]byte) {
			var blockHeader wire.BlockHeader
			errLocal := blockHeader.FromBytes(serializedBlockHeader)
			if errLocal != nil {
				log.Printf("Failed to deserialize block header: %v", errLocal)
				return
			}
			log.Printf("received new block %v (height %d)", blockHeader.BlockHash(),
				blockHeader.Height)
			connectChan <- blockHeader
		},
	}

	// rpclient configuration
	connCfgDaemon := &rpcclient.ConnConfig{
		Host:         cfg.RPCHost,
		Endpoint:     "ws",
		User:         cfg.RPCUser,
		Pass:         cfg.RPCPass,
		Certificates: dcrdCerts,
		DisableTLS:   cfg.DisableTLS,
	}

	log.Printf("Attempting to connect to dcrd RPC %s as user %s "+
		"using certificate %s", cfg.RPCHost, cfg.RPCUser, cfg.RPCCert)
	// Attempt to connect rpcclient and daemon
	dcrdClient, err := rpcclient.New(connCfgDaemon, &ntfnHandlersDaemon)
	if err != nil {
		log.Printf("Failed to start dcrd rpcclient: %v", err)
		return 1
	}
	defer func() {
		log.Printf("Disconnecting from dcrd.")
		dcrdClient.Disconnect()
	}()

	// Subscribe to block notifications
	if err = dcrdClient.NotifyBlocks(); err != nil {
		log.Printf("Failed to start register daemon rpc client for  "+
			"block notifications: %v\n", err)
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
		log.Println("CTRL+C hit.  Closing.")
		close(quit)
	}()

	// Get the current best block (height and hash)
	hash, err := dcrdClient.GetBestBlockHash()
	if err != nil {
		log.Println(err)
		return 1
	}
	// Request the current block header
	latestBlockHeader, err = dcrdClient.GetBlockHeader(hash)
	if err != nil {
		log.Println(err)
		return 1
	}

	// Run an initial templateInforation update based on current change
	updatetemplateInformation(dcrdClient)

	// Run goroutine for notifications
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case blkHdr := <-connectChan:
				latestBlockHeader = &blkHdr
				log.Printf("Block %v (height %v) connected",
					blkHdr.BlockHash(), blkHdr.Height)
				updatetemplateInformation(dcrdClient)
			case <-quit:
				log.Println("Closing hardfork demo.")
				wg.Done()
				return
			}
		}
	}()

	// Create new web UI to deal with HTML templates and provide the
	// http.HandleFunc for the web server
	webUI, err := NewWebUI()
	if err != nil {
		log.Printf("NewWebUI failed: %v", err)
		os.Exit(1)
	}
	webUI.TemplateData = templateInformation
	// Register OS signal (USR1 on non-Windows platforms) to reload templates
	webUI.UseSIGToReloadTemplates()

	// URL handlers for js/css/fonts/images
	http.HandleFunc("/", webUI.homePage)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("public/js/"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("public/css/"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("public/fonts/"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("public/images/"))))

	// Start http server listening and serving, but no way to signal to quit
	go func() {
		log.Printf("Starting webserver on %v", cfg.Listen)
		err = http.ListenAndServe(cfg.Listen, nil)
		if err != nil {
			log.Printf("Failed to bind http server: %v", err)
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

// fmtDuration will convert a Duration into a human readable string formatted "0d 0h 0m".
func fmtDuration(dur time.Duration) string {
	dur = dur.Round(time.Minute)
	days := dur / (time.Hour * 24)
	dur -= days * (time.Hour * 24)
	hours := dur / time.Hour
	dur -= hours * time.Hour
	mins := dur / time.Minute
	return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
}
