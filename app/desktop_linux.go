package main

import (
	"os"
	"strings"
)

// noVoice: the engine says no sample aloud here.
const noVoice = true

// useDesktop picks how the windows reach the desktop, before they open. Under Wayland GNOME lets
// no window stay over the others, the subtitles neither, and has no layer for them as KDE's and
// others' have; it keeps an X11 window over them, so there Tomoru's windows are X11 ones
// (XWayland). GDK_BACKEND set, the choice is the user's.
func useDesktop() {
	if os.Getenv("WAYLAND_DISPLAY") == "" || os.Getenv("DISPLAY") == "" || os.Getenv("GDK_BACKEND") != "" {
		return
	}
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME") {
		os.Setenv("GDK_BACKEND", "x11")
	}
}
