package main

import "testing"

func TestStakeInfoStub(t *testing.T) {
	bestHeightsExpected := []int64{273167, 289984}
	bestHashesExpected := []string{"00000000062b8a5baa35ce53f274f29aeb57d77a4c15375c6f82c92ed1bea1e8",
		"0000000000b0a130a86d7fe1e475b39da048c3215d958ea4b2e102ad5a687fc0"}

	for ib, bestHeightExpected := range bestHeightsExpected {
		bestHashExpected := bestHashesExpected[ib]

		// StakeInfoStub constructor
		stub := NewStakeInfoStub(uint32(bestHeightExpected))
		if stub == nil {
			t.Error("Unable to create stub")
		}

		// GetBestBlock
		hash, height, err := stub.GetBestBlock()
		if err != nil {
			t.Error("GetBestBlock failed: ", err)
		}

		if hash.String() != bestHashExpected {
			t.Errorf("Incorrect hash.  Got %s, expecting %s",
				hash.String(), bestHashExpected)
		}

		if height != bestHeightExpected {
			t.Errorf("Incorrect height.  Got %d, expecting %d",
				height, bestHeightExpected)
		}

		// go test -v
		t.Log("Hash: ", hash)
		t.Log("Height: ", height)

		// GetBlockVerbose
		block, err := stub.GetBlockVerbose(hash, false)
		if err != nil {
			t.Errorf("Unable to get block %v: %v", hash, err)
		}

		if block.Height != bestHeightExpected {
			t.Errorf("Got block with height %d, expected %d",
				block.Height, bestHeightExpected)
		}

		// GetVoteInfo
		voteInfo, err := stub.GetVoteInfo(4)
		if err != nil {
			t.Error("GetVoteInfo(4) failed: ", err)
		}

		if len(voteInfo.Agendas) != 1 {
			t.Fatalf("Found %d agendas, expected %d", len(voteInfo.Agendas), 1)
		}

		choices := voteInfo.Agendas[0].Choices
		if len(choices) != 3 {
			t.Errorf("Found %d agenda choices, expected %d", len(choices), 3)
		}

		// GetStakeVersionInfo
		numIntervals := 144
		stakeVersionInfo, err := stub.GetStakeVersionInfo(int32(numIntervals))
		if err != nil {
			t.Errorf("GetStakeVersionInfo(%d) failed: %v", numIntervals, err)
		}

		if len(stakeVersionInfo.Intervals) != numIntervals {
			t.Fatalf("Found %d stake version intervals, expected %d",
				len(stakeVersionInfo.Intervals), numIntervals)
		}

		if stakeVersionInfo.CurrentHeight != bestHeightExpected {
			t.Errorf("best block was %d, expected %d",
				stakeVersionInfo.CurrentHeight, bestHeightExpected)
		}

		if stakeVersionInfo.Hash != bestHashExpected {
			t.Errorf("Incorrect hash.  Got %s, expecting %s",
				stakeVersionInfo.Hash, bestHashExpected)
		}

		// GetStakeVersions
		numStakeVersions := 2016
		stakeVersions, err := stub.GetStakeVersions(bestHashExpected, int32(numStakeVersions))
		if err != nil {
			t.Errorf("GetStakeVersions(%d) failed: %v", numStakeVersions, err)
		}

		if len(stakeVersions.StakeVersions) != numStakeVersions {
			t.Fatalf("Found %d stake versions, expected %d",
				len(stakeVersions.StakeVersions), numStakeVersions)
		}

		if stakeVersions.StakeVersions[0].Hash != bestHashExpected {
			t.Errorf("Incorrect hash of first block.  Got %s, expecting %s",
				stakeVersions.StakeVersions[0].Hash, bestHashExpected)
		}

		if stakeVersions.StakeVersions[0].Height != bestHeightExpected {
			t.Errorf("first block height was %d, expected %d",
				stakeVersions.StakeVersions[0].Height, bestHeightExpected)
		}

		if len(stakeVersions.StakeVersions[0].Votes) == 0 {
			t.Error("votes is empty")
		}

	}
}
