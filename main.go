// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
)

type hardForkInfo struct {
	BlockHeight              int64
	BlockVersionWindowValues map[uint64]*blockVersions
}

type blockVersions struct {
	VersionPercentages map[int32]int64
}

var activeNetParams = &chaincfg.TestNetParams

var hardForkInformation = &hardForkInfo{}

func demoPage(w http.ResponseWriter, r *http.Request) {

	fp := filepath.Join("public/views", "design_sketch.html")
	tmpl, err := template.New("home").ParseFiles(fp)
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
	}
	stakeVersionResults, err := dcrdClient.GetStakeVersions(hash.String(), int32(activeNetParams.BlockUpgradeNumToCheck*2))
	if err != nil {
		fmt.Println(err)
	}
	blockVersionLookBack := make(map[uint64]*blockVersions, activeNetParams.BlockUpgradeNumToCheck)

	for i := uint64(0); i < activeNetParams.BlockUpgradeNumToCheck; i++ {
		currentBlockVersions := &blockVersions{}
		currentBlockVersionPercentages := make(map[int32]int64)
		for j := i; j < activeNetParams.BlockUpgradeNumToCheck+i; j++ {
			currentBlockVersionPercentages[stakeVersionResults.StakeVersions[i].BlockVersion] += currentBlockVersionPercentages[stakeVersionResults.StakeVersions[i].BlockVersion] + int64(1)
			//fmt.Println(i, j)
		}
		currentBlockVersions.VersionPercentages = currentBlockVersionPercentages
		blockVersionLookBack[i] = currentBlockVersions
	}
	hardForkInformation.BlockVersionWindowValues = blockVersionLookBack
	hardForkInformation.BlockHeight = height
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
	dcrdCertPath := ("/home/user/.dcrd/rpc.cert")
	dcrdCerts, err := ioutil.ReadFile(dcrdCertPath)
	if err != nil {
		fmt.Printf("Failed to read dcrd cert file at %s: %s\n", dcrdCertPath,
			err.Error())
		os.Exit(1)
	}
	dcrdServer := "127.0.0.1:19109"
	dcrdUser := "USER"
	dcrdPass := "PASSWORD"
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
	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Printf("Failed to bind http server: %s\n", err.Error())
	}
}
