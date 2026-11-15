#!/bin/sh
# Builds Tomoru's Linux programs, on Linux x86-64, into app/resources/linux-amd64/bin/:
# - tomoru-engine (engine/linux): hearing the default output, through PulseAudio's API (libpulse,
#   which PipeWire's pipewire-pulse serves too), and selecting a part of the screen (GTK 3);
# - tomoru-asr (engine/asr): recognition with downloaded models;
# - the sherpa-onnx libraries both load from beside them, which come with its Go module, and its
#   license in licenses/.
# It needs a C compiler, pkg-config, libpulse's and GTK 3's headers and patchelf (on Debian and
# Ubuntu: apt install build-essential pkg-config libpulse-dev libgtk-3-dev patchelf). The voice detector they load,
# vad/silero_vad.onnx, comes from scripts/fetch-vad.sh.
set -e
cd "$(dirname "$0")/.."
res=$PWD/app/resources/linux-amd64
mkdir -p "$res/bin" "$res/licenses"
export CGO_ENABLED=1 GOOS=linux GOARCH=amd64
(cd engine/linux && go build -trimpath -o "$res/bin/tomoru-engine" .)
(cd engine/asr && go build -trimpath -o "$res/bin/tomoru-asr" .)

mod=$(cd engine/asr && go list -m -f '{{.Dir}}' github.com/k2-fsa/sherpa-onnx-go-linux)
for lib in libsherpa-onnx-c-api.so libonnxruntime.so; do
	rm -f "$res/bin/$lib" # the module cache's files are read-only, and so are their copies
	cp "$mod/lib/x86_64-unknown-linux-gnu/$lib" "$res/bin/"
	chmod 755 "$res/bin/$lib"
done
# cgo points the programs at the module cache: point them beside themselves instead.
for bin in tomoru-engine tomoru-asr; do
	patchelf --set-rpath '$ORIGIN' "$res/bin/$bin"
done
scripts/package-licenses.sh linux-amd64
echo "built $res/bin/tomoru-engine and tomoru-asr"
