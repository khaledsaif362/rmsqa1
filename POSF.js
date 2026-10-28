/* =========================================================
   CONSTANTS
========================================================= */
const MEMBER = "6538";

/* ---------------------------------------------------------
   SERVER BHAVCOPY (rms_api.go, listens on :8090)

   GET /api/nse-fo-bhavcopy  -> latest NSE F&O CSV
   GET /api/bse-fo-bhavcopy  -> latest BSE F&O CSV
   The API runs on the same host as this page, port 8090.
   (rms_api.go only allows CORS from http://172.16.101.74:8080
   and does not send Allow-Credentials, so no cookies are sent.)
--------------------------------------------------------- */
const SERVER = {
  base: location.protocol + "//" + location.hostname + ":8090",
  bhavcopy: [
    { label: "NSE FO", path: "/api/nse-fo-bhavcopy" },
    { label: "BSE FO", path: "/api/bse-fo-bhavcopy" }
  ]
};

const FO_HEADER = ("Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4").split(",");

const FILES = [
  ["Position_ICCL_FO_0_CM_6538_2026", "ICCL CM"],
  ["Position_ICCL_FO_0_TM_6538_2026", "ICCL TM"],
  ["Position_NCL_FO_0_CM_6538_2026",  "NCL CM"],
  ["Position_NCL_FO_0_TM_6538_2026",  "NCL TM"]
];

/* sample set: [symbol, instrument, option type, lots] */
const SAMPLE = [
  ["SBIN",      "STF", "",   1],
  ["BANKNIFTY", "IDF", "",   1],
  ["NIFTY",     "IDO", "CE", 1],
  ["TCS",       "STO", "CE", 1],
  ["SENSEX",    "IDF", "",   5],
  ["SENSEX",    "IDO", "CE", 5],
  ["BANKEX",    "IDO", "PE", 5]
];

let bhav = [];          // parsed Bhavcopy rows
let clients = [];       // uploaded client list
let positions = [];     // [{key, r, lots, side}]
let mode = "manual";

/* =========================================================
   HELPERS
========================================================= */
const $ = id => document.getElementById(id);
const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c]));
const sym = r => String(r.TckrSymb || "").trim().toUpperCase();
const tp  = r => String(r.FinInstrmTp || "").trim().toUpperCase();

function csvNumber(v){
  const n = parseFloat(String(v ?? "").replace(/,/g, ""));
  return Number.isFinite(n) ? n : 0;
}
function fmt(n){ return Number(n).toLocaleString("en-IN", {maximumFractionDigits: 2}); }

/* normalise yyyy-mm-dd / yyyymmdd / dd-mm-yyyy / dd/mm/yyyy to yyyymmdd */
function normDate(v){
  const s = String(v || "").trim();
  if (!s) return "";
  const m = s.match(/^(\d{2})[-\/](\d{2})[-\/](\d{4})$/);
  if (m) return m[3] + m[2] + m[1];
  return s.replace(/[-\/]/g, "");
}
function prettyDate(v){
  const d = normDate(v);
  if (d.length !== 8) return v || "";
  const mon = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"][parseInt(d.slice(4,6),10)-1];
  return d.slice(6,8) + "-" + mon + "-" + d.slice(0,4);
}
function todayStr(){
  const n = new Date();
  return n.getFullYear() + String(n.getMonth() + 1).padStart(2, "0") + String(n.getDate()).padStart(2, "0");
}
function priceOf(r){ return csvNumber(r.SttlmPric || r.ClsPric || r.LastPric || r.PrvsClsgPric); }
function isOpt(r){ return tp(r).endsWith("O"); }
function contractKey(r){
  return [sym(r), tp(r), normDate(r.XpryDt), csvNumber(r.StrkPric), String(r.OptnTp||"").trim()].join("|");
}
function contractLabel(r){
  let s = sym(r) + " · " + tp(r) + " · " + prettyDate(r.XpryDt);
  if (isOpt(r)) s += " · " + csvNumber(r.StrkPric) + " " + String(r.OptnTp||"").trim();
  return s;
}
function randomItem(a){ return a[Math.floor(Math.random() * a.length)]; }

function parseCSVLine(line){
  const out = []; let cur = "", q = false;
  for (let i = 0; i < line.length; i++){
    const ch = line[i];
    if (ch === '"' && line[i+1] === '"'){ cur += '"'; i++; continue; }
    if (ch === '"'){ q = !q; continue; }
    if (ch === "," && !q){ out.push(cur); cur = ""; continue; }
    cur += ch;
  }
  out.push(cur);
  return out.map(v => v.trim());
}

/* =========================================================
   CACHE (same IndexedDB the Position Files page uses)
========================================================= */
/* never let a stuck IndexedDB call freeze the page */
function withTimeout(promise, ms, fallback){
  return Promise.race([promise, new Promise(res => setTimeout(() => res(fallback), ms))]);
}

function cacheOpen(){
  return new Promise((resolve, reject) => {
    try{
      const req = indexedDB.open("posFilesCache", 1);
      req.onupgradeneeded = () => req.result.createObjectStore("kv");
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    }catch(e){ reject(e); }
  });
}
function cacheSet(k, v){
  return cacheOpen().then(db => new Promise(res => {
    const tx = db.transaction("kv", "readwrite");
    tx.objectStore("kv").put(v, k);
    tx.oncomplete = () => res(true); tx.onerror = () => res(false);
  })).catch(() => false);
}
function cacheGet(k){
  return cacheOpen().then(db => new Promise(res => {
    const rq = db.transaction("kv").objectStore("kv").get(k);
    rq.onsuccess = () => res(rq.result || null); rq.onerror = () => res(null);
  })).catch(() => null);
}
function cacheDelete(k){
  return cacheOpen().then(db => new Promise(res => {
    const tx = db.transaction("kv", "readwrite");
    tx.objectStore("kv").delete(k);
    tx.oncomplete = () => res(true); tx.onerror = () => res(false);
  })).catch(() => false);
}

/* =========================================================
   LOAD BHAVCOPY / CLIENTS
========================================================= */
/* parse one Bhavcopy file's text into row objects */
function parseBhavText(text){
  const lines = String(text || "").split(/\r?\n/).filter(x => x.trim());
  if (lines.length < 2) return [];
  const headers = parseCSVLine(lines[0]);
  return lines.slice(1).map(line => {
    const c = parseCSVLine(line), o = {};
    headers.forEach((h, i) => o[h] = c[i] ?? "");
    return o;
  }).filter(r => r.TckrSymb && r.XpryDt);
}

/* files = [{name, text}, ...] — NSE and BSE are merged into one list */
function applyBhavFiles(files, cached){
  const all = [];
  const names = [];
  files.forEach(f => {
    const rows = parseBhavText(f.text);
    if (rows.length){ rows.forEach(r => all.push(r)); names.push(f.name); }
  });

  if (!all.length){
    bhav = [];
    $("bhavStatus").textContent = "Invalid or empty Bhavcopy";
    return false;
  }
  bhav = all;

  $("bhavStatus").textContent =
    names.join(", ") + " · " + bhav.length.toLocaleString() + " contracts" + (cached ? " (saved)" : "");

  const syms = [...new Set(bhav.map(sym))].sort();
  $("underList").innerHTML = syms.map(n => '<option value="' + esc(n) + '">').join("");
  refreshManualOptions();
  return true;
}

function onBhavFile(input){
  const list = input.files ? [...input.files] : [];
  if (!list.length) return;
  Promise.all(list.map(f => new Promise((resolve, reject) => {
    const rd = new FileReader();
    rd.onload = e => resolve({name: f.name, text: String(e.target.result || "")});
    rd.onerror = () => reject(new Error("read"));
    rd.readAsText(f);
  }))).then(files => {
    if (applyBhavFiles(files, false)){
      cacheSet("bhavcopy", {
        name: files.map(f => f.name).join(", "),
        text: files[0].text,      /* first file kept here so the old Position Files page still works */
        files: files
      });
    }
  }).catch(() => alert("Could not read the Bhavcopy."));
}

function setClients(list, name, cached){
  clients = list;
  const on = clients.length > 0;
  $("clientStatus").textContent = on
    ? clients.length.toLocaleString() + " client(s) · " + name + (cached ? " (saved)" : "")
    : "";
  $("clientClear").style.display = on ? "inline-block" : "none";
  ["client1", "client2"].forEach(id => { $(id).disabled = on; if (on) $(id).value = ""; });
}

function onClientFile(input){
  const f = input.files && input.files[0];
  if (!f) return;
  const rd = new FileReader();
  rd.onload = e => {
    const list = String(e.target.result || "").split(/\r?\n/)
      .map(l => l.split(",")[0].trim().replace(/^"|"$/g, ""))
      .filter(Boolean);
    setClients(list, f.name, false);
    cacheSet("clients", {name: f.name, list});
  };
  rd.onerror = () => alert("Could not read the client list.");
  rd.readAsText(f);
}

function clearClients(){
  cacheDelete("clients");
  $("clientFile").value = "";
  setClients([], "", false);
}

/* fetch one Bhavcopy from the server; returns {name, text} or throws */
async function fetchBhavFromServer(item){
  const ctl = new AbortController();
  const timer = setTimeout(() => ctl.abort(), 10000);
  let res;
  try{
    res = await fetch(SERVER.base + item.path, {cache: "no-store", signal: ctl.signal});
  }catch(e){
    throw new Error(item.label + ": cannot reach " + SERVER.base + item.path + " (" + (e.name === "AbortError" ? "timed out" : "blocked or server down") + ")");
  }finally{
    clearTimeout(timer);
  }
  if (!res.ok) throw new Error(item.label + ": HTTP " + res.status);

  let name = item.label + " Bhavcopy";
  let text;
  const type = (res.headers.get("content-type") || "").toLowerCase();
  if (type.includes("json")){
    const j = await res.json();
    text = j.csv || j.text || j.data || "";
    if (j.name) name = j.name;
  } else {
    text = await res.text();
  }
  if (!/TckrSymb/.test(String(text).split(/\r?\n/)[0] || "")){
    throw new Error(item.label + ": response is not a Bhavcopy CSV");
  }
  return {name, text};
}

/* fetch NSE + BSE together; returns the files that worked */
async function fetchAllBhav(){
  const results = await Promise.allSettled(SERVER.bhavcopy.map(fetchBhavFromServer));
  const files = results.filter(r => r.status === "fulfilled").map(r => r.value);
  const errors = results.filter(r => r.status === "rejected").map(r => r.reason.message);
  return {files, errors};
}

/* Auto Upload: server first, saved copy as fallback */
async function autoUpload(){
  $("autoStatus").textContent = "Fetching Bhavcopy from server…";

  /* client list from the saved copy (never blocks the fetch) */
  withTimeout(cacheGet("clients"), 3000, null).then(c => {
    if (c && c.list && c.list.length) setClients(c.list, c.name || "saved list", true);
  });

  const {files, errors} = await fetchAllBhav();

  if (files.length && applyBhavFiles(files, false)){
    cacheSet("bhavcopy", {name: files.map(f => f.name).join(", "), text: files[0].text, files});
    let msg = "Bhavcopy fetched from server (" + files.length + "/" + SERVER.bhavcopy.length + ")";
    if (errors.length) msg += " — " + errors.join("; ");
    $("autoStatus").textContent = msg;
    return;
  }

  const b = await withTimeout(cacheGet("bhavcopy"), 3000, null);
  if (b && (b.files || b.text)){
    applyBhavFiles(b.files || [{name: b.name || "Bhavcopy", text: b.text}], true);
    $("autoStatus").textContent = "Server failed — using saved Bhavcopy. " + (errors[0] || "");
  } else {
    $("autoStatus").textContent = "Server fetch failed: " + (errors[0] || "no data") + " — choose files below";
  }
}

/* =========================================================
   PICK A CONTRACT FROM THE BHAVCOPY
   Nearest upcoming expiry. Futures preferred when no
   instrument is given; options use the middle strike.
========================================================= */
function pickContract(symbol, inst, optType){
  let rows = bhav.filter(r => sym(r) === symbol);
  if (inst) rows = rows.filter(r => tp(r) === inst);
  if (!rows.length) return null;

  if (!inst){
    const futs = rows.filter(r => !isOpt(r));
    if (futs.length) rows = futs;
  }

  const exps = [...new Set(rows.map(r => normDate(r.XpryDt)))].sort();
  const upcoming = exps.filter(e => e >= todayStr());
  const exp = (upcoming.length ? upcoming : exps)[0];
  rows = rows.filter(r => normDate(r.XpryDt) === exp);

  if (!rows.some(isOpt)) return rows[0];

  let opts = rows.filter(isOpt);
  const want = optType || "CE";
  const same = opts.filter(r => String(r.OptnTp || "").trim() === want);
  if (same.length) opts = same;
  opts.sort((a, b) => csvNumber(a.StrkPric) - csvNumber(b.StrkPric));
  return opts[Math.floor(opts.length / 2)];
}

/* =========================================================
   MODES
========================================================= */
function setMode(m){
  mode = m;
  ["sample", "manual", "bulk"].forEach(k => $("mb_" + k).classList.toggle("active", k === m));
  $("sampleBox").style.display = m === "sample" ? "block" : "none";
  $("manualBox").style.display = m === "manual" ? "block" : "none";
  $("bulkBox").style.display   = m === "bulk"   ? "block" : "none";
  $("posWrap").style.display   = m === "bulk"   ? "none"  : "block";
  if (m === "sample") loadSample();
}

function loadSample(){
  if (!bhav.length){
    $("sampleStatus").textContent = "Upload the Bhavcopy first.";
    alert("Please upload the Bhavcopy first.");
    return;
  }
  positions = [];
  const missing = [];
  SAMPLE.forEach(([s, inst, opt, lots]) => {
    const r = pickContract(s, inst, opt);
    if (!r){ missing.push(s + " " + inst); return; }
    addToList(r, lots, "B");
  });
  $("sampleStatus").textContent =
    positions.length + " sample position(s) loaded" +
    (missing.length ? " — not in Bhavcopy: " + missing.join(", ") : "");
  renderPositions();
}

/* ---- what can be chosen for a symbol: type, expiry, CE/PE, strike ---- */
function manualChoices(s, type, exp, optType, strike){
  const rows = bhav.filter(r => sym(r) === s);
  const types = [];
  if (rows.some(r => !isOpt(r))) types.push("F");
  if (rows.some(isOpt)) types.push("O");
  if (!types.includes(type)) type = types[0] || "";

  const pool = rows.filter(r => type === "O" ? isOpt(r) : !isOpt(r));
  const exps = [...new Set(pool.map(r => normDate(r.XpryDt)))].sort();
  if (!exps.includes(exp)){
    const up = exps.filter(e => e >= todayStr());
    exp = (up.length ? up : exps)[0] || "";
  }

  let opts = [], strikes = [];
  if (type === "O"){
    const ep = pool.filter(r => normDate(r.XpryDt) === exp);
    opts = [...new Set(ep.map(r => String(r.OptnTp || "").trim()))].filter(Boolean).sort();
    if (!opts.includes(optType)) optType = opts.includes("CE") ? "CE" : (opts[0] || "");
    strikes = [...new Set(ep.filter(r => String(r.OptnTp || "").trim() === optType)
                            .map(r => csvNumber(r.StrkPric)))].sort((x, y) => x - y);
    if (!strikes.includes(strike)) strike = strikes[Math.floor(strikes.length / 2)];
  }
  return {types, type, exps, exp, opts, optType, strikes, strike};
}

function refreshManualOptions(){
  const s = $("mSym").value.trim().toUpperCase();
  const ch = manualChoices(s, $("mType").value, $("mExp").value, $("mOpt").value, parseFloat($("mStrike").value));

  const fill = (id, vals, label, sel) => {
    $(id).innerHTML = vals.map(v =>
      '<option value="' + esc(v) + '"' + (String(v) === String(sel) ? " selected" : "") + '>' + esc(label(v)) + '</option>'
    ).join("");
  };
  fill("mType",   ch.types,   v => v === "F" ? "Future" : "Option", ch.type);
  fill("mExp",    ch.exps,    prettyDate, ch.exp);
  fill("mOpt",    ch.opts,    v => v, ch.optType);
  fill("mStrike", ch.strikes, v => v, ch.strike);

  const isO = ch.type === "O";
  $("mOptWrap").style.display = isO ? "" : "none";
  $("mStrikeWrap").style.display = isO ? "" : "none";
}

/* the contract picked in the Manual form */
function manualContract(s){
  const ch = manualChoices(s, $("mType").value, $("mExp").value, $("mOpt").value, parseFloat($("mStrike").value));
  if (!ch.type) return null;
  return bhav.find(r =>
    sym(r) === s &&
    (ch.type === "O" ? isOpt(r) : !isOpt(r)) &&
    normDate(r.XpryDt) === ch.exp &&
    (ch.type !== "O" || (String(r.OptnTp || "").trim() === ch.optType && csvNumber(r.StrkPric) === ch.strike))
  ) || null;
}

function addManual(){
  if (!bhav.length){ alert("Please upload the Bhavcopy first."); return; }
  const s = $("mSym").value.trim().toUpperCase();
  const lots = parseInt($("mLots").value, 10);
  const side = $("mSide").value;
  if (!s){ alert("Enter a symbol."); return; }
  if (!(lots > 0)){ alert("Enter lots (1 or more)."); return; }

  const r = manualContract(s);
  if (!r){ alert(s + " was not found in the Bhavcopy."); return; }

  addToList(r, lots, side);
  renderPositions();   /* symbol and choices stay, so you can add another strike quickly */
}

/* =========================================================
   POSITIONS LIST
========================================================= */
function addToList(r, lots, side){
  const key = contractKey(r);
  const ex = positions.find(p => p.key === key && p.side === side);
  if (ex) ex.lots += lots; else positions.push({key, r, lots, side});
}

function removePosition(i){ positions.splice(i, 1); renderPositions(); }
function clearPositions(){ positions = []; renderPositions(); }
function setLots(i, v){
  const n = parseInt(v, 10);
  if (n > 0) positions[i].lots = n;
  renderPositions();
}
function toggleSide(i){ positions[i].side = positions[i].side === "B" ? "S" : "B"; renderPositions(); }

/* Clear button next to Generate */
function clearAll(){
  positions = [];
  $("bulkRecords").value = "";
  $("mSym").value = "";
  $("mLots").value = "1";
  refreshManualOptions();
  if (mode === "sample") $("sampleStatus").textContent = "";
  renderPositions();
}

/* short one-line label: NIFTY 25Nov26 23000CE / SBIN 25Nov26 FUT */
function shortLabel(r){
  const d = normDate(r.XpryDt);
  let exp = d;
  if (d.length === 8){
    const mon = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"][parseInt(d.slice(4,6),10)-1];
    exp = d.slice(6,8) + mon + d.slice(2,4);
  }
  return sym(r) + " " + exp + (isOpt(r)
    ? " " + csvNumber(r.StrkPric) + String(r.OptnTp || "").trim()
    : " FUT");
}

function renderPositions(){
  $("posCount").textContent = positions.length ? positions.length + " position(s)" : "";
  const box = $("posList");
  if (!positions.length){
    box.innerHTML = '<div class="empty">No positions added yet.</div>';
    return;
  }
  box.innerHTML = positions.map((p, i) => {
    const lotSize = csvNumber(p.r.NewBrdLotQty) || 1;
    const qty = p.lots * lotSize;
    return '<div class="pr">' +
      '<span class="pill ' + p.side + '" onclick="toggleSide(' + i + ')" title="Tap to switch Buy / Sell">' + (p.side === "B" ? "BUY" : "SELL") + '</span>' +
      '<span class="txt" title="' + esc(contractLabel(p.r)) + '">' + esc(shortLabel(p.r)) + ' · ' + fmt(qty) + '</span>' +
      '<input type="number" min="1" step="1" value="' + p.lots + '" onchange="setLots(' + i + ', this.value)" title="Lots">' +
      '<button type="button" class="red x" onclick="removePosition(' + i + ')">✕</button>' +
    '</div>';
  }).join("");
}

/* clients that receive rows for one position */
function clientPlan(side){
  if (clients.length) return clients.map(c => ({id: c, side}));
  const c1 = $("client1").value.trim(), c2 = $("client2").value.trim();
  const plan = [];
  if (c1) plan.push({id: c1, side});
  if (c2) plan.push({id: c2, side: side === "B" ? "S" : "B"});
  return plan;
}

/* =========================================================
   BUILD ROWS
========================================================= */
function makeRow(r, clientId, isBuy, lots, date){
  const lotSize = csvNumber(r.NewBrdLotQty) || 1;
  const qty = lotSize * lots;
  const price = priceOf(r);
  const val = qty * price;
  const opt = isOpt(r);

  const o = {};
  FO_HEADER.forEach(h => o[h] = "");

  o.Sgmt = "FO"; o.Src = "NCL";
  o.RptgDt = date; o.BizDt = date;
  o.TradRegnOrgn = "1";
  o.ClrMmbId = MEMBER; o.BrkrOrCtdnPtcptId = MEMBER;
  o.ClntTp = "C"; o.ClntId = clientId;

  o.FinInstrmTp = tp(r);
  o.ISIN = r.ISIN || "";
  o.TckrSymb = sym(r);
  o.XpryDt = normDate(r.XpryDt);
  o.FininstrmActlXpryDt = normDate(r.FininstrmActlXpryDt || r.XpryDt);
  o.StrkPric = opt ? String(csvNumber(r.StrkPric)) : "";
  o.OptnTp = opt ? String(r.OptnTp || "").trim() : "";
  o.NewBrdLotQty = String(lotSize);

  ["OpngLngQty","OpngLngVal","OpngShrtQty","OpngShrtVal",
   "OpnBuyTradgQty","OpnBuyTradgVal","OpnSellTradgQty","OpnSellTradgVal",
   "PreExrcAssgndLngQty","PreExrcAssgndLngVal","PreExrcAssgndShrtQty","PreExrcAssgndShrtVal",
   "ExrcdQty","AssgndQty",
   "PstExrcAssgndLngQty","PstExrcAssgndLngVal","PstExrcAssgndShrtQty","PstExrcAssgndShrtVal",
   "PrmAmt","DalyMrkToMktSettlmVal","FutrsFnlSttlmVal","ExrcAssgndVal"
  ].forEach(n => o[n] = "0");

  if (isBuy){
    o.OpnBuyTradgQty = String(qty); o.OpnBuyTradgVal = String(val);
    o.PstExrcAssgndLngQty = String(qty); o.PstExrcAssgndLngVal = String(val);
  } else {
    o.OpnSellTradgQty = String(qty); o.OpnSellTradgVal = String(val);
    o.PstExrcAssgndShrtQty = String(qty); o.PstExrcAssgndShrtVal = String(val);
  }
  o.SttlmPric = String(price);

  return FO_HEADER.map(h => o[h]).join(",");
}

function buildRows(){
  const date = todayStr();
  const out = [FO_HEADER.join(",")];
  positions.forEach(p => {
    clientPlan(p.side).forEach(c => out.push(makeRow(p.r, c.id, c.side === "B", p.lots, date)));
  });
  return out;
}

function buildBulk(){
  const n = parseInt($("bulkRecords").value, 10);
  if (!(n > 0)){ alert("Please enter the number of Records required."); return null; }

  const pool = clients.length
    ? clients
    : [$("client1").value.trim(), $("client2").value.trim()].filter(Boolean);

  const groups = ["IDF", "STF", "IDO", "STO"]
    .map(t => bhav.filter(r => tp(r) === t))
    .filter(g => g.length);
  if (!groups.length){ alert("No IDF/STF/IDO/STO contracts found in the Bhavcopy."); return null; }

  const date = todayStr();
  const out = [FO_HEADER.join(",")];
  for (let i = 0; i < n; i++){
    const r = randomItem(groups[i % groups.length]);
    const lots = 1 + Math.floor(Math.random() * 4);          // 1 to 4 lots
    const isBuy = (Math.floor(i / groups.length) % 2) === 0;  // alternate Buy / Sell per instrument type
    out.push(makeRow(r, randomItem(pool), isBuy, lots, date));
  }
  return out;
}

/* =========================================================
   GENERATE
========================================================= */
function generateFiles(){
  if (!bhav.length){ alert("Please upload the Bhavcopy first."); return; }
  if (!clients.length && !$("client1").value.trim()){
    alert("Enter Client 1, or upload a client list.");
    return;
  }

  let lines;
  if (mode === "bulk"){
    lines = buildBulk();
    if (!lines) return;
  } else {
    if (!positions.length){ alert("Add at least one position."); return; }
    lines = buildRows();
  }

  const selected = FILES.filter((f, i) => $("fc_" + i).checked);
  if (!selected.length){ alert("Select at least one file."); return; }

  const text = lines.join("\n") + "\n";
  selected.forEach((f, i) => {
    setTimeout(() => {
      const blob = new Blob([text], {type: "text/csv;charset=utf-8;"});
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = f[0] + ".csv";
      a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 1000);
    }, i * 300);
  });
}

function buildFileBoxes(){
  $("fileBoxes").innerHTML =
    '<label><input type="checkbox" id="fc_all" onclick="toggleAllFiles()"> Select All</label>' +
    FILES.map((f, i) => '<label><input type="checkbox" id="fc_' + i + '"> ' + f[1] + '</label>').join("");
}
function toggleAllFiles(){
  const on = $("fc_all").checked;
  FILES.forEach((f, i) => $("fc_" + i).checked = on);
}

/* =========================================================
   INIT
========================================================= */
window.onload = function(){
  buildFileBoxes();
  renderPositions();
  autoUpload();
};

window.addEventListener("error", e => {
  const el = document.getElementById("autoStatus");
  if (el) el.textContent = "Script error: " + e.message;
});
