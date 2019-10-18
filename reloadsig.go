// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package main

import "syscall"

// UseSIGToReloadTemplates wraps (*WebUI).UseSIGToReloadTemplates for
// non-Windows systems, where there are actually signals.
func (td *WebUI) UseSIGToReloadTemplates() {
	td.reloadTemplatesSig(syscall.SIGUSR1)
}
