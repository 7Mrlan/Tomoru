// The engine's calls into PulseAudio's API, in C: its callbacks are C's. PipeWire's
// pipewire-pulse serves the same API, so this hears on either.
#ifndef TOMORU_ENGINE_LINUX_H
#define TOMORU_ENGINE_LINUX_H

#include <stdint.h>

enum {
	TC_CHANGED = -1, // the default output changed: open it again
	TC_ENDED = -2,   // the stream or the connection ended: the reason is in tc_why
};

typedef struct tc_capture tc_capture;

// tc_open connects to the sound server and records what the default output plays, its monitor,
// as 16 kHz mono 16-bit, which the server converts to. It returns 0, or -1 with the reason in
// why, a buffer of n bytes; *out is then NULL.
int tc_open(tc_capture **out, char *why, int n);
// tc_sink is the output being heard.
const char *tc_sink(tc_capture *c);
// tc_read takes up to n samples heard into buf, waiting up to ms milliseconds for some. It
// returns how many, 0 when none came, or TC_CHANGED or TC_ENDED.
int tc_read(tc_capture *c, int16_t *buf, int n, int ms);
// tc_why is why tc_read returned TC_ENDED.
const char *tc_why(tc_capture *c);
void tc_close(tc_capture *c);

#endif
