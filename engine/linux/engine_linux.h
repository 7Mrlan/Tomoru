// The engine's calls into PulseAudio's API and GTK, in C: their callbacks are C's. PipeWire's
// pipewire-pulse serves PulseAudio's API, so this hears on either.
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

// tc_select has the user select a part of the screen, over a picture of every screen, which the
// portals take for app, by the ID they know it by, and writes the part's picture to out, as PNG.
// It returns 1 with where the part is, in the pixels of the monitor it is on; 0 when the user
// cancelled, writing nothing; -1 with the reason in why, a buffer of n bytes.
int tc_select(const char *app, const char *out, int *x, int *y, int *w, int *h, char *why, int n);

enum { TC_DENIED = 0, TC_GRANTED = 1, TC_UNKNOWN = 2 }; // stored answers; the engine reports no TC_DENIED

// tc_screen_access tells whether the user lets app, by the ID the portals know it by, take
// pictures of the screen: TC_GRANTED, or TC_UNKNOWN until they did, a no included, which is not
// for good: when request, it forgets a no they gave before and has the desktop ask them, by
// taking a picture it does not keep. -1 with the reason in why when it cannot tell. Under X11
// the engine needs no permission.
int tc_screen_access(const char *app, int request, char *why, int n);

#endif
