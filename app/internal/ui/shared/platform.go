package shared

import "runtime"

// Windows tells whether the app runs on Windows, which has words of its own for some of what the
// app says: the taskbar's corner for the menu bar, File Explorer for Finder, a computer for a Mac.
var Windows = runtime.GOOS == "windows"

// linux tells whether the app runs on Linux, which says a computer, as Windows does, and has words
// of its own for the rest.
var linux = runtime.GOOS == "linux"

// OnWindows is win on Windows and Linux, else mac.
func OnWindows(mac, win string) string {
	if Windows || linux {
		return win
	}
	return mac
}

// OnEach is mac, win or lin, for the system the app runs on.
func OnEach(mac, win, lin string) string {
	if linux {
		return lin
	}
	return OnWindows(mac, win)
}
