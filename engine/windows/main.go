//go:build windows

// tomoru-engine: Tomoru's Windows engine. It hears what the computer plays with WASAPI's
// loopback, and speaks docs/engine-protocol.md on stdin and stdout, through engine/shared. It
// hears only: Windows has no recognizer for apps to use, so tomoru-asr or the cloud recognizes
// what it hears.
//
//	tomoru-engine listen --locale <bcp47> --asr none [--app <exe> | --file <wav>] [--vad <silero_vad.onnx> | none]
//	tomoru-engine speak --locale <bcp47> -- <sentence>...
//	tomoru-engine select --out <png>
//	tomoru-engine access | request-access | screen-access | request-screen-access | apps
//
// An app is its program's file name, as "chrome.exe": its helper processes are of the same file.
package main

import "tomoru-engine-shared"

func main() {
	shared.Run(shared.Platform{Engine: "windows", System: "Windows", App: "<exe>",
		Source: systemSource, Apps: listApps, Speak: speak, Select: selectArea})
}
