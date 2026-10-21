package web

import "net/http"

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(pageHTML))
}

// pageHTML is the browser page: the picture, a bar with the knob's actions
// and the headphone knobs, and the stream behind it. It speaks the
// browser's language for its own few words; the interface itself speaks
// the desk's.
const pageHTML = `<!doctype html>
<html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Screen</title>
<style>
 html,body{margin:0;background:#0c0c0c;color:#ddd;font-family:system-ui,sans-serif;height:100%;overflow:hidden}
 .wrap{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:10px;height:100%;padding:10px;box-sizing:border-box}
 #screen{background:#000;touch-action:none;user-select:none;border-radius:10px;box-shadow:0 8px 40px #000;max-width:100%;max-height:calc(100% - 70px);aspect-ratio:1280/720;width:min(100%,calc((100vh - 90px)*1280/720))}
 .bar{display:flex;gap:8px;flex-wrap:wrap;justify-content:center;align-items:center}
 .bar button{background:#2a2a2a;color:#eee;border:0;border-radius:10px;padding:10px 16px;font-size:15px;cursor:pointer}
 .bar button:active{background:#3bd33b;color:#000}
 .bar .lbl{color:#8a8a8a;font-size:13px;margin:0 4px}
 #state{color:#8a8a8a;font-size:12px;min-width:90px;text-align:right}
</style></head><body>
<div class="wrap">
 <canvas id="screen" width="1280" height="720"></canvas>
 <div class="bar">
  <span class="lbl" data-de="Hauptregler" data-en="Main knob">Main knob</span>
  <button id="kl">−</button><button id="kr">+</button>
  <button id="kp" data-de="Drücken" data-en="Press">Press</button>
  <button id="kh" data-de="Halten" data-en="Hold">Hold</button>
  <span class="lbl" data-de="Kopfhörer" data-en="Headphones">Headphones</span>
  <button id="h1">1</button><button id="h2">2</button>
  <span id="state"></span>
 </div>
</div>
<script>
(() => {
 const de = (navigator.language||'').toLowerCase().startsWith('de');
 document.querySelectorAll('[data-de]').forEach(e => { e.textContent = de ? e.dataset.de : e.dataset.en; });
 const cv = document.getElementById('screen'), ctx = cv.getContext('2d'), state = document.getElementById('state');
 let ws = null, down = false, backoff = 500;
 const words = de ? {on:'verbunden', off:'getrennt', conn:'verbinde…'} : {on:'connected', off:'disconnected', conn:'connecting…'};
 function send(m){ if (ws && ws.readyState === 1) ws.send(JSON.stringify(m)); }
 function connect(){
  state.textContent = words.conn;
  const url = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + location.pathname.replace(/\/$/, '') + '/ws';
  ws = new WebSocket(url); ws.binaryType = 'arraybuffer';
  ws.onopen = () => { state.textContent = words.on; backoff = 500; };
  ws.onclose = () => { state.textContent = words.off; setTimeout(connect, backoff); backoff = Math.min(backoff * 2, 8000); };
  ws.onmessage = async (ev) => {
   if (!(ev.data instanceof ArrayBuffer)) return;
   const v = new DataView(ev.data);
   const x = v.getUint16(0), y = v.getUint16(2), w = v.getUint16(4), h = v.getUint16(6);
   const blob = new Blob([ev.data.slice(8)], {type: 'image/png'});
   try { const bmp = await createImageBitmap(blob); ctx.drawImage(bmp, x, y, w, h); bmp.close && bmp.close(); } catch (e) {}
  };
 }
 function pos(e){ const r = cv.getBoundingClientRect(); return {x: Math.round((e.clientX - r.left) / r.width * 1280), y: Math.round((e.clientY - r.top) / r.height * 720)}; }
 cv.addEventListener('pointerdown', e => { cv.setPointerCapture(e.pointerId); down = true; const p = pos(e); send({t:'touch', p:'down', x:p.x, y:p.y}); e.preventDefault(); });
 cv.addEventListener('pointermove', e => { if (!down) return; const p = pos(e); send({t:'touch', p:'move', x:p.x, y:p.y}); e.preventDefault(); });
 const up = e => { if (!down) return; down = false; const p = pos(e); send({t:'touch', p:'up', x:p.x, y:p.y}); e.preventDefault(); };
 cv.addEventListener('pointerup', up); cv.addEventListener('pointercancel', up);
 cv.addEventListener('wheel', e => { e.preventDefault(); send({t:'knob', d: e.deltaY > 0 ? 1 : -1}); }, {passive: false});
 cv.addEventListener('contextmenu', e => e.preventDefault());
 document.getElementById('kl').onclick = () => send({t:'knob', d:-1});
 document.getElementById('kr').onclick = () => send({t:'knob', d:1});
 document.getElementById('kp').onclick = () => send({t:'press'});
 document.getElementById('kh').onclick = () => send({t:'hold'});
 document.getElementById('h1').onclick = () => send({t:'hp', i:0});
 document.getElementById('h2').onclick = () => send({t:'hp', i:1});
 document.addEventListener('keydown', e => {
  if (e.key === 'ArrowRight' || e.key === 'ArrowUp') send({t:'knob', d:1});
  else if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') send({t:'knob', d:-1});
  else if (e.key === 'Enter') send({t:'press'});
  else return;
  e.preventDefault();
 });
 connect();
})();
</script>
</body></html>
`
