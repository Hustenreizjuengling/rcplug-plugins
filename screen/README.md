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
- A combo input's channel page carries RØDE's microphone choice: a carousel
  of the models with their pictures (`INPUTSOURCE.inputMicrophoneType`, the
  values 0…10 in the order RØDE's screen program adds them: Line-In,
  Instrument, Dynamic, Condenser, PodMic, Broadcaster, Procaster, NT1, NT1-A,
  RE20, SM7B). A tap on a neighbour or a swipe chooses; only the type is
  written, as RØDE's screen does — the mixer then sets the model's gain,
  phantom power and voicing.
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
  The channel buttons above the faders select a channel in the desk
  (`SYSTEM.systemChannelSelected`), which opens the strip's channel page;
  the next press deselects it and the pages close. Opening or closing a
  channel's pages by touch sets the same property, as RØDE's screen does, so
  the button's light always shows whose pages are open.
- The texts follow the desk's language setting (`GUI.lang`) while the plugin
  runs: a start before the desk had loaded begins in English and switches as
  soon as the setting is known, and the language page applies at once, as on
  RØDE's screen.
- Since 0.4.0 it carries every page of RØDE's screen program (firmware
  1.7.6), read from the program itself: the mixer, the channel pages per
  source with the processors and effects, the pads with their editors,
  shows and recordings, every settings page, CallMe, network and Bluetooth,
  and the first-run welcome and setup assistant. It writes the desk's
  properties the way that program does; saving presets and quick dials,
  the system requests of the settings pages and setting the clock need
  rcplug 0.8.22 or later (the console's screen level, `api/node/add` and
  `api/node/remove`, `POST /api/system/clock`). What could not be read from
  the program is listed in the CHANGELOG as open.
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

Other plugins can add a page without drawing pixels themselves: they serve a
small JSON description of their controls at `/screen` (the host proxies it as
`/plugins/<name>/screen`) and accept changes there. The interface renders those
controls in the RØDE style, so a plugin can never break the screen. While the
plugin runs, its page has a tile on the rcplug menu and a button on its plugin
page under rcplug → Plugins. Sonar and Backup have one.

`GET /screen?lang=<de|en>` answers with the page in that language:

```json
{"title": "Sonar", "icon": "IconListen", "every": 100, "controls": [
  {"id": "radar",     "type": "canvas", "w": 100, "h": 100, "shapes": [
    {"shape": "arc",    "x": 50, "y": 56, "r": 44, "a0": 180, "a1": 360, "color": "line"},
    {"shape": "text",   "x": 50, "y": 8, "text": "0°", "size": "tiny", "color": "dim"},
    {"shape": "line",   "id": "beam", "points": [[50, 56], [20, 24]], "width": 0.5, "color": "accent", "alpha": 0.7},
    {"shape": "glow",   "id": "glow", "x": 36, "y": 40, "r": 6, "color": "accent"},
    {"shape": "circle", "id": "blip", "x": 36, "y": 40, "r": 0.9, "color": "text"}
  ]},
  {"id": "source",    "type": "value",  "label": "Listening to", "text": "Mic 1 + Mic 2"},
  {"id": "level",     "type": "meter",  "label": "Level", "value": 0.6, "min": 0, "max": 1},
  {"id": "listen",    "type": "choice", "label": "Listen to", "options": ["Mic 1 + Mic 2", "USB 1"], "value": 0},
  {"id": "smoothing", "type": "slider", "label": "Smoothing", "value": 0.8, "min": 0, "max": 0.99, "unit": ""},
  {"id": "flip",      "type": "toggle", "label": "Swap left and right", "on": false},
  {"id": "reset",     "type": "button", "label": "Reset", "danger": true, "confirm": "Forget the calibration?"}
]}
```

`value` and `meter` show a text or a level and follow the plugin (the page is
fetched again while it is shown: about twice a second, or at the pace the page
asks for in `every`, 80 to 5000 ms). A `choice` lists its options and names the
chosen one by index in `value`. A change is a `POST /screen` with
`{"control": "<id>", "value": …}`: `true`/`false` for a toggle, the number for a
slider, the option's index for a choice, `true` for a button (after the
confirmation, if one is declared). The plugin answers 200 and reflects the
change in its next description. `icon` names a RØDE icon for the plugin's tile
(`IconListen`, `IconMetering`, `IconRecordings` …; `IconSystem` when left out).

**A picture of its own.** A `canvas` is the plugin's own display: a list of
shapes in the plugin's units (`w` × `h`, 100 × 100 when left out) that the
screen scales to fit and draws itself, in the RØDE style. The first canvas of a
page gets the big spot — on the left when it is about square or taller, on top
when it is wide — and the controls take the rest. Shapes:

| shape | fields |
|---|---|
| `circle` | `x`, `y`, `r` (filled) |
| `glow` | `x`, `y`, `r`: a soft light fading to the edge |
| `ring` | `x`, `y`, `r`, `width` (outline) |
| `arc` | `x`, `y`, `r`, `a0`, `a1` in degrees (0 = right, growing clockwise), `width` |
| `line` | `points` `[[x, y], …]`, `width` (round ends and joins) |
| `polygon` | `points` (filled, closed) |
| `rect` | `x`, `y`, `w`, `h`, `radius` (filled) |
| `text` | `x`, `y`, `text`, `size` (`tiny` `small` `body` `label` `title` `big` `huge`), `align` (`left` `centre` `right`; the point is the text's middle line) |

Every shape takes `color` — a name from the look (`accent`, `text`, `dim`,
`muted`, `line`, `panel`, `black`, `record`, `warn`, `blue`) or `#rrggbb`;
`accent` when left out, `text` for text — and `alpha` (0 … 1). A `width` of 0 is
a hairline. A shape with an `id` glides from where the previous description had
it to where this one has it, over the time between the two descriptions, so a
plugin that describes its picture ten times a second gets motion as smooth as
the screen draws; shapes without an id just appear where they are. The canvas
is display only: touch goes to the controls.

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
