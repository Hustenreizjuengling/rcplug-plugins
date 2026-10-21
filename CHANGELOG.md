# Changelog

All notable changes. The format follows [Keep a Changelog](https://keepachangelog.com/),
versions follow [Semantic Versioning](https://semver.org/) (before 1.0, minor versions
may break compatibility).

## 0.5.39 - 2026-10-21

### Removed
- **Screen 0.4.6 is Screen 0.4.5 again.** The browser interface and the device's screen
  on request (0.5.0, 0.5.1) are taken back out: the plugin holds the device's screen for
  its whole run, starts from the dashboard, and RØDE's screen comes back when it ends,
  as before. Works with rcplug 0.8.23 or 0.8.26.

## 0.5.38 - 2026-10-21

### Fixed
- **Screen 0.5.1: the browser page and the device's screen after the update.** 0.5.0
  did not declare its web page to the host, so the dashboard had no Screen page and
  nothing could take the device's screen; and a fresh install or an update that lost the
  screen left RØDE's interface on the device with no way back. The page is declared
  now, and the plugin remembers itself whether the interface belongs on the device's
  screen (it does until "Back to RØDE"): when RØDE's program has the screen and nobody
  else, it asks the host for it once RØDE's program is up.

## 0.5.37 - 2026-10-21

### Changed
- **Screen 0.5.0: the interface in the browser, the device's screen on request.** The
  plugin now runs like any other plugin and serves the same interface to browsers (the
  dashboard's Screen page, opened as the dashboard's start page): a picture stream of
  the parts that change, the pointer as the finger, the mouse wheel and arrow keys as
  the main knob, buttons for press, hold and the headphone knobs. Every browser gets a
  page stack of its own over the one desk. The device's touchscreen is taken with
  **Take the device's screen** on the rcplug page and given back with **Back to RØDE**
  while the browser interface stays; the host remembers who holds the screen across a
  restart of the device. The welcome flow and the physical knobs and buttons belong to
  the device's screen only. Needs rcplug 0.8.24 or later (permission
  `screen:request`); the old preview server is gone, `--serve` streams the same way.

## 0.5.36 - 2026-10-21

### Fixed
- **Screen 0.4.5: the headphone knobs no longer lag either.** Every reading of a
  turning headphone knob put up the level popup anew and rebuilt the whole page for it;
  the popup now stays up and its slider follows the level itself, as the mixer's ring
  does since 0.4.4.

## 0.5.35 - 2026-10-21

### Fixed
- **Screen 0.4.4: the main knob no longer lags behind a quick turn.** Every detent
  rebuilt the whole mixer page before it was drawn, so fast turns queued up behind the
  drawing and the level kept moving long after the knob had stopped. The views now follow
  the level themselves (the ring, the fader caps), a page rebuilds only when its layout
  changes, and the turns that queued up while a frame was drawn are taken together
  before the next one.

## 0.5.34 - 2026-10-21

### Fixed
- **Screen 0.4.3: a channel button opens its page without the felt delay.** The page
  is built and shown with the press itself, not at the next frame; the microphone
  pictures are scaled once at the start, not when the first channel page opens; rounded
  panels and the gradients fill their straight parts row by row and run the rasterizer
  only at the corners (a page opens five times faster, a mixer frame with moving meters
  repaints only the meter columns); opaque fills copy rows instead of setting pixels.

## 0.5.33 - 2026-10-21

### Added
- **Screen 0.4.2: the last open values, read from the screen program.** The routing
  mix pages write the desk's selected mix (`SYSTEM.systemMixSelected`: the output's
  index while a mix page is open, 13 when none); the processing pages set the
  assignable meter's source the way the program does (gain panel 0, compressor 1,
  de-esser 2, noise gate 3, panning 4); the operation-mode page binds Podcast, Karaoke
  and RØDECaster Sync to `systemMainMode` with the program's dialogs and the reboot
  request; the recordings list reads the record's fourteen fields (name, path, time in
  milliseconds, length, multitrack, envelope, upload state), the player loads and saves
  the envelope as the program does and reads the markers from the WAV file's cue chunk;
  leaving a wireless channel while it pairs cancels the pairing (`radioPair` −1), unpair
  writes the transmitter's serial; the hosted USB microphone is named from the program's
  product table; the brightness page writes the dim time as seconds (rcplug 0.8.23
  takes it while the device still holds the factory's bool).

### Open
- Screen: the player's state and speed numbers and "new show from the current show"
  stay as before (their final writes were not found); the strip shows no transmitter
  status, as the program's strip does not either.

## 0.5.32 - 2026-10-21

### Fixed
- **Screen 0.4.1: findings of a review of the assembled pages.** The new-show options
  ask through RØDE's Export/Skip dialog before the show is replaced (the dialog was
  skipped); the CallMe indicator reads the slot state the desk writes after a call
  (JSON), so the dot goes out; the clock's result reaches the interface on its own loop
  (no page popped from a background task); the welcome flow is marked as shown when it
  starts, so leaving it by the home or a channel button does not bring it back, and its
  language and date-time steps keep the pick and apply it on the tick; the show-loading
  page leaves before showing an error and gives up waiting after eight seconds; no
  headphone popup before the desk is online; a pad with an unknown mixer mode no longer
  rebuilds every frame; the setup assistant makes the desk discoverable once and turns it
  off when it ends; saving a preset before the first tree is refused instead of
  crashing; four texts use RØDE's own keys.

## 0.5.31 - 2026-10-21

### Added
- **Screen 0.4.0: RØDE's screen, page for page.** Every page, control and text of
  RØDE's screen program (firmware 1.7.6) was read from the program itself and rebuilt;
  the plugin writes what that program writes and nothing else. Needs rcplug 0.8.22 for
  the parts marked so.
  - **Mixer and status bar:** the strip icon per source and input mode exactly as the
    device chooses it (combo, headset, wireless, USB, Bluetooth, pads, CallMe, sync
    sources), the network and cell indicators, the clock and date formats, the CallMe
    arrows, the monitor ring's fixed, disabled and auto-muted states, the passing level
    card for the monitor and Bluetooth, the headphone popup with the headphones' type and
    a cross when the outputs are off, the marker hint when a recording starts, and the
    knob's ring colour on the mixer (`ENCODER.encoderColour`).
  - **Channel pages per source:** the assignment page with RØDE's tiles, descriptions,
    the headset tile and the duplicate check; the combo input's gain per mode, the mic
    carousel's hint texts; the headset input; the wireless channel with its transmitter
    (type, battery, signal, gain assist, pad, pairing) and the wireless info page; the
    USB 1, Chat, USB 2 and hosted USB microphone panels; Bluetooth; CallMe; sync sources;
    the presets panel with save (keyboard, overwrite question) and delete (rcplug
    0.8.22); the processing centre with RØDE's tabs, the Presets | Advanced foot and the
    advanced dialog; every processor page with the program's scales, the EQ's draggable
    band dots, the exciter, stereo panning and the hosted microphone's high-pass
    switches; the echo, reverb (size knob with its five rooms), megaphone, robot,
    voice-disguise and pitch panels; the master processing with its meter; the faders
    page with drag and drop. Applying a preset sets the VoxLab-derived values the way the
    program computes them.
  - **Pads:** the bank page with RØDE's arrows, grid and colours; options, rename, the
    twelve pad colours, the new-pad page per kind; the sound editor (envelope handles,
    playhead, play mode, loop, replay, trim, record view, preview); the effects editor
    (inputs, latching, the skeuomorphic panels); the mixer actions (censor, trash talk,
    fade, back channel, ducking); MIDI triggers; CallMe quick dials; RØDECaster Video
    actions; Icecast. Shows: main, new, load, import, export, icons, errors. Recordings:
    the list, the player with jumps, markers, speed and edit handles, delete; the
    unrecognised-disk popup.
  - **Settings:** headphones (type per output with the volume warning), metering,
    haptics, REC button, brightness with dim-after, multitrack with USB 1 modes and
    confirmations, language, date and time with spinners and RØDE's 137 time zones
    (the clock is set through rcplug 0.8.22), MIDI, information, device information
    with regulatory and reset, analytics, firmware update states, storage with erase,
    eject and format, transfer mode, operation mode, cloud connection, compression
    quality, and a number pad. The menus carry RØDE's tiles in RØDE's order.
  - **CallMe and network:** the configuration page with slot states, hosting with its
    code and QR, licence activation, the advanced pages, quick dials and dialling
    (rcplug 0.8.22 for new entries), fader assignment, incoming-call assignment, the
    popups (incoming, conflict, recording warning, welcome, rating, time limit); the
    network pages (status, join with the scan list and password keyboard, advanced,
    DNS, proxy); Bluetooth pairing with discoverability, codes and the device lists.
  - **First run:** the welcome flow and the setup assistant, step for step as the
    program walks them (hearing warning, faders, inputs, microphones, gain, presets,
    Bluetooth, USB, assignment, pads, shows, show name), entered once the way the
    program decides it.

### Changed
- Screen: toasts stay four seconds, as RØDE's notification popup does.

### Open
- Screen: what the program did not give up — `assignableMeterSource` and
  `systemMixSelected` (their value maps), the operation-mode page's binding beyond sync
  mode, the recordings' `recordingContent` field order (shown as the whole text until a
  capture with a card exists), the player's state and speed numbers, a transmitter's
  status on the mixer strip, the RØDECaster Sync mixes, "new show from the current
  show", and real waveforms (the files are the desk's). The pages say so where it
  shows.

## 0.5.30 - 2026-10-21

### Fixed
- **Screen 0.3.12: the interface follows the desk's language.** When the plugin started
  before the console had loaded the desk (after an update or at the device's start) it
  came up in English and stayed so. It now switches to the desk's language as soon as
  the setting is known, and a change on the language page applies at once, as on RØDE's
  screen.

## 0.5.29 - 2026-10-21

### Added
- **Screen 0.3.11: the microphone choice on a combo input's channel page**, as on
  RØDE's screen: a carousel of the models with their pictures (Line-In, Instrument,
  Dynamic, Condenser, PodMic, Broadcaster, Procaster, NT1, NT1-A, RE20, SM7B), the
  chosen one in the middle on the channel's colour. A swipe slides through, a tap on a
  neighbour chooses it, a tap on the middle opens the channel assignment. The list,
  its order and the pictures are RØDE's own (`INPUTSOURCE.inputMicrophoneType` 0…10,
  read from the screen program); the desk sets the model's gain, phantom power and
  voicing.

## 0.5.28 - 2026-10-21

### Fixed
- **Screen 0.3.10: the rcplug and plugin pages show a plugin installed or removed a
  moment ago** without a restart. The list is refreshed when such a page opens and the
  pages are rebuilt when it changes (an install, a removal, a plugin starting or
  stopping).

## 0.5.27 - 2026-10-21

### Fixed
- **Screen 0.3.9: the channel button and its light agree with the pages.** The desk
  toggles its selected channel with every press (the light follows), but the interface
  acted only on the press that selected one, so closing took two presses and the light
  was out of step. Every change counts now: a press opens the channel page, the next
  closes it; opening or closing a channel's pages by touch sets the desk's selection the
  way RØDE's screen does, so the light goes out when the pages close, and the selection
  is cleared when the interface ends.

## 0.5.26 - 2026-10-21

### Fixed
- **PlugRain's catalog entry has its title, description, icon and permission again.**
  The release tool that describes the packages came from rcplug 0.8.9, whose reader of
  a plugin's translations knew only title and description; PlugRain's translations also
  carry its options, so the tool left the whole entry out and the catalog showed a bare
  `plugrain`. The build uses rcplug 0.8.20 now (the plugins run with the same SDK).

### Changed
- All plugins are built with the rcplug SDK 0.8.20 (Backup 0.3.1, PlugRain 0.3.3,
  Screen 0.3.8, Sonar 0.7.1); nothing changes in what they do.

## 0.5.25 - 2026-10-21

### Fixed
- **Screen 0.3.7: the channel buttons above the faders work.** The desk reports a press
  (`systemChannelSelected`), and the model read it but never passed it on, so the button
  did nothing. It opens the strip's channel page now, as on RØDE's screen, and returns to
  the mixer when pressed again on that page.
- **Screen 0.3.7 and PlugRain 0.3.2 hold the black for two refreshes before they give
  the display up.** A page flip is done when its buffer starts to be scanned out, so the
  black frame could be cut off before it reached the panel's last rows (the page's right
  side), where the last page then flickered through at the next program's start.

### Changed
- The font atlas tool (`tools/fontatlas`, not shipped with any plugin) uses
  `golang.org/x/image` 0.41, which closes two advisories in its TIFF decoder; the tool
  only ever reads TrueType fonts.

## 0.5.24 - 2026-10-21

### Added
- **Screen 0.3.6: processing presets.** The channel page shows the current preset (or
  *custom settings*) above Depth, Sparkle and Punch; a tap lists the desk's presets and
  applies one, as on RØDE's screen. The preset format (`Name|value|…`) was read off a
  device's diagnostics: depth, sparkle and punch, APHEX, compressor, de-esser, noise gate,
  EQ and high-pass filter in a fixed order. Saving and deleting presets stay on RØDE's
  screen. The log names the presets now instead of printing their contents.

### Fixed
- **Screen 0.3.6** no longer crashes when rcplug starts it before the console is up (the
  headphone-knob watch asked the desk before it had loaded; the supervisor started it
  again, so the screen came a second late). Handing the screen to PlugRain is done off the
  interface's loop, so the plugin ends cleanly at once instead of being killed after three
  seconds of black screen (with rcplug 0.8.18 too). On ending it leaves the display black:
  the buffers are cleared and shown once more, so nothing of the last page flickers
  through when the next program takes the panel.
- **PlugRain 0.3.1** leaves the display black on ending the same way: the glitch on the
  right side of the screen when the Screen plugin came back was the last rain frame
  showing through the memory the new buffers landed in.

## 0.5.23 - 2026-10-20

### Added
- **PlugRain 0.3.0** joins the catalog as the demo of a plugin with the device's screen:
  green rain and the proof of root for a few minutes, then the screen goes back. It has
  a tile on the **Screen** plugin's rcplug page: a tap hands the display over to it
  without RØDE's screen in between, and when PlugRain ends the Screen interface comes
  back (rcplug 0.8.18). The tile appears for every plugin with the `screen` permission.
- **Screen 0.3.5** logs the device's processing presets whole (their format is still
  being learned) and documents the demo tiles.

## 0.5.22 - 2026-10-20

### Changed
- **Screen 0.3.4:** the configuration menu shows the device's own *Firmware update
  available* notice; the sound pad editor's play button previews the pad
  (`padButtonPreview`) instead of playing it into the mix; the README notes that with
  rcplug 0.8.17 the plugin comes back after a reboot once started.

## 0.5.21 - 2026-10-20

### Changed
- **Screen 0.3.3:** while a Bluetooth device is connected, its output's level ring sits
  beside the monitor's in the status bar (drag to set, as on RØDE's screen); a tap on a
  pad tile of the mixer opens that pad (its options, or the new-pad page for an empty
  slot) instead of playing it — the hardware buttons play; the equaliser's gain/Q
  choice is kept where RØDE's screen keeps it (`GUI.eqParamMode…`), so both agree.

## 0.5.20 - 2026-10-20

### Added
- **Plugin pages on the device's screen.** Sonar 0.7.0 and Backup 0.3.0 describe a
  page for the `screen` plugin (`GET /screen?lang=`, changes by `POST /screen`): Sonar
  shows what it listens to, the direction, level and distance, lets the source,
  smoothing and left/right swap be set and the distance calibrated; Backup shows how
  many backups there are and when the last one was made, and backs up rcplug's
  settings on a tap. **Screen 0.3.2** draws such pages with live values (the page is
  fetched about twice a second while shown), choices as option rows, meters as bars,
  asks the plugin in the screen's language, and documents the format in its README.

## 0.5.19 - 2026-10-20

### Added
- **Screen 0.3.1**, more of RØDE's screen: turning a headphone knob shows the level
  for a moment (and the popup's slider sets it); the channel page follows the source —
  USB, chat, USB 2 and Bluetooth get a **Routing** button to their own output's mix,
  the Effects tab appears for microphones only, and the SMART pads' channel shows the
  ducking depth instead of processing; the equaliser's **Q** switch swaps the
  frequency knobs for the bands' widths; Outputs gains a **Bluetooth** page (level,
  mute, auto-mute); the Shows page names the current show; Date & Time gets the
  daylight-saving switch and Brightness the Auto switch; the REC button follows the
  device's REC button setting (press pauses / hold stops, or the other way round); while
  the device is in transfer mode the screen says so instead of showing a silent mixer.

## 0.5.18 - 2026-10-20

### Added
- **Screen 0.3.0** edits the SMART pads and the custom mixes, as RØDE's screen does:
  - The pad menu shows a bank's six pads in the device's two columns with their names
    beside them; a tap opens the pad's options (edit, rename with the keyboard, colour
    from the twelve-colour picker, clear after a question), an empty slot the new-pad
    page (sound, effects, mixer action, MIDI trigger). The editors: a sound's trim and
    fades on four handles, play mode, loop, replay and level; an effects pad's input,
    latching and its own effects rack; a mixer action's latching, mode (censor, trash
    talk, fade with its times and the host, back channel with its members, ducking with
    the depth); a trigger's mode, type, control, channel, values and what is sent. The
    plugin never records on the device, so sounds come from the RØDE app.
  - A **custom mix** (Routing → output → Custom) shows one fader per strip with its
    meter and the link button under it: linked sends follow the channel fader (marked
    ▸◂), dragging frees a send, the button cycles linked → unlinked → muted.
  - Needs **rcplug 0.8.16** for these writes: the host now lets the plugin that stands
    in for RØDE's screen write at the console's expert level. On older hosts the
    settings beyond the basic rules (metering, haptics, the clock, pads, …) are refused.

### Fixed
- **Screen 0.3.0:** a strip's gain editor follows the kind of input (microphones:
  gain; USB, chat, Bluetooth and calls: level), the effects rack opens the strip's own
  slot (the device keeps one per strip, not per source), the output delay knob covers
  its 0–2 s range, pads play from every bank (the buttons act on the bank shown, which
  is selected first), the storage indicator ignores the pads' internal volume, the
  mixer's pads run down the columns as the device numbers them, and values pulled from
  the desk show on a page's first frame (some labels stayed empty before).

## 0.5.17 - 2026-10-20

### Fixed
- **Screen 0.2.2:** the button above each physical fader opens that strip's channel
  page, as on RØDE's screen (pressed again there, it returns to the mixer; another
  strip's button switches the page). The third test on a Duo confirmed the touch range
  and the knob's detents of 0.2.1.

## 0.5.16 - 2026-10-20

### Fixed
- **Screen 0.2.1**, after the second test on a Duo: the main knob counts detents as the
  device sends them (one to four a signal; the monitor moves in finer steps and a turn
  moves the focus by one control), the touchscreen's own pixel range is used (taps on
  the header buttons landed too low), and the strip meters fall smoothly between the
  console's readings instead of jumping. The channel page gains the **Effects** rack:
  reverb (level, low and high cut, room size), echo, megaphone, robot, voice disguise
  and pitch shift, on the source's own effects slot.

## 0.5.15 - 2026-10-20

### Changed
- **Screen 0.2.0** is rebuilt to the device's own layout and sizes after the first
  test on a RØDECaster Duo: the mixer with dBFS scales, peak hold and the SMART pads
  beside the strips, the channel page with gain, meter and target zone, the
  processors as the rack with response curves, tiles and option cards on the menus.
  The main knob works: it sets the monitor level, a press moves it on to the virtual
  faders, and on the menus it steps through the controls, presses them and adjusts
  values; holding it goes back. Channel assignment only changes with the green
  tick (a tap on the list used to write at once). The device's translations load
  (the first release ran in English), and the storage indicator, the recording
  timer and the Broadcast gain-reduction meters are shown.

## 0.5.14 - 2026-10-20

### Fixed
- **Screen 0.1.1** shows its title, description, icon and permissions in the
  catalog again and installs from it: its manifest no longer translates the option
  title, which the release tool could not read and so left out the plugin's whole
  catalog card.

## 0.5.13 - 2026-10-20

### Added
- **Screen 0.1.0** is an alternative interface on the device's own touchscreen: a
  mixer with channel, processing and assignment pages, SMART pads, the menus and
  settings, output routing, and pages for rcplug and other plugins — drawn to look and
  work like RØDE's own screen. It reads RØDE's icons and text from the device at run
  time (no RØDE files are shipped) and controls the desk through the console with
  optimistic, coalesced writes. RØDE's screen returns with the “Back to RØDE” button
  or whenever the plugin stops, so the desk is never left without a screen. It starts
  from the dashboard (no autostart yet) and needs rcplug 0.8.11 or later for the
  screen handover.

## 0.5.12 - 2026-10-19

### Added
- **Backup 0.2.3** lists RØDE's services that a restore switches on or off, as *RØDE
  service CallMe: switch off* (rcplug 0.8.11 keeps them in its settings).

## 0.5.11 - 2026-10-19

### Fixed
- **Sonar 0.6.5:** a long state (an error while switching sources) no longer makes
  the page wider than a phone's screen: the pill shows the start, its tooltip the
  whole text. *Switch them on* opens the dashboard's audio settings directly, and the
  radar's “far” label no longer runs into the 90° mark on small screens.
- **Backup 0.2.2:** the buttons for installing missing plugins sit under the text
  again, and *Choose a file…* shows the keyboard focus.
- Both use rcplug 0.8.9's shared look, which pages now load fresh after an update.

## 0.5.10 - 2026-10-19

### Changed
- **Sonar 0.6.4 and Backup 0.2.1 in rcplug's new look:** both pages use the shared
  design of rcplug 0.8.8 (`sdk/ui`, served by each plugin under `ui/`), so they read as
  part of the dashboard; inside it they leave out their own title. Building them needs
  rcplug 0.8.8; on the device Sonar runs with rcplug 0.6.11 or later, Backup with 0.8.6
  or later, as before.

## 0.5.9 - 2026-10-18

### Changed
- **Backup 0.2.0** backs up rcplug's settings only: which plugins run, their
  options, updates and catalogs. The console and the desk's settings are no longer
  part of it: they go with the device (RØDE's shows keep them), and a firmware update
  can change what their thousands of values mean. Backups made with 0.1.0 still
  restore their rcplug part; the rest is ignored, and dropped when such a file is
  uploaded. With rcplug 0.8.6, a restore also switches on again a plugin that its
  autostart had started and that was switched off by hand since.

## 0.5.8 - 2026-10-18

### Added
- **Backup 0.1.0**, a new plugin: backs up rcplug's settings (plugins switched on,
  their options, updates, catalogs, the console's control level and write protection)
  and the desk's settings (a few thousand: channels and their processing,
  routing, inputs, SMART pads, effects, outputs, the device's own settings), keeps up
  to 50 backups on the device, and downloads and uploads them as files. **Restore…**
  first shows what would change, with readable names, and restores only after
  confirmation: the desk through the console's checks (switching to the expert level
  for it and back, asking for the PIN if one is set), then rcplug's settings, with a
  button to install plugins the device lacks from the catalog. Never in a backup:
  passwords and the PIN, recordings, shows, network settings and the volume of
  headphones and monitors. Needs rcplug 0.8.5.

## 0.5.7 - 2026-10-18

### Changed
- **Sonar 0.6.3**, simpler and smoother:
  - **One list** chooses what Sonar hears, by name (Mic 1 + Mic 2, USB 1 from the
    computer, chat, calls, RØDE wireless, the mix to the computer, the mixer's
    outputs, the test signal), grouped into inputs, outputs and other audio. Choosing
    switches at once, without assigning sources to channels and without a question.
    Opened from a channel of the console, Sonar listens to where it comes in right
    away (for a single microphone, the pair it belongs to).
  - The radar's **sweep line is gone**; the dot glides between readings instead of
    jumping, and the numbers change five times a second.
  - If the host's audio taps are switched off, Sonar says so with a link to switch
    them on.
  - The loopback of rcplug's own audio bridge is no longer offered as a source.

## 0.5.6 - 2026-10-18

### Changed
- **Sonar 0.6.2:** proposes the right JACK pair for USB 1, USB 2, the chat and the
  calls too, as a real Duo (firmware 1.7.6) wires them, besides the microphone inputs;
  the README lists the desk's JACK ports. The bridge through the host's taps was
  measured on a Duo: about 1 % of the device with rcplug 0.7.4.
- A plugin's binary reports its own version (from its `plugin.json`) and the release of
  this repository it came with; before, it reported only the latter.

## 0.5.5 - 2026-10-18

### Fixed
- **Sonar 0.6.1:** on the desk the source list no longer offers ALSA devices that can
  never be recorded there (the desk's own sound card and the USB audio to the computer
  belong to RØDE's mixer; there is no PulseAudio); the desk's audio comes through JACK.

## 0.5.4 - 2026-10-18

### Changed
- **Sonar 0.6.0**, after the first tests on a real Duo:
  - **Distance:** estimated from the level and from how much both microphones hear
    the same signal (direct sound against the room's reverberation), with a
    calibration (*Near*, *Far*) for the room. On the radar, near is now at the centre
    and far at the outer ring; before, a louder sound was drawn further out, which
    read as "far". A *Distance* readout shows near, middle or far.
  - **What Sonar hears:** one card for the channel of the desk and its audio. A mono
    channel (one microphone) says that it has no direction and proposes a pair; for
    microphone channels the JACK pair of inputs 1+2 (or 3+4) is proposed. Choosing
    the audio without a channel moved under *Choose the audio directly*.
  - The radar says what a stereo signal cannot tell: front from back, and height.
  - The question before a JACK source describes the host's shared audio taps.

## 0.5.3 - 2026-10-18

### Fixed
- **Sonar 0.5.2:** Sonar's own version was not raised in 0.5.1 and 0.5.2, so devices
  with Sonar 0.5.0 were not offered the version that hears the desk through rcplug's
  shared audio taps. It is 0.5.2 now, built with the rcplug 0.7.1 module (its console
  requests carry the plugin's token).
- The release workflow refuses a release in which a plugin changed (or the module's
  `go.mod`) without a new version in its `plugin.json` (`make check-versions`).

## 0.5.2 - 2026-10-18

### Changed
- Releases are signed with `rcplug-release` from rcplug 0.7.1, which records each
  plugin's title, description, icon, permissions and translations in `release.json`:
  rcplug's plugin catalog (Plugins → *More plugins*) shows Sonar with them, and its
  permission `audio`.
- Requires the rcplug module 0.7.1 (`go.mod`).

## 0.5.1 - 2026-10-18

### Changed
- Sonar hears the desk through rcplug's **shared audio taps** (rcplug 0.6.11 and
  later): one bridge for every plugin, switched on in the dashboard (System → *Audio
  taps*). While they are off, Sonar asks for them instead of starting its own client in
  RØDE's audio graph; on older hosts it still bridges JACK itself.
- Requires the rcplug module 0.6.11 (`go.mod`).

## 0.5.0 - 2026-10-18

First release as its own repository. Sonar was part of rcplug up to 0.4.0.

### Fixed
- Switching away from an ALSA or JACK source could hang Sonar's capture until a
  restart (the radar then showed "live" with a frozen bearing): `arecord` was waited
  for twice.
- ALSA and JACK sources now give about 30 radar updates per second instead of 5 to 8
  (`arecord` uses 20 ms periods instead of the device's default).
- A source that fails shows `arecord`'s reason on the page instead of "EOF"; an empty
  WAV file and one with another sample rate than Sonar's are refused with a clear
  message.
- Out-of-range `--rate`, `--block`, `--smoothing` and `--noise-floor` stop Sonar at
  start with a message instead of crashing or spinning later.
- The retry pause of a failing source stays at most 10 s, as documented.
- Opening Sonar from a console strip that has no source selects that strip instead of
  silently keeping the previous channel.

### Added
- Automation on GitHub: a development build (pre-release *dev*) on every push to
  `main`, one-click releases (*Actions → Release → Run workflow*), `govulncheck` and
  CodeQL, Dependabot for the GitHub Actions and the rcplug SDK, and `make docs-check`.
  Everything that needs rcplug's code starts once rcplug is public.
- Sonar updates from the dashboard: its plugin.json names this repository as its
  update source, and the release workflow signs every release (`release.json` +
  `release.json.sig`, checked by rcplugd before it offers or installs an update).
- Sonar lists the console's channels and remembers which audio source carries each.
- Sonar can listen to the desk on the Duo, opt-in: a `jack:<port>,<port>` source.
  The firmware runs the whole desk as a JACK graph, and `jackd` holds the card's only
  capture device, so `arecord` cannot reach it. Sonar bridges two JACK ports with
  the firmware's own tools (snd-aloop as card `RcplugLoop`, `alsa_out`,
  `jack_connect`, `arecord`), lists the stereo pairs JACK offers and asks before
  adding its client to RØDE's audio graph. Tested with the firmware's JACK tools and
  end to end with JACK2 and a real ALSA loopback; not yet on a real Duo.

### Changed
- Sonar looks like the rest of the interface.
