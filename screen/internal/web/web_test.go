package web

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/preview"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A browser connects, gets the whole page as the first picture, touches,
// and its interface ends with the connection.
func TestSessionStreamsPictures(t *testing.T) {
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	texts := lang.New("de", nil)
	s := New(ctx, model, func() *lang.Texts { return texts }, nil, preview.NewPlugins(), nil)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	req := "GET /ws HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, nil)
	if err != nil || resp.StatusCode != 101 || resp.Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("handshake: %v %v", err, resp)
	}
	// The first message: the whole page as a PNG with its place.
	op, data := readFrame(t, r)
	if op != opBinary || len(data) < 8 {
		t.Fatalf("first message: op %d, %d bytes", op, len(data))
	}
	x, y, w, h := binary.BigEndian.Uint16(data), binary.BigEndian.Uint16(data[2:]), binary.BigEndian.Uint16(data[4:]), binary.BigEndian.Uint16(data[6:])
	if x != 0 || y != 0 || w != pageW || h != pageH {
		t.Fatalf("first picture at %d,%d %dx%d", x, y, w, h)
	}
	img, err := png.Decode(bytes.NewReader(data[8:]))
	if err != nil || img.Bounds().Dx() != pageW {
		t.Fatalf("not a page PNG: %v", err)
	}
	// A touch on the settings cog (top left of the mixer) changes the page.
	writeFrame(t, conn, opText, []byte(`{"t":"touch","p":"down","x":1215,"y":57}`))
	writeFrame(t, conn, opText, []byte(`{"t":"touch","p":"up","x":1215,"y":57}`))
	op, data = readFrame(t, r)
	if op != opBinary || len(data) < 8 {
		t.Fatalf("after the touch: op %d, %d bytes", op, len(data))
	}
	if s.Sessions() != 1 {
		t.Fatalf("sessions = %d", s.Sessions())
	}
	conn.Close()
	for i := 0; i < 50 && s.Sessions() != 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if s.Sessions() != 0 {
		t.Fatal("the interface did not end with the connection")
	}
}

func readFrame(t *testing.T, r *bufio.Reader) (byte, []byte) {
	t.Helper()
	h := make([]byte, 2)
	if _, err := r.Read(h[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(h[1:]); err != nil {
		t.Fatal(err)
	}
	n := int(h[1] & 0x7f)
	switch n {
	case 126:
		b := make([]byte, 2)
		readFull(t, r, b)
		n = int(binary.BigEndian.Uint16(b))
	case 127:
		b := make([]byte, 8)
		readFull(t, r, b)
		n = int(binary.BigEndian.Uint64(b))
	}
	data := make([]byte, n)
	readFull(t, r, data)
	return h[0] & 0x0f, data
}

func readFull(t *testing.T, r *bufio.Reader, b []byte) {
	t.Helper()
	for got := 0; got < len(b); {
		n, err := r.Read(b[got:])
		if err != nil {
			t.Fatal(err)
		}
		got += n
	}
}

// writeFrame sends a masked client frame.
func writeFrame(t *testing.T, conn net.Conn, op byte, data []byte) {
	t.Helper()
	mask := [4]byte{1, 2, 3, 4}
	f := []byte{0x80 | op, 0x80 | byte(len(data))}
	f = append(f, mask[:]...)
	for i, b := range data {
		f = append(f, b^mask[i%4])
	}
	if _, err := conn.Write(f); err != nil {
		t.Fatal(err)
	}
}
