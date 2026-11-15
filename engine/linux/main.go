//go:build linux

// tomoru-engine: Tomoru's Linux engine. It hears what the default output plays, its monitor,
// through PulseAudio's API, which PipeWire serves too, has the user select a part of the screen,
// and speaks docs/engine-protocol.md on stdin and stdout, through engine/shared. It hears only:
// Linux has no recognizer for apps to use, so tomoru-asr or the cloud recognizes what it hears.
//
//	tomoru-engine listen --locale <bcp47> --asr none [--file <wav>] [--vad <silero_vad.onnx> | none]
//	tomoru-engine select --out <png>
//	tomoru-engine access | request-access | screen-access | request-screen-access | apps
//
// It neither hears one app alone nor speaks yet.
package main

import "tomoru-engine-shared"

func main() {
	shared.Run(shared.Platform{Engine: "linux", System: "Linux", App: "<app>",
		Source: systemSource, Apps: func() {}, Speak: speak, Select: selectArea, ScreenAccess: screenAccess})
}

func speak(locale string, _ []string) { shared.Fail("no-voice", "no voice for %s on Linux", locale) }
