package overlay

import (
	"os"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo"
)

// toolWindow keeps the keyboard from w: an X11 window manager gives it to a window clicked in, or
// to the one left when another closes, and the subtitles would take it from the video under them,
// and hold it while Tomoru's other windows open, which then show behind. Clicks and drags reach
// it all the same. Where the subtitles are a layer's surface (Wayland), they never take it.
func toolWindow(w *mygo.Window) {
	gtk, err := purego.Dlopen("libgtk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	var setAcceptFocus func(window uintptr, accept int32)
	purego.RegisterLibFunc(&setAcceptFocus, gtk, "gtk_window_set_accept_focus")
	setAcceptFocus(w.NativeHandle(), 0)
}

// pointerUnseen: in a Wayland session, an app hears where the pointer is only while it is over one
// of the app's windows, X11 ones under XWayland included.
var pointerUnseen = os.Getenv("WAYLAND_DISPLAY") != ""
