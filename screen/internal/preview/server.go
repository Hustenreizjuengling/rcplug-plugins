package preview

import (
	"image"
	"image/png"
	"net/http"
	"strconv"
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Server shows a running App in a web browser: it renders frames to PNG and
// takes pointer events back, so the interface can be driven without a device.
type Server struct {
	app *app.App
	mu  sync.Mutex
}

// NewServer wraps an app for the browser.
func NewServer(a *app.App) *Server { return &Server{app: a} }

// Handler is the preview's HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.page)
	mux.HandleFunc("/frame.png", s.frame)
	mux.HandleFunc("/touch", s.touch)
	return mux
}

func (s *Server) frame(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	img := s.app.Frame().ToNRGBA()
	s.mu.Unlock()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_ = png.Encode(w, img)
}

func (s *Server) touch(w http.ResponseWriter, r *http.Request) {
	x, _ := strconv.Atoi(r.URL.Query().Get("x"))
	y, _ := strconv.Atoi(r.URL.Query().Get("y"))
	var phase ui.TouchPhase
	switch r.URL.Query().Get("phase") {
	case "down":
		phase = ui.Down
	case "move":
		phase = ui.Move
	default:
		phase = ui.Up
	}
	s.mu.Lock()
	s.app.Touch(phase, image.Pt(x, y))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(previewHTML))
}

const previewHTML = `<!doctype html>
<html lang="de"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>rcplug – Bildschirm-Vorschau</title>
<style>
 html,body{margin:0;background:#0c0c0c;color:#ddd;font-family:system-ui,sans-serif;height:100%}
 .wrap{display:flex;flex-direction:column;align-items:center;gap:10px;padding:14px}
 h1{font-size:15px;font-weight:600;color:#9a9a9a;margin:4px 0}
 #frame{width:1280px;max-width:98vw;aspect-ratio:1280/720;touch-action:none;
   border:1px solid #333;border-radius:8px;box-shadow:0 8px 40px #000;user-select:none}
 .hint{font-size:12px;color:#666}
</style></head>
<body><div class="wrap">
 <h1>RØDECaster Duo · alternative Oberfläche (Vorschau)</h1>
 <img id="frame" src="/frame.png" draggable="false">
 <div class="hint">Klicken und ziehen wie am Touchscreen. Live-Vorschau ohne Gerät.</div>
</div>
<script>
const img = document.getElementById('frame');
const W = 1280, H = 720;
let down = false;
function at(e){
  const r = img.getBoundingClientRect();
  const x = Math.round((e.clientX - r.left) / r.width  * W);
  const y = Math.round((e.clientY - r.top)  / r.height * H);
  return {x: Math.max(0, Math.min(W-1, x)), y: Math.max(0, Math.min(H-1, y))};
}
async function send(phase, p){ await fetch('/touch?phase='+phase+'&x='+p.x+'&y='+p.y, {method:'POST'}); refresh(); }
let busy = false, again = false;
async function refresh(){
  if (busy){ again = true; return; }
  busy = true;
  await new Promise(r => { const n = new Image(); n.onload = ()=>{ img.src = n.src; r(); }; n.onerror = r; n.src = '/frame.png?t='+Date.now(); });
  busy = false;
  if (again){ again = false; refresh(); }
}
img.addEventListener('pointerdown', e=>{ down=true; img.setPointerCapture(e.pointerId); send('down', at(e)); e.preventDefault(); });
img.addEventListener('pointermove', e=>{ if(down) send('move', at(e)); });
img.addEventListener('pointerup',   e=>{ down=false; send('up', at(e)); });
setInterval(refresh, 250); // pick up live changes (meters, clock)
</script>
</body></html>`
