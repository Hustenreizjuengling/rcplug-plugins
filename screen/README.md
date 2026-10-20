# Screen — an alternative interface on the device's touchscreen

`screen` puts a second, open interface on the RØDECaster's screen: a mixer,
channel and processing pages, SMART pads, the menus and settings, and pages
for rcplug and other plugins. It is built to look and work like RØDE's own
screen, and to be extended with your own controls.

It ships **no RØDE assets**. The icons and text are read from the device's own
screen program (`/usr/bin/rc_gui_juce`) at run time, and the font is
Liberation Sans (OFL), the same font the device uses. On GitHub there is only
this plugin's own code.

## How it runs on the device

- The plugin has the `screen` permission, so the host stops RØDE's screen
  program while the plugin runs and starts it again afterwards. Audio keeps
  running the whole time; only the screen changes.
- It draws with the kernel's DRM interface and reads the touchscreen through
  evdev (package `internal/panel`, shared with PlugRain).
- It controls the desk through the **console** plugin's HTTP API (package
  `internal/desk`): it loads the state tree, follows the live stream of
  changes, and writes single properties back optimistically. It needs the
  `desk:write` permission, and the console must be running.
- The main knob works as on RØDE's screen (package `internal/app`, the
  focus): on the mixer it sets the monitor level, a press moves it on to the
  virtual faders (and back), holding it mutes the focused one. On every other
  page turning moves a green ring from control to control, a press taps the
  control or starts adjusting a value (turn, then press again to let go), and
  holding the knob goes back a page. The knob's turns and presses reach the
  plugin through the desk's tree (`ENCODER.encoderSignal`, `encoderPressed`).
- RØDE's screen comes back with the **Back to RØDE** button, when the plugin
  is stopped from the dashboard, or if it exits for any reason.

## Safety

If the plugin exits — on purpose, on an error or on a crash — the host starts
RØDE's screen program again. The audio engine is independent of the screen, so
the sound is never interrupted. Settings that rcplug does not change (network,
Bluetooth, firmware, shows, storage, recordings) are shown read-only, with a
button that hands the screen back to RØDE where they are made.

## Development without a device

The icons and text then come from a folder of the device's extracted
resources, passed with `--resources`:

    # one frame to a PNG
    go run ./screen --png out.png --resources <folder> --lang de --page settings

    # the whole interface in a web browser (click and drag like the touchscreen)
    go run ./screen --serve 127.0.0.1:8099 --resources <folder> --lang de

`--page` is one of: `settings`, `outputs`, `display`, `system`, `pads`,
`routing`, `mix`, `rcplug`, `info`, `channel`, `channel:<n>`, `assign`,
`proc:<hpf|gate|comp|deesser|eq|aphex|pan>`, `master`, `metering`, `listen`,
`faders`, `language`, `datetime`, `brightness`, `keyboard`, `dialog` (or
omitted for the mixer). In the browser the main knob is the mouse wheel over
the picture, the buttons under it, or the arrow keys (space = press,
backspace = hold).

## Extending it

Other plugins can add a page without drawing anything themselves: they serve a
small JSON description of their controls at `/plugins/<name>/screen` and accept
changes there. The interface renders those controls in the RØDE style, so a
plugin can never break the screen. The controls are: `toggle`, `slider`,
`choice`, `button` (with an optional confirmation), `value` and `meter`.

## Packages

| package | what it does |
|---|---|
| `internal/gfx` | software renderer (anti-aliased paths, strokes, gradients, images) |
| `internal/svg` | SVG subset for RØDE's icons |
| `internal/font` | TrueType text with Liberation Sans |
| `internal/rode` | reads RØDE's icons and text from the device at run time |
| `internal/panel` | DRM output, touchscreen and backlight |
| `internal/ui` | the look: theme, view tree with partial redraw, controls |
| `internal/desk` | the live desk model over the console |
| `internal/pages` | the screens |
| `internal/app` | navigation, the shell and the draw/touch loop |
| `internal/ext` | discovers plugins and their pages |
| `internal/preview` | the browser and PNG preview |
