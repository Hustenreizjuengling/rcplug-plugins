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
  `desk:write` permission, and the console must be running. With rcplug
  0.8.16 or later it writes at the console's expert level, as RØDE's screen
  does (older hosts refuse the settings beyond the basic rules: metering,
  haptics, the clock, the pads' names and modes, and more).
- The buttons above the physical faders open the strip's channel page
  (`SYSTEM.systemChannelSelected`), as on RØDE's screen.
- The channel page's **Preset** button lists the desk's processing presets
  and applies one (`FXPRESET.fxPresetContents`, `Name|value|…` in the order
  read off a device: depth, sparkle and punch, APHEX, compressor, de-esser,
  noise gate, EQ, high-pass filter). Saving and deleting presets stay on
  RØDE's screen: they change the tree's nodes, which the console does not
  pass on.
- When it ends it leaves the display black (the buffers cleared and the
  black held for two refreshes, until the panel has it whole), so nothing of
  its last page flickers through while the next program takes the panel.
- The main knob works as on RØDE's screen (package `internal/app`, the
  focus): on the mixer it sets the monitor level, a press moves it on to the
  virtual faders (and back), holding it mutes the focused one. On every other
  page turning moves a green ring from control to control, a press taps the
  control or starts adjusting a value (turn, then press again to let go), and
  holding the knob goes back a page. The knob's turns and presses reach the
  plugin through the desk's tree (`ENCODER.encoderSignal`, `encoderPressed`).
  The channel buttons above the faders open the strip's channel page, and
  return to the mixer when pressed again there (`SYSTEM.systemChannelSelected`).
- RØDE's screen comes back with the **Back to RØDE** button, when the plugin
  is stopped from the dashboard, or if it exits for any reason.
- It is started in the dashboard the first time. With rcplug 0.8.17 or later
  it then comes back at every start of the device (once RØDE's screen program
  is up) until it is stopped there; if it should ever crash three times in ten
  minutes, rcplug switches it off and RØDE's screen stays.

## Safety

If the plugin exits — on purpose, on an error or on a crash — the host starts
RØDE's screen program again. The audio engine is independent of the screen, so
the sound is never interrupted. Settings that rcplug does not change (network,
Bluetooth, firmware, shows, storage, recordings) are shown read-only, with a
button that hands the screen back to RØDE where they are made. The plugin
never records on the device: a SMART pad's sound comes from the RØDE app, and
its editor trims, fades and levels it.

## Development without a device

The icons and text then come from a folder of the device's extracted
resources, passed with `--resources`:

    # one frame to a PNG
    go run ./screen --png out.png --resources <folder> --lang de --page settings

    # the whole interface in a web browser (click and drag like the touchscreen)
    go run ./screen --serve 127.0.0.1:8099 --resources <folder> --lang de

`--page` is one of: `settings`, `outputs`, `display`, `system`, `pads`,
`pad:<n>` (a pad's editor), `pad:<n>:options|new|colour`, `routing`, `mix`,
`mix:<output>`, `rcplug`, `info`, `channel`, `channel:<n>`, `assign`, `fx`,
`headphones` (the headphone popup), `transfer`, `bluetooth-out`,
`proc:<hpf|gate|comp|deesser|eq|aphex|pan>`, `master`, `metering`, `listen`,
`faders`, `language`, `datetime`, `brightness`, `keyboard`, `dialog` (or
omitted for the mixer). In the browser the main knob is the mouse wheel over
the picture, the buttons under it, or the arrow keys (space = press,
backspace = hold).

## Demos with the screen of their own

A plugin that takes over the screen itself (permission `screen`, like PlugRain) gets
a tile on the rcplug page: a tap hands the display over to it without RØDE's screen in
between, and when that plugin ends this interface comes back (rcplug 0.8.18).

## Extending it

Other plugins can add a page without drawing anything themselves: they serve a
small JSON description of their controls at `/screen` (the host proxies it as
`/plugins/<name>/screen`) and accept changes there. The interface renders those
controls in the RØDE style, so a plugin can never break the screen; it lists the
page under rcplug → Plugins while the plugin runs. Sonar and Backup have one.

`GET /screen?lang=<de|en>` answers with the page in that language:

```json
{"title": "Sonar", "controls": [
  {"id": "source",    "type": "value",  "label": "Listening to", "text": "Mic 1 + Mic 2"},
  {"id": "level",     "type": "meter",  "label": "Level", "value": 0.6, "min": 0, "max": 1},
  {"id": "listen",    "type": "choice", "label": "Listen to", "options": ["Mic 1 + Mic 2", "USB 1"], "value": 0},
  {"id": "smoothing", "type": "slider", "label": "Smoothing", "value": 0.8, "min": 0, "max": 0.99, "unit": ""},
  {"id": "flip",      "type": "toggle", "label": "Swap left and right", "on": false},
  {"id": "reset",     "type": "button", "label": "Reset", "danger": true, "confirm": "Forget the calibration?"}
]}
```

`value` and `meter` show a text or a level and follow the plugin (the page is
fetched again about twice a second while it is shown). A `choice` lists its
options and names the chosen one by index in `value`. A change is a
`POST /screen` with `{"control": "<id>", "value": …}`: `true`/`false` for a
toggle, the number for a slider, the option's index for a choice, `true` for a
button (after the confirmation, if one is declared). The plugin answers 200
and reflects the change in its next description.

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
