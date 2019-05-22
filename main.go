// Copyright (c) 2017-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	// blockVersionMain is the version of the block being generated for
	// the main network.
	blockVersionMain = 6

	// blockVersionTest is the version of the block being generated
	// for the testnet network.
	blockVersionTest = 7
)

var (

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
		"fixlnseqlocks":  template.HTML("In order to fully support the <a href='https://lightning.network/' target='_blank' rel='noopener noreferrer'>Lightning Network</a>, the current sequence lock consensus rules need to be modified.")}
)

// updatetemplateInformation is called on startup and upon every block connected notification received.
func updatetemplateInformation(dcrdClient *rpcclient.Client, latestBlockHeader *wire.BlockHeader) {
	log.Println("updating vote information")

	hash := latestBlockHeader.BlockHash()
	height := int64(latestBlockHeader.Height)

	log.Printf("Current best block height: %d", height)

	// Set Current block height
	templateInformation.BlockHeight = height

	templateInformation.FriendlyAgendaLabels = friendlyAgendaLabels
	templateInformation.LongAgendaDescriptions = longAgendaDescriptions

	// Request GetStakeVersions to receive information about past block versions.
	//
	// Request twice as many, so we can populate the rolling block version window's first
	stakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(activeNetParams.BlockUpgradeNumToCheck*2))
	if err != nil {
		log.Printf("GetStakeVersions error: %v", err)
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

	stakeVersionsWindow := stakeVersionResults.StakeVersions[:activeNetParams.BlockUpgradeNumToCheck]
	blockVersionsCounts := make(map[int32]int64)
	for _, sv := range stakeVersionsWindow {
		blockVersionsCounts[sv.BlockVersion]++
	}

	var mostPopularBlockVersion int32
	mostPopularBlockVersionCount := int64(0)
	for v, count := range blockVersionsCounts {
		if count > mostPopularBlockVersionCount {
			mostPopularBlockVersion = v
			mostPopularBlockVersionCount = count
		}
	}

	log.Printf("Most popular block version in the last %d blocks: v%d (%d blocks)",
		len(stakeVersionsWindow), mostPopularBlockVersion, blockVersionsCounts[mostPopularBlockVersion])

	templateInformation.BlockVersionCurrent = mostPopularBlockVersion

	templateInformation.BlockVersionNext = blockVersion

	blockCountPercentage := 100 * float64(blockVersionsCounts[blockVersion]) / float64(activeNetParams.BlockUpgradeNumToCheck)
	templateInformation.BlockVersionNextPercentage = blockCountPercentage

	if blockVersionsCounts[blockVersion] > int64(activeNetParams.BlockRejectNumRequired) {
		templateInformation.BlockVersionSuccess = true
	}

	// Voting intervals ((height-4096) mod 2016)
	blocksIntoStakeVersionInterval := (height - activeNetParams.StakeValidationHeight) %
		activeNetParams.StakeVersionInterval
	// Stake versions per block in current voting interval (getstakeversions hash blocksIntoInterval)
	intervalStakeVersions, err := dcrdClient.GetStakeVersions(hash.String(),
		int32(blocksIntoStakeVersionInterval))
	if err != nil {
		log.Printf("GetStakeVersions error: %v", err)
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
	var CurrentSVIEndHeightHeight = int64(0)
	var stakeVersionIntervalResults []intervalVersionCounts
	stakeVersionLabels := make([]string, numIntervals)
	// Oldest to newest interval (charts are left to right)
	for i := 0; i < numIntervals; i++ {
		interval := &stakeVersionInfo.Intervals[numIntervals-1-i]
		stakeVersionLabels[i] = fmt.Sprintf("%v - %v", interval.StartHeight, interval.EndHeight-1)
		if i == numIntervals-1 {
			CurrentSVIEndHeightHeight = interval.StartHeight + activeNetParams.StakeVersionInterval - 1
			templateInformation.CurrentSVIStartHeight = interval.StartHeight
			templateInformation.CurrentSVIEndHeight = CurrentSVIEndHeightHeight
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
	blocksRemainingStakeInterval := CurrentSVIEndHeightHeight - height
	timeLeftDuration := activeNetParams.TargetTimePerBlock * time.Duration(blocksRemainingStakeInterval)
	templateInformation.StakeVersionTimeRemaining = fmtDuration(timeLeftDuration)
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

	templateInformation.StakeVersionMostPopularPercentage = float64(mostPopularVersionCount) / float64(maxPossibleVotes) * 100
	templateInformation.StakeVersionMostPopular = mostPopularVersion

	svis, err := AllStakeVersionIntervals(dcrdClient, height)
	if err != nil {
		log.Printf("Error stake version intervals: %v", err)
		return
	}

	templateInformation.PosUpgrade.Completed = false

	// Check if upgrade to the latest version occurred in a previous SVI
	upgradeOccurred, svi := svis.GetStakeVersionUpgradeSVI(svis.MaxVoteVersion)
	if upgradeOccurred {
		templateInformation.StakeVersionMostPopularPercentage = 100
		templateInformation.PosUpgrade.Completed = true
		templateInformation.PosUpgrade.UpgradeInterval = svi
	}

	// Check if Phase Upgrading or Voting
	if templateInformation.PosUpgrade.Completed && templateInformation.BlockVersionSuccess {
		templateInformation.IsUpgrading = false
	} else {
		templateInformation.IsUpgrading = true
	}

	templateInformation.Agendas, err = agendasForVersions(dcrdClient, svis.MaxVoteVersion, height, svis)
	if err != nil {
		log.Printf("Error getting agendas: %v", err)
		return
	}

	// Assume all agendas have been voted and are pending activation
	templateInformation.PendingActivation = true

	templateInformation.RulesActivated = true

	for _, agenda := range templateInformation.Agendas {

		// Check to see if all agendas are pending activation
		if !agenda.IsLockedIn() {
			templateInformation.PendingActivation = false
		}
		if !agenda.IsActive() {
			templateInformation.RulesActivated = false
		}
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
	latestBlockHeader, err := dcrdClient.GetBlockHeader(hash)
	if err != nil {
		log.Println(err)
		return 1
	}

	// Run an initial templateInforation update based on current change
	updatetemplateInformation(dcrdClient, latestBlockHeader)

	// Run goroutine for notifications
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case blkHdr := <-connectChan:
				log.Printf("Block %v (height %v) connected",
					blkHdr.BlockHash(), blkHdr.Height)
				updatetemplateInformation(dcrdClient, &blkHdr)
			case <-quit:
				log.Println("Closing dcrvotingweb")
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

	noDirListing := func(h http.Handler) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/") {
				webUI.homePage(w, r)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	// URL handlers for js/css/fonts/images
	http.HandleFunc("/", webUI.homePage)
	http.Handle("/js/", noDirListing(http.StripPrefix("/js/", http.FileServer(http.Dir("public/js/")))))
	http.Handle("/css/", noDirListing(http.StripPrefix("/css/", http.FileServer(http.Dir("public/css/")))))
	http.Handle("/fonts/", noDirListing(http.StripPrefix("/fonts/", http.FileServer(http.Dir("public/fonts/")))))
	http.Handle("/images/", noDirListing(http.StripPrefix("/images/", http.FileServer(http.Dir("public/images/")))))

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
