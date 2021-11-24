// Copyright (c) 2017-2021 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || linux || nacl || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package main

import "syscall"

// UseSIGToReloadTemplates wraps (*WebUI).UseSIGToReloadTemplates for
// non-Windows systems, where there are actually signals.
func (td *WebUI) UseSIGToReloadTemplates() {
	td.reloadTemplatesSig(syscall.SIGUSR1)
}
