//go:build !windows && !linux

package overlay

import "github.com/egoist/mygo"

// toolWindow does nothing: elsewhere, apps do not judge what covers them by its window's styles.
func toolWindow(*mygo.Window) {}

// pointerUnseen: the system tells where the pointer is, over any window.
const pointerUnseen = false
