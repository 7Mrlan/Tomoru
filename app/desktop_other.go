//go:build !linux

package main

// noVoice: the engine says the samples aloud here.
const noVoice = false

// useDesktop picks how the windows reach the desktop, which only Linux has more than one way to.
func useDesktop() {}
