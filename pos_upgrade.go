// Copyright (c) 2017-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/decred/dcrd/dcrjson"
)

type posUpgrade struct {
	// Completed is a bool indicating whether the stake version has upgraded to the latest version.
	Completed bool
	// UpgradeInterval is the interval where the upgrade took place. Nil if upgrade has not happened.
	UpgradeInterval dcrjson.VersionInterval
}
