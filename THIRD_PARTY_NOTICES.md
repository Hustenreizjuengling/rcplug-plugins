# Third-party notices

The plugins use only the Go standard library and rcplug's SDK, and ship no third-party
code, except the Go fonts below. On the device, Sonar's optional JACK source runs tools that are part of the
device's own firmware (`jack_lsp`, `jack_connect`, `alsa_out`, `arecord`, the
`snd-aloop` kernel module); they are not part of this repository.

## Go fonts

Screen demo draws its text with glyph atlases made from the Go fonts (Go Regular and
Go Bold) by [tools/fontatlas](tools/fontatlas/main.go), which reads them from
`golang.org/x/image` (only that tool, not the plugins, depends on it). The atlases are
in [screendemo/font](screendemo/font) and built into the plugin's program.

```
Copyright (c) 2016 Bigelow & Holmes Inc.. All rights reserved.

Distribution of this font is governed by the following license. If you do not
agree to this license, including the disclaimer, do not distribute or modify
this font.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

	* Redistributions of source code must retain the above copyright notice,
	  this list of conditions and the following disclaimer.

	* Redistributions in binary form must reproduce the above copyright notice,
	  this list of conditions and the following disclaimer in the documentation
	  and/or other materials provided with the distribution.

	* Neither the name of Google Inc. nor the names of its contributors may be
	  used to endorse or promote products derived from this software without
	  specific prior written permission.

DISCLAIMER: THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Trademarks

RØDE, RØDECaster and RØDECaster Duo are trademarks of Freedman Electronics
Pty Ltd. rcplug is not affiliated with, endorsed by or supported by RØDE.
They are used here only to describe what rcplug works with.
