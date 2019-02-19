// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/dcrutil"
	flags "github.com/jessevdk/go-flags"
)

const (
	defaultConfigFilename = "hardforkdemo.conf"
)

var (
	// Default network parameters
	activeNetParams *chaincfg.Params
	// stakeVersion is the stake version we call getvoteinfo with.
	blockVersion int32

	// Default configuration options
	defaultConfigFile  = filepath.Join(defaultHomeDir, defaultConfigFilename)
	defaultHomeDir     = dcrutil.AppDataDir("hardforkdemo", false)
	defaultRPCCertFile = filepath.Join(defaultHomeDir, "rpc.cert")
	defaultListenPort  = "8000"
)

// config defines the configuration options for hardforkdemo.
//
// See loadConfig for details on the configuration load process.
type config struct {
	Listen     string `short:"l" long:"listen" description:"Listen on [host]:port"`
	TestNet    bool   `long:"testnet" description:"Use the test network"`
	RPCHost    string `short:"c" long:"rpchost" description:"Hostname/IP and port of dcrd RPC server to connect to"`
	RPCUser    string `short:"u" long:"rpcuser" description:"Username for RPC connections"`
	RPCPass    string `short:"P" long:"rpcpass" default-mask:"-" description:"Password for RPC connections"`
	RPCCert    string `long:"rpccert" description:"File containing the dcrd certificate file"`
	DisableTLS bool   `long:"notls" description:"Disable TLS on the RPC client"`
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(defaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// normalizeAddress returns addr with the passed default port appended if
// there is not already a port specified.
func normalizeAddress(addr, defaultPort string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.JoinHostPort(addr, defaultPort)
	}
	return addr
}

func loadConfig() (*config, error) {
	err := os.MkdirAll(defaultHomeDir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		str := "failed to create home directory: %v"
		err := fmt.Errorf(str, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	// Default config.
	cfg := config{
		Listen:  net.JoinHostPort("localhost", defaultListenPort),
		RPCCert: defaultRPCCertFile,
	}

	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.Default)
	_, err = preParser.Parse()
	if err != nil {
		e, ok := err.(*flags.Error)
		if ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		preParser.WriteHelp(os.Stderr)
		return nil, err
	}

	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)

	// Load additional config from file.
	parser := flags.NewParser(&cfg, flags.Default)
	err = flags.NewIniParser(parser).ParseFile(defaultConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
	}

	// Parse command line options again to ensure they take precedence.
	_, err = parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return nil, err
	}

	var blockExplorerURL string
	var defaultRPCPort string

	if cfg.TestNet {
		activeNetParams = &chaincfg.TestNet3Params
		blockVersion = blockVersionTest
		blockExplorerURL = "https://testnet.dcrdata.org"
		defaultRPCPort = "19109"
	} else {
		activeNetParams = &chaincfg.MainNetParams
		blockVersion = blockVersionMain
		blockExplorerURL = "https://mainnet.dcrdata.org"
		defaultRPCPort = "9109"
	}

	cfg.Listen = normalizeAddress(cfg.Listen, defaultListenPort)
	cfg.RPCHost = normalizeAddress(cfg.RPCHost, defaultRPCPort)

	cfg.RPCCert = cleanAndExpandPath(cfg.RPCCert)

	if cfg.RPCHost == "" {
		cfg.RPCHost = net.JoinHostPort("localhost", defaultRPCPort)
	}

	if cfg.RPCUser == "" || cfg.RPCPass == "" {
		fmt.Fprintf(os.Stderr, "Please set both rpcuser and rpcpass\n")
		os.Exit(1)
	}

	// Set all activeNetParams fields now that we know what network we are on.
	templateInformation = &templateFields{
		Network:          activeNetParams.Name,
		BlockExplorerURL: blockExplorerURL,

		// BlockVersion params
		BlockVersionRejectThreshold: int(float64(activeNetParams.BlockRejectNumRequired) /
			float64(activeNetParams.BlockUpgradeNumToCheck) * 100),
		BlockVersionWindowLength: int64(activeNetParams.BlockUpgradeNumToCheck),
		// StakeVersion params
		StakeVersionWindowLength: activeNetParams.StakeVersionInterval,
		StakeVersionThreshold: float64(activeNetParams.StakeMajorityMultiplier) /
			float64(activeNetParams.StakeMajorityDivisor) * 100,
		// RuleChange params
		QuorumThreshold: float64(activeNetParams.RuleChangeActivationQuorum) /
			float64(activeNetParams.RuleChangeActivationInterval*
				uint32(activeNetParams.TicketsPerBlock)) * 100,
		RuleChangeActivationInterval: int64(activeNetParams.RuleChangeActivationInterval),
	}

	return &cfg, nil
}
