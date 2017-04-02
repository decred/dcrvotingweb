package main

import "testing"

func TestStakeInfoStub(t *testing.T) {
	stub := NewStakeInfoStub()
	if stub == nil {
		t.Error("Unable to create stub")
	}

	hash, height, err := stub.GetBestBlock()
	if err != nil {
		t.Error("GetBestBlock failed: ", err)
	}

	bestHashExpected := "0000000000b0a130a86d7fe1e475b39da048c3215d958ea4b2e102ad5a687fc0"
	if hash.String() != bestHashExpected {
		t.Errorf("Incorrect hash.  Got %s, expecting %s",
			hash.String(), bestHashExpected)
	}

	bestHeightExpected := int64(289984)
	if height != bestHeightExpected {
		t.Errorf("Incorrect height.  Got %d, expecting %d",
			height, bestHeightExpected)
	}

	// go test -v
	t.Log("Hash: ", hash)
	t.Log("Height: ", height)

	block, err := stub.GetBlockVerbose(hash, false)
	if err != nil {
		t.Errorf("Unable to get block %v: %v", hash, err)
	}

	if block.Height != bestHeightExpected {
		t.Errorf("Got block with height %d, expected %d",
			block.Height, bestHeightExpected)
	}

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

	// TODO: GetStakeVersions and GetStakeVersionInfo
}
