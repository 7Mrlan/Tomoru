// Package shared is what Tomoru's Go engines share: the protocol of docs/engine-protocol.md, its
// command line, hearing a Source or a recording, and voice detection. Each engine, as
// engine/windows and engine/linux, gives it what its system does its own way.
package shared

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"slices"
)

// Platform is what an engine does its own way.
type Platform struct {
	Engine string // the engine's name in hello, as "windows"
	System string // the system's, as people say it: "Windows"
	App    string // what --app takes, as "<exe>"

	// Source hears the system, or app alone when it is not "".
	Source func(app string) Source
	// Apps sends an `app` for each app that can be heard on its own.
	Apps func()
	// Speak says lines aloud in locale through the default output, as `speak` does.
	Speak func(locale string, lines []string)
	// Select has the user select a part of the screen and writes its picture to path, as PNG,
	// and tells where it is; nil when cancelled, with nothing written.
	Select func(path string) (*image.Rectangle, error)
}

// Run runs the command of the command line, as p does it: hearing only, for the system has no
// recognizer for apps to use, so tomoru-asr or the cloud recognizes what it hears.
func Run(p Platform) {
	log.SetFlags(0)
	log.SetPrefix("[engine] ")
	Send(Event{Type: "hello", Protocol: 1, Engine: p.Engine, Version: version})
	usage := func() {
		fmt.Fprintf(os.Stderr, "usage: tomoru-engine listen --locale <bcp47> --asr none [--app %s | --file <wav>] [--vad <model> | none] | speak --locale <bcp47> -- <sentence>... | select --out <png> | access | request-access | screen-access | request-screen-access | apps\n", p.App)
		os.Exit(2)
	}
	if len(os.Args) < 2 {
		usage()
	}
	noASR := p.System + " has no speech recognition for apps"
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "access", "request-access", "screen-access", "request-screen-access":
		// The system asks no permission to hear what plays, nor to capture the screen.
		Send(Event{Type: "access", State: "granted"})

	case "apps":
		p.Apps()

	case "prepare":
		Fail("unsupported-locale", "%s", noASR)

	case "speak":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		locale := fs.String("locale", "", "the language to speak, BCP 47")
		fs.Parse(args)
		lines := fs.Args()
		if i := slices.Index(args, "--"); i >= 0 {
			lines = args[i+1:]
		}
		if *locale == "" || len(lines) == 0 {
			usage()
		}
		p.Speak(*locale, lines)

	case "select":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		path := fs.String("out", "", "where the picture of the selection goes, as PNG")
		fs.Parse(args)
		if *path == "" {
			usage()
		}
		at, err := p.Select(*path)
		if err != nil {
			Fail("capture-failed", "%v", err)
		}
		if at != nil {
			Send(Event{Type: "area", X: at.Min.X, Y: at.Min.Y, Width: at.Dx(), Height: at.Dy()})
		}

	case "listen":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		fs.String("locale", "", "the language spoken, BCP 47: the recognizer's business")
		asr := fs.String("asr", "", "none: hear only")
		app := fs.String("app", "", "the app to hear alone, "+p.App)
		file := fs.String("file", "", "a WAV recording to hear instead of the system")
		vad := fs.String("vad", defaultVAD(), "Silero VAD's ONNX model, or none")
		fs.Parse(args)
		if *asr != "none" {
			Fail("unsupported-locale", "%s: listen --asr none", noASR)
		}
		var src Source = &FileSource{path: *file}
		if *file == "" {
			src = p.Source(*app)
		}
		listen(src, *vad)
		select {} // until stdin closes

	default:
		usage()
	}
}

// listen hears src and sends what it hears until stdin closes.
func listen(src Source, vad string) {
	// Commands come on stdin; when it closes, the app has gone, and so does the engine.
	go func() {
		if err := readCommands(os.Stdin, func(Command) {}); err != nil {
			Fail("internal", "stdin: %v", err)
		}
		log.Print("stdin closed")
		os.Exit(0)
	}()
	var voice func([]float32)
	if vad != "none" {
		if v, err := newSilero(vad); err != nil {
			log.Printf("voice detection off: %v", err)
		} else {
			voice = Feeder(NewDetector(v, func(on bool, t float64) { Send(Event{Type: "voice", On: on, T: t}) }))
			// Voice detection is on: nobody speaks yet.
			Send(Event{Type: "voice", On: false, T: 0})
		}
	}
	if err := NewHearer(Send, voice).Start(src); err != nil {
		Fail("capture-failed", "%v", err)
	}
}
