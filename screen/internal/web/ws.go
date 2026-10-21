package web

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// A small WebSocket server side (RFC 6455): the handshake, masked frames
// from the browser, unmasked frames to it, ping/pong and close. Enough for
// one picture stream and a few short messages, without a dependency.

type wsConn struct {
	conn net.Conn
	rw   *bufio.ReadWriter
	wmu  sync.Mutex
}

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// upgrade answers the browser's handshake and takes the connection over.
func upgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return nil, errors.New("not a websocket request")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("missing Sec-WebSocket-Key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("the connection cannot be taken over")
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	return &wsConn{conn: conn, rw: rw}, nil
}

const (
	opText   = 1
	opBinary = 2
	opClose  = 8
	opPing   = 9
	opPong   = 10
)

// read returns the next data message (text or binary); it answers pings
// and reports a close as io.EOF. Fragmented messages are joined.
func (c *wsConn) read(max int) (op byte, data []byte, err error) {
	var msg []byte
	var msgOp byte
	for {
		h := make([]byte, 2)
		if _, err := io.ReadFull(c.rw, h); err != nil {
			return 0, nil, err
		}
		fin := h[0]&0x80 != 0
		o := h[0] & 0x0f
		masked := h[1]&0x80 != 0
		n := int(h[1] & 0x7f)
		switch n {
		case 126:
			b := make([]byte, 2)
			if _, err := io.ReadFull(c.rw, b); err != nil {
				return 0, nil, err
			}
			n = int(binary.BigEndian.Uint16(b))
		case 127:
			b := make([]byte, 8)
			if _, err := io.ReadFull(c.rw, b); err != nil {
				return 0, nil, err
			}
			n = int(binary.BigEndian.Uint64(b))
		}
		if n > max || n < 0 {
			return 0, nil, errors.New("websocket message too large")
		}
		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(c.rw, mask[:]); err != nil {
				return 0, nil, err
			}
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(c.rw, payload); err != nil {
			return 0, nil, err
		}
		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}
		switch o {
		case opClose:
			_ = c.write(opClose, nil)
			return 0, nil, io.EOF
		case opPing:
			if err := c.write(opPong, payload); err != nil {
				return 0, nil, err
			}
			continue
		case opPong:
			continue
		case opText, opBinary:
			msgOp = o
			msg = append(msg, payload...)
		case 0: // a continuation
			msg = append(msg, payload...)
		default:
			return 0, nil, errors.New("unknown websocket opcode")
		}
		if fin {
			return msgOp, msg, nil
		}
		if len(msg) > max {
			return 0, nil, errors.New("websocket message too large")
		}
	}
}

// write sends one message (unmasked, as a server does).
func (c *wsConn) write(op byte, data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	h := []byte{0x80 | op}
	switch n := len(data); {
	case n < 126:
		h = append(h, byte(n))
	case n < 1<<16:
		h = append(h, 126, byte(n>>8), byte(n))
	default:
		h = append(h, 127)
		h = binary.BigEndian.AppendUint64(h, uint64(n))
	}
	if _, err := c.rw.Write(h); err != nil {
		return err
	}
	if _, err := c.rw.Write(data); err != nil {
		return err
	}
	return c.rw.Flush()
}

func (c *wsConn) close() { _ = c.conn.Close() }
