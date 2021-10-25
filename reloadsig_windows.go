// Copyright (c) 2017-2021 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import "fmt"

// UseSIGToReloadTemplates wraps (*WebUI).UseSIGToReloadTemplates for Windows
// systems, where there are no signals to use.
func (td *WebUI) UseSIGToReloadTemplates() {
	fmt.Println("Signals are unsupported on Windows.")
}
