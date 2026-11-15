package main

// #cgo pkg-config: gtk+-3.0
// #include <stdlib.h>
// #include "engine_linux.h"
import "C"

import (
	"errors"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

// selectArea has the user select a part of the screen and writes its picture to path, as PNG, and
// tells where it is. Cancelled, it writes nothing, as macOS's screencapture does, and tells nothing.
func selectArea(path string) (*image.Rectangle, error) {
	runtime.LockOSThread() // GTK, on the thread that started it
	app, out := C.CString(appID()), C.CString(path)
	defer C.free(unsafe.Pointer(app))
	defer C.free(unsafe.Pointer(out))
	var x, y, w, h C.int
	var why [256]C.char
	switch C.tc_select(app, out, &x, &y, &w, &h, &why[0], C.int(len(why))) {
	case 1:
		at := image.Rect(int(x), int(y), int(x+w), int(y+h))
		return &at, nil
	case 0:
		return nil, nil
	default:
		return nil, errors.New(C.GoString(&why[0]))
	}
}

// screenAccess tells whether the user lets the app take pictures of the screen, under Wayland,
// where the desktop asks them: "granted", else "unknown", a no included, since asking again
// works; request has the desktop ask, which GNOME does only while the app has the focus, as in
// the first-run guide.
func screenAccess(request bool) string {
	app := C.CString(appID())
	defer C.free(unsafe.Pointer(app))
	var why [256]C.char
	ask := C.int(0)
	if request {
		ask = 1
	}
	switch C.tc_screen_access(app, ask, &why[0], C.int(len(why))) {
	case C.TC_GRANTED:
		return "granted"
	case C.TC_UNKNOWN:
		return "unknown"
	default:
		log.Printf("screen access: %s", C.GoString(&why[0]))
		return "unknown"
	}
}

// appID is the ID the portals know the app by, which MyGo gives them: its desktop entry's, the
// one that launched it, else its program's name.
func appID() string {
	for _, env := range []string{"GIO_LAUNCHED_DESKTOP_FILE", "BAMF_DESKTOP_FILE_HINT"} {
		if p := os.Getenv(env); p != "" {
			return strings.TrimSuffix(filepath.Base(p), ".desktop")
		}
	}
	exe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(os.Getppid()), "exe"))
	if err != nil {
		return ""
	}
	return filepath.Base(exe)
}
