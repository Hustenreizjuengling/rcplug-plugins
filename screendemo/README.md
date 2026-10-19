# Screen demo

**Experimental.** Screen demo shows that a plugin can have the device's screen: it
stops RØDE's screen for a few minutes, shows a page of its own and gives the screen
back. The page shows rcplug's version, the device's name, model and firmware, and how
the console holds the mixer meanwhile.

![The page on the device's screen](screendemo.png)

Not yet tried on a real Duo; until then it is not in the catalog.

## How it starts and ends

- It starts only from the dashboard (*Start* on its card), never by itself, not after a
  restart of rcplug and not from a backup.
- Before it starts, rcplug stops RØDE's screen program; the device keeps mixing and
  recording, and the console holds the mixer alone meanwhile (full control).
- The page ends, and RØDE's screen comes back, with **Back to RØDE**, a press of three
  seconds anywhere on the screen, after the set number of minutes (default 5), with
  *Stop* in the dashboard, or when rcplug stops or is updated. If rcplugd itself ends
  while the page is shown, it gives the screen back when it starts again.
- The big encoder does nothing while the page is shown (RØDE's screen program reads
  it).

Where a finger touched the screen, a dot stays on the page (the last eight): it shows
whether touches land where the page is drawn.

## Settings

| Option | Meaning |
|---|---|
| Back to RØDE after (minutes) | 1 to 30, default 5 |
| Language on the screen | English or German |

## Requirements

- rcplug 0.8.11 or later (permission `screen`; older versions do not load the plugin).
- A RØDECaster Duo: the page is drawn for its panel (720 × 1280, turned to landscape
  as RØDE's screen program draws it) and its touchscreen (`/dev/input/event0`).

## How it draws

Only the Go standard library and rcplug's SDK, as for every plugin here:

- **Display:** the kernel's DRM interface with plain ioctls: it finds the connected
  panel and its mode, makes a buffer of its own (XRGB8888), and sets it on the panel's
  CRTC ([display_linux.go](display_linux.go)). The page is drawn on a landscape canvas
  of 1280 × 720 and turned by 90° into the portrait buffer, with the page's top on the
  panel's right, as RØDE's screen program has it.
- **Touch:** evdev, the touchscreen's single-touch axes scaled by the ranges the
  driver reports ([touch_linux.go](touch_linux.go)).
- **Text:** glyph atlases of the Go fonts in [font/](font/), made by
  [tools/fontatlas](../tools/fontatlas/main.go) (`cd tools/fontatlas && go run . -out
  ../../screendemo/font`).

It writes nothing on the device but the backlight: if RØDE's screen program had dimmed
it below a tenth or switched it off, the page turns it up to half and puts the old value
back when it ends, before RØDE's screen program starts again and sets its own. Its
buffer goes with it.

## Development

```sh
go run ./screendemo --png page.png              # the page as an image, no device needed
go run ./screendemo --png page.png --lang de    # in German
go test ./screendemo                            # turn, touch mapping, button, long press
```

Without a console the *Console* row says *not reachable*. On a machine with a DRM card,
the tests also find its output and make a buffer, without showing it.
