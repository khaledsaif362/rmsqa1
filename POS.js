/* =========================================================
   CONSTANTS
========================================================= */
const FO_HEADER = ("Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4").split(",");

/* per-segment settings (member ids etc. are copied from your old templates) */
const SEGMENTS = {
  FO: {
    files: [
      ["Position_ICCL_FO_0_CM_6538_2026", "ICCL CM"],
      ["Position_ICCL_FO_0_TM_6538_2026", "ICCL TM"],
      ["Position_NCL_FO_0_CM_6538_2026",  "NCL CM"],
      ["Position_NCL_FO_0_TM_6538_2026",  "NCL TM", true]   /* ticked by default */
    ],
    base: {Sgmt: "FO", Src: "NCL", Mem: "6538", Brkr: "6538"},
    insts: [["IDF", "Index Future"], ["STF", "Stock Future"], ["IDO", "Index Option"], ["STO", "Stock Option"]]
  },
  MCX: {
    files: [["Position_MCXCCL_CO_0_TM_6538_2026", "MCX"]],
    base: {Sgmt: "CO", Src: "MCXCCL", Mem: "56645", Brkr: "56645"},
    insts: [["COF", "Commodity Future"], ["FUO", "Commodity Option"]]
  },
  CD: {
    files: [["Position_NCL_CD_0_TM_6538_2026", "CD"]],
    base: {Sgmt: "CD", Src: "NCL", Mem: "M52040", Brkr: "90144", RefRate: "100.13"},
    insts: [["CDF", "Currency Future"], ["CDO", "Currency Option"]]
  },
  NCDEX: {
    files: [["Position_NCCL_CO_0_TM_6538_2026", "NCDEX"]],
    base: {Sgmt: "CO", Src: "NCCL", Mem: "M51085", Brkr: "00094", FutStrike: "0"},
    insts: [["COF", "Commodity Future"], ["FUO", "Commodity Option"]]
  },
  NSECOM: {
    files: [["Position_NCL_CO_0_TM_6538_2026", "NSE COM"]],
    base: {Sgmt: "CO", Src: "NCL", Mem: "56645", Brkr: "56645"},
    insts: [["COF", "Commodity Future"], ["FUO", "Commodity Option"]]
  }
};
const SEG_ORDER = ["FO", "MCX", "CD", "NCDEX", "NSECOM"];

/* ---------------------------------------------------------
   SERVER BHAVCOPY (rms_api.go, listens on :8090)
   The API runs on the same host as this page, port 8090.
   rms_api.go only allows CORS from http://172.16.101.74:8080
   and does not send Allow-Credentials, so no cookies are sent.
--------------------------------------------------------- */
const SERVER = {
  base: location.protocol + "//" + location.hostname + ":8090",
  routes: {
    FO: [
      {label: "NSE FO", path: "/api/nse-fo-bhavcopy", info: "/api/nse-fo-bhavcopy-info"},
      {label: "BSE FO", path: "/api/bse-fo-bhavcopy", info: "/api/bse-fo-bhavcopy-info"}
    ]
    ,
    /* these routes are in the edited rms_api.go that comes with this version */
    MCX:    [{label: "MCX",     path: "/api/mcx-bhavcopy",    info: "/api/mcx-bhavcopy-info"}],
    CD:     [{label: "CD",      path: "/api/cd-bhavcopy",     info: "/api/cd-bhavcopy-info"}],
    NCDEX:  [{label: "NCDEX",   path: "/api/ncdex-bhavcopy",  info: "/api/ncdex-bhavcopy-info"}],
    NSECOM: [{label: "NSE COM", path: "/api/nsecom-bhavcopy", info: "/api/nsecom-bhavcopy-info"}]
  }
};

/* FO sample: [symbol, instrument, option type, lots] */
const SAMPLE = [
  ["SBIN",      "STF", "",   1],
  ["BANKNIFTY", "IDF", "",   1],
  ["NIFTY",     "IDO", "CE", 1],
  ["TCS",       "STO", "CE", 1],
  ["SENSEX",    "IDF", "",   5],
  ["SENSEX",    "IDO", "CE", 5],
  ["BANKEX",    "IDO", "PE", 5]
];


/* Sample for MCX / CD / NCDEX / NSE COM, picked from the Bhavcopy: [symbol, instrument, CE/PE, lots] */
const SAMPLE_X = {
  MCX: [
    ["CRUDEOIL", "COF", "", 1], ["SILVERM", "COF", "", 1],
    ["COPPER", "FUO", "PE", 1], ["NATURALGAS", "FUO", "CE", 1]
  ],
  CD: [["GBPINR", "CDF", "", 1], ["USDINR", "CDO", "CE", 1]],
  NCDEX: [["COCUDAKL", "COF", "", 1], ["DHANIYA", "COF", "", 2]]
};
SAMPLE_X.NSECOM = SAMPLE_X.MCX;

let segment = "FO";
const bhavStore = {};   // segment -> parsed Bhavcopy rows
const bhavInfo = {};    // segment -> status text
const posStore = {};    // segment -> {sample, manual, bulk} -> saved positions (each window keeps its own list)
let bhav = [];          // Bhavcopy rows of the current segment
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
function lotSizeOf(r){ return csvNumber(r.NewBrdLotQty) || 1; }
function qtyOf(r, lots){ return lots * (r._qtyPerLot !== undefined ? r._qtyPerLot : lotSizeOf(r)); }
function contractKey(r){
  return [sym(r), tp(r), normDate(r.XpryDt), csvNumber(r.StrkPric), String(r.OptnTp||"").trim()].join("|");
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

/* never let a stuck IndexedDB call freeze the page */
function withTimeout(promise, ms, fallback){
  return Promise.race([promise, new Promise(res => setTimeout(() => res(fallback), ms))]);
}

/* =========================================================
   CACHE (same IndexedDB the Position Files page uses)
========================================================= */
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
const cacheKey = s => s === "FO" ? "bhavcopy" : "bhavcopy_" + s;

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

/* shows the current segment's Bhavcopy status and symbol list */
function refreshBhavUI(){
  $("bhavStatus").textContent = "";   /* file-name line removed; only the one status line is shown */
  const syms = [...new Set(bhav.map(sym))].sort();
  $("underList").innerHTML = syms.map(n => '<option value="' + esc(n) + '">').join("");
  refreshManualOptions();
}

/* files = [{name, text}, ...] — several files (NSE + BSE) are merged into one list */
function applyBhavFiles(files, cached, seg){
  seg = seg || segment;
  const all = [];
  const names = [];
  files.forEach(f => {
    const rows = parseBhavText(f.text);
    if (rows.length){ rows.forEach(r => all.push(r)); names.push(f.name); }
  });

  if (!all.length){
    bhavStore[seg] = [];
    bhavInfo[seg] = "Invalid or empty Bhavcopy";
    $("autoStatus").textContent = seg + ": invalid or empty Bhavcopy";
  } else {
    bhavStore[seg] = all;
    bhavInfo[seg] = (seg === "FO" ? "" : seg + ": ") + names.join(", ") + " · " +
      all.length.toLocaleString() + " contracts" + (cached ? " (saved)" : "");
  }
  if (seg === segment){ bhav = bhavStore[seg]; refreshBhavUI(); }
  return all.length > 0;
}

/* which segment the Bhavcopy button loads into */
function bhavTarget(){
  return segment;
}

const manualKey = seg => cacheKey(seg) + "_manual";

function onBhavFile(input){
  const list = input.files ? [...input.files] : [];
  if (!list.length) return;
  const target = bhavTarget();
  Promise.all(list.map(f => new Promise((resolve, reject) => {
    const rd = new FileReader();
    rd.onload = e => resolve({
      name: f.name, text: String(e.target.result || ""),
      meta: {name: f.name, size: f.size, modified: f.lastModified}
    });
    rd.onerror = () => reject(new Error("read"));
    rd.readAsText(f);
  }))).then(async files => {
    /* same file(s) as the last manual upload (name + size + date)? then nothing new to save */
    const old = await withTimeout(cacheGet(manualKey(target)), 3000, null);
    const same = old && old.files && old.files.length === files.length &&
      files.every(f => old.files.some(o => sameMeta(o.meta, f.meta)));

    if (!applyBhavFiles(files, !!same, target)) return;
    if (!same){
      cacheSet(manualKey(target), {name: files.map(f => f.name).join(", "), at: Date.now(), files});
    } else {
      cacheSet(manualKey(target), Object.assign({}, old, {at: Date.now()}));   /* it is the latest one again */
    }
    $("autoStatus").textContent = target + ": " + (same ? "your upload (saved)" : "uploaded") + " · " + (bhavStore[target] || []).length.toLocaleString() + " contracts";
  }).catch(() => alert("Could not read the Bhavcopy.")).finally(() => { input.value = ""; });
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
  input.value = "";
}

function clearClients(){
  cacheDelete("clients");
  $("clientFile").value = "";
  setClients([], "", false);
}

/* =========================================================
   SERVER FETCH
========================================================= */
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

/* small request: {name, modified, size} of the newest file on the server (null if unavailable) */
async function fetchBhavInfo(item){
  if (!item.info) return null;
  const ctl = new AbortController();
  const timer = setTimeout(() => ctl.abort(), 8000);
  try{
    const res = await fetch(SERVER.base + item.info, {cache: "no-store", signal: ctl.signal});
    if (!res.ok) return null;
    const j = await res.json();
    return j && j.name ? {name: j.name, modified: j.modified, size: j.size} : null;
  }catch(e){
    return null;
  }finally{
    clearTimeout(timer);
  }
}

const sameMeta = (a, b) => !!(a && b && a.name === b.name && a.modified === b.modified && a.size === b.size);

/* fetch all routes of one segment. A file whose name / time / size is unchanged since the
   saved copy is NOT downloaded again - the saved copy is reused. */
async function fetchSegmentBhav(items, cachedFiles){
  const results = await Promise.allSettled(items.map(async item => {
    const meta = await fetchBhavInfo(item);
    const old = (cachedFiles || []).find(f => f.label === item.label);
    if (old && sameMeta(meta, old.meta)) return {file: old, reused: true};
    const f = await fetchBhavFromServer(item);
    if (meta) f.name = meta.name;
    f.label = item.label;
    f.meta = meta;
    return {file: f, reused: false};
  }));
  const ok = results.filter(r => r.status === "fulfilled").map(r => r.value);
  return {
    files: ok.map(v => v.file),
    reused: ok.filter(v => v.reused).length,
    errors: results.filter(r => r.status === "rejected").map(r => r.reason.message)
  };
}

/* Auto Upload: only changed server files are downloaded. Two saved copies are kept per segment -
   the one from the server and the one you uploaded by hand - and the NEWEST of them is used. */
async function autoUpload(){
  $("autoStatus").textContent = "Checking Bhavcopy on server…";

  /* client list from the saved copy (never blocks the fetch) */
  withTimeout(cacheGet("clients"), 3000, null).then(c => {
    if (c && c.list && c.list.length) setClients(c.list, c.name || "saved list", true);
  });

  const filesOf = e => e ? (e.files || (e.text ? [{name: e.name || "Bhavcopy", text: e.text}] : null)) : null;

  const out = [];
  await Promise.all(Object.keys(SERVER.routes).map(async s => {
    const items = SERVER.routes[s];
    const auto = await withTimeout(cacheGet(cacheKey(s)), 3000, null);
    const man  = await withTimeout(cacheGet(manualKey(s)), 3000, null);
    const {files, reused, errors} = await fetchSegmentBhav(items, auto && auto.files);

    let autoAt = (auto && auto.at) || 0;
    if (files.length && reused < files.length){          /* something new came from the server */
      autoAt = Date.now();
      cacheSet(cacheKey(s), {name: files.map(f => f.name).join(", "), text: files[0].text, at: autoAt, files});
    }
    const manAt = (man && man.at) || 0;

    if (files.length && !(manAt > autoAt && filesOf(man))){
      applyBhavFiles(files, reused === files.length, s);
      out.push(s + ": " + (reused === files.length ? "up to date" : "downloaded " + (files.length - reused) + "/" + items.length));
    } else if (filesOf(man) && (manAt >= autoAt || !filesOf(auto))){
      applyBhavFiles(filesOf(man), true, s);
      out.push(s + ": your upload");
    } else if (filesOf(auto)){
      applyBhavFiles(filesOf(auto), true, s);
      out.push(s + ": saved");
    } else {
      out.push(s + ": none" + (s === "FO" && errors[0] ? " (" + errors[0] + ")" : ""));
    }
  }));
  out.sort();
  $("autoStatus").textContent = out.join(" · ");
}

/* =========================================================
   PICK A CONTRACT FROM THE BHAVCOPY
   Nearest upcoming expiry. Futures preferred when no
   instrument is given; options use the middle strike.
========================================================= */
function pickContract(symbol, inst, optType){
  let rows = bhav.filter(r => sym(r) === symbol);
  if (inst){
    const exact = rows.filter(r => tp(r) === inst);
    /* other segments may use different instrument codes - fall back to future / option */
    rows = exact.length ? exact : rows.filter(r => isOpt(r) === inst.endsWith("O"));
  }
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
   SEGMENTS
========================================================= */
function savePos(){ (posStore[segment] = posStore[segment] || {})[mode] = positions; }
function loadPos(){ positions = (posStore[segment] || {})[mode] || []; }

function setSegment(s){
  savePos();
  segment = s;
  bhav = bhavStore[s] || [];

  SEG_ORDER.forEach(k => $("tab_" + k).classList.toggle("active", k === s));
  buildFileBoxes();
  refreshBhavUI();
  loadPos();
  showMode(mode);
  if (mode === "sample" && !positions.length) loadSample();
  renderPositions();
}

/* =========================================================
   MODES
========================================================= */
function showMode(m){
  mode = m;
  ["sample", "manual", "bulk"].forEach(k => $("mb_" + k).classList.toggle("active", k === m));
  let free = false;
  $("sampleBox").style.display = m === "sample" ? "block" : "none";
  $("manualBox").style.display = m === "manual" && !free ? "block" : "none";
  $("bulkBox").style.display   = m === "bulk"   ? "block" : "none";
  $("posWrap").style.display   = m === "bulk"   ? "none"  : "block";
}

function setMode(m){
  /* Sample, Manual and Bulk each keep their own positions; switching only swaps the list */
  savePos();
  showMode(m);
  loadPos();
  if (m === "sample" && !positions.length) loadSample();
  renderPositions();
}

function loadSample(){
  if (!bhav.length){
    $("sampleStatus").textContent = "Upload the Bhavcopy first.";
    alert("Please upload the Bhavcopy first.");
    return;
  }
  let list = SAMPLE;
  list = segment === "FO" ? SAMPLE : (SAMPLE_X[segment] || []);
  positions = [];
  const missing = [];
  list.forEach(([s, inst, opt, lots]) => {
    const r = pickContract(s, inst, opt);
    if (!r){ missing.push(s); return; }
    addToList(r, lots, "B");
  });
  $("sampleStatus").textContent =
    positions.length + " sample position(s) loaded" +
    (missing.length ? " — not in Bhavcopy: " + missing.join(", ") : "");
  renderPositions();
}


/* ---- Manual entry from the Bhavcopy: type, expiry, CE/PE, strike ---- */
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
  let exp = "no expiry";
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
    const qty = qtyOf(p.r, p.lots);
    return '<div class="pr">' +
      '<span class="pill ' + p.side + '" onclick="toggleSide(' + i + ')" title="Tap to switch Buy / Sell">' + (p.side === "B" ? "BUY" : "SELL") + '</span>' +
      '<span class="txt">' + esc(shortLabel(p.r)) + ' · ' + fmt(qty) + '</span>' +
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
  const b = SEGMENTS[segment].base;
  const qty = qtyOf(r, lots);
  const price = priceOf(r);
  const val = qty * (r._unitVal !== undefined ? r._unitVal : price);
  const opt = isOpt(r);

  const o = {};
  FO_HEADER.forEach(h => o[h] = "");

  o.Sgmt = b.Sgmt; o.Src = b.Src;
  o.RptgDt = date; o.BizDt = date;
  o.TradRegnOrgn = "1";
  o.ClrMmbId = b.Mem; o.BrkrOrCtdnPtcptId = b.Brkr;
  o.ClntTp = "C"; o.ClntId = clientId;

  o.FinInstrmTp = tp(r);
  o.ISIN = r.ISIN || "";
  o.TckrSymb = sym(r);
  o.XpryDt = normDate(r.XpryDt);
  o.FininstrmActlXpryDt = normDate(r.FininstrmActlXpryDt || r.XpryDt);
  o.StrkPric = opt ? String(csvNumber(r.StrkPric)) : (b.FutStrike || "");
  o.OptnTp = opt ? String(r.OptnTp || "").trim() : "";
  o.NewBrdLotQty = String(lotSizeOf(r));

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
  if (b.RefRate) o.RefRate = b.RefRate;

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

function bulkGroups(){
  if (segment === "FO"){
    return ["IDF", "STF", "IDO", "STO"].map(t => bhav.filter(r => tp(r) === t)).filter(g => g.length);
  }
  return [bhav.filter(r => !isOpt(r)), bhav.filter(isOpt)].filter(g => g.length);
}

function buildBulk(){
  const n = parseInt($("bulkRecords").value, 10);
  if (!(n > 0)){ alert("Please enter the number of Records required."); return null; }

  const pool = clients.length
    ? clients
    : [$("client1").value.trim(), $("client2").value.trim()].filter(Boolean);

  const groups = bulkGroups();
  if (!groups.length){ alert("No contracts found in the Bhavcopy."); return null; }

  const date = todayStr();
  const out = [FO_HEADER.join(",")];
  for (let i = 0; i < n; i++){
    const r = randomItem(groups[i % groups.length]);
    const lots = 1 + Math.floor(Math.random() * 4);          // 1 to 4 lots
    const isBuy = (Math.floor(i / groups.length) % 2) === 0;  // alternate Buy / Sell per group
    out.push(makeRow(r, randomItem(pool), isBuy, lots, date));
  }
  return out;
}

/* =========================================================
   GENERATE
========================================================= */
function generateFiles(){
  if (!clients.length && !$("client1").value.trim()){
    alert("Enter Client 1, or upload a client list.");
    return;
  }

  let lines;
  if (mode === "bulk"){
    if (!bhav.length){ alert("Please upload the Bhavcopy first."); return; }
    lines = buildBulk();
    if (!lines) return;
  } else {
    if (!positions.length){ alert("Add at least one position."); return; }
    const noExp = positions.find(p => !normDate(p.r.XpryDt));
    if (noExp){ alert("Select the expiry for " + sym(noExp.r) + "."); return; }
    lines = buildRows();
  }

  const files = SEGMENTS[segment].files;
  const selected = files.filter((f, i) => $("fc_" + i).checked);
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
  const files = SEGMENTS[segment].files;
  const single = files.length === 1;
  $("fileBoxes").innerHTML =
    (single ? "" : '<label><input type="checkbox" id="fc_all" onclick="toggleAllFiles()"> Select All</label>') +
    files.map((f, i) => '<label><input type="checkbox" id="fc_' + i + '"' + (single || f[2] ? " checked" : "") + '> ' + f[1] + '</label>').join("");
}
function toggleAllFiles(){
  const on = $("fc_all").checked;
  SEGMENTS[segment].files.forEach((f, i) => $("fc_" + i).checked = on);
}

/* =========================================================
   INIT
========================================================= */
window.onload = function(){
  buildFileBoxes();
  showMode(mode);
  renderPositions();
  autoUpload();
};

window.addEventListener("error", e => {
  const el = document.getElementById("autoStatus");
  if (el) el.textContent = "Script error: " + e.message;
});
