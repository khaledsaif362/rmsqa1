/* =========================================================
   POSITION FILE NAMES
========================================================= */

const fileNames = {

    FO: [
        "Position_ICCL_FO_0_CM_6538_2026",
        "Position_ICCL_FO_0_TM_6538_2026",
        "Position_NCL_FO_0_CM_6538_2026",
        "Position_NCL_FO_0_TM_6538_2026"
    ],

    MCX: [
        "Position_MCXCCL_CO_0_TM_6538_2026"
    ],

    CD: [
        "Position_NCL_CD_0_TM_6538_2026"
    ],

    NCDEX: [
        "Position_NCCL_CO_0_TM_6538_2026"
    ],

    NSECOM: [
        "Position_NCL_CO_0_TM_6538_2026"
    ]

};


/* =========================================================
   SHORT CHECKBOX LABEL

   The FO filenames are long
   ("Position_ICCL_FO_0_CM_6538_2026"), which pushes the
   checkbox list wide. This pulls out just the exchange
   (ICCL / NCL) and segment (CM / TM) for display, while
   the checkbox's value/id still carries the full filename
   used when generating the download.
========================================================= */

function shortFileLabel(name) {

    const match =
        name.match(/^Position_([A-Z]+)_FO_0_(CM|TM)_/);


    if (match) {

        return match[1] + " " + match[2];

    }


    return name;

}


/* =========================================================
   STRIKE STEPS
========================================================= */

const strikeSteps = {

    /* =========================
       NSE / BSE
    ========================== */

    NIFTY: 50,

    SENSEX: 100,

    BANKEX: 100,

    TCS: 20,


    /* =========================
       MCX
    ========================== */

    CRUDEOIL: 100,

    NATURALGAS: 5,

    COPPER: 10,

    SILVERM: 500,

    GOLD: 1000,

    GOLDM: 100,

    SILVER: 500

};


/* =========================================================
   MANUAL SCRIP ENTRY
========================================================= */

/*
 * Instrument type codes available per segment.
 * Codes ending with "O" are treated as options
 * (strike + CE/PE), everything else as futures.
 */

const instrumentTypes = {

    FO: [
        ["STF", "Stock Future"],
        ["IDF", "Index Future"],
        ["STO", "Stock Option"],
        ["IDO", "Index Option"]
    ],

    MCX: [
        ["COF", "Commodity Future"],
        ["FUO", "Commodity Option"]
    ],

    NSECOM: [
        ["COF", "Commodity Future"],
        ["FUO", "Commodity Option"]
    ],

    NCDEX: [
        ["COF", "Commodity Future"],
        ["FUO", "Commodity Option"]
    ],

    CD: [
        ["CDF", "Currency Future"],
        ["CDO", "Currency Option"]
    ]

};


let manualRowSeq = 0;


/*
 * Which top-level mode the form is in — set explicitly
 * by the Sample Positions / Manual Entry toggle at the
 * top of the form, rather than being silently inferred
 * from whether a symbol has been typed.
 */

let currentMode = "sample";


/*
 * Client list loaded from an uploaded .txt/.csv file —
 * one client ID per line. When non-empty, this replaces
 * Client ID 1/2 entirely: every position (sample or
 * manual) is generated once per client here, as a
 * buy-side row.
 */

let uploadedClients = [];


/* =========================================================
   HANDLE CLIENT LIST FILE UPLOAD
========================================================= */

function handleClientFileUpload(input) {

    const file =
        input.files && input.files[0];


    if (!file) {

        return;

    }


    const reader =
        new FileReader();


    reader.onload = function (e) {

        const text =
            String(e.target.result || "");


        /*
         * One client per line. Also tolerate a CSV
         * export by taking just the first comma-separated
         * value on each line, and strip any surrounding
         * quotes Excel sometimes adds.
         */

        uploadedClients =
            text
                .split(/\r?\n/)
                .map(line =>
                    line
                        .split(",")[0]
                        .trim()
                        .replace(/^"|"$/g, "")
                )
                .filter(id => id.length > 0);


        refreshClientFileStatus();

    };


    reader.onerror = function () {

        alert(
            "Could not read that file. Please try again."
        );

    };


    reader.readAsText(file);

}


/* =========================================================
   CLEAR CLIENT LIST FILE
========================================================= */

function clearClientFile() {

    uploadedClients = [];


    const input =
        document.getElementById("clientFile");


    if (input) {

        input.value = "";

    }


    refreshClientFileStatus();

}


/* =========================================================
   REFRESH CLIENT FILE STATUS / GREY OUT CLIENT 1 & 2
========================================================= */

function refreshClientFileStatus() {

    const status =
        document.getElementById("clientFileStatus");


    const clearBtn =
        document.getElementById("clientFileClearBtn");


    const active =
        uploadedClients.length > 0;


    if (status) {

        status.textContent =
            active
                ? uploadedClients.length +
                  " client(s) loaded from file"
                : "";

    }


    if (clearBtn) {

        clearBtn.style.display =
            active ? "inline-block" : "none";

    }


    ["client1", "client2"].forEach(id => {

        const el =
            document.getElementById(id);


        if (!el) {

            return;

        }


        el.disabled = active;


        if (active) {

            el.value = "";

        }

    });

}


/* =========================================================
   BULK GENERATOR
========================================================= */
let bulkBhavcopyRows = [];
let bulkClients = [];

function handleBulkBhavcopy(input) {
    const file = input.files && input.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = e => {
        const lines = String(e.target.result || '').split(/\r?\n/).filter(x => x.trim());
        if (lines.length < 2) {
            bulkBhavcopyRows = [];
            document.getElementById('bulkBhavcopyStatus').textContent = 'Invalid/empty Bhavcopy';
            return;
        }
        const headers = parseCSVLine(lines[0]);
        bulkBhavcopyRows = lines.slice(1).map(line => {
            const c = parseCSVLine(line), o = {};
            headers.forEach((h,i) => o[h] = c[i] ?? '');
            return o;
        }).filter(r => r.TckrSymb && r.XpryDt);
        document.getElementById('bulkBhavcopyStatus').textContent =
            `${file.name} — ${bulkBhavcopyRows.length.toLocaleString()} contracts loaded`;
    };
    reader.onerror = () => alert('Could not read the Bhavcopy.');
    reader.readAsText(file);
}

function handleBulkClientFile(input) {
    const file = input.files && input.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = e => {
        bulkClients = String(e.target.result || '')
            .split(/\r?\n/)
            .map(line => line.split(',')[0].trim().replace(/^"|"$/g,''))
            .filter(Boolean);
        document.getElementById('bulkClientStatus').textContent =
            `${file.name} — ${bulkClients.length.toLocaleString()} clients loaded`;
    };
    reader.onerror = () => alert('Could not read the client list.');
    reader.readAsText(file);
}

function parseCSVLine(line) {
    const out=[]; let cur='', quoted=false;
    for(let i=0;i<line.length;i++){
        const ch=line[i];
        if(ch==='"' && line[i+1]==='"'){cur+='"';i++;continue;}
        if(ch==='"'){quoted=!quoted;continue;}
        if(ch===',' && !quoted){out.push(cur);cur='';continue;}
        cur+=ch;
    }
    out.push(cur);
    return out.map(v=>v.trim());
}

function closeBulkDrawer(){
    document.getElementById('bulkDrawer')?.classList.remove('open');
}

function clearBulkInputs(){
    bulkBhavcopyRows=[]; bulkClients=[];
    const b=document.getElementById('bulkBhavcopy'); if(b)b.value='';
    const c=document.getElementById('bulkClientFile'); if(c)c.value='';
    const r=document.getElementById('bulkRecords'); if(r)r.value='';
    document.getElementById('bulkBhavcopyStatus').textContent='No Bhavcopy selected';
    document.getElementById('bulkClientStatus').textContent='No client list selected';
}

function setBulkUiState(active){
    const card = document.querySelector('.card');
    if (!card) return;

    // Disable normal position-entry controls while Bulk mode is active.
    card.querySelectorAll('input, select, button').forEach(el => {
        if (el.closest('#modeToggle') || el.closest('#bulkDrawer')) return;
        el.disabled = active;
    });

    const box=document.getElementById('checkboxContainer');
    if(box) box.classList.toggle('bulk-disabled',active);

    const manual=document.getElementById('manualReopen');
    if(manual && active) manual.style.display='none';

    ['commonExpiryRow','sensexBankexExpiryRow','additionalStrikeRow','clientFileRow']
        .forEach(id => document.getElementById(id)?.classList.toggle('bulk-disabled',active));
}

function randomItem(arr){
    return arr[Math.floor(Math.random()*arr.length)];
}

function csvNumber(v){
    const n=parseFloat(String(v ?? '').replace(/,/g,''));
    return Number.isFinite(n) ? n : 0;
}

function generateBulk(){
    if(!bulkBhavcopyRows.length){ alert('Please upload a Bhavcopy.'); return; }
    if(!bulkClients.length){ alert('Please upload a client list.'); return; }

    const records=parseInt(document.getElementById('bulkRecords')?.value||'0',10);
    if(!records || records<1){ alert('Please enter the number of Records required.'); return; }

    const type=document.getElementById('positionType').value;
    const raw=rawCSVs[type];
    if(!raw){ alert(type+' CSV template not found.'); return; }
    if(type!=='FO'){ alert('Bulk Bhavcopy generation is currently supported for FO.'); return; }

    /*
       Bhavcopy instrument groups:
       IDF = Index Future
       STF = Stock Future
       IDO = Index Option
       STO = Stock Option
    */
    const groups={
        indexFuture: bulkBhavcopyRows.filter(r=>String(r.FinInstrmTp).trim()==='IDF'),
        stockFuture: bulkBhavcopyRows.filter(r=>String(r.FinInstrmTp).trim()==='STF'),
        indexOption: bulkBhavcopyRows.filter(r=>String(r.FinInstrmTp).trim()==='IDO'),
        stockOption: bulkBhavcopyRows.filter(r=>String(r.FinInstrmTp).trim()==='STO')
    };

    const available=Object.entries(groups).filter(([,rows])=>rows.length);
    if(!available.length){
        alert('No IDF/STF/IDO/STO contracts found in the uploaded Bhavcopy.');
        return;
    }

    const lines=raw.trim().split(/\r?\n/);
    const header=lines[0];
    const template=parseCSVLine(header);
    const idx=n=>template.indexOf(n);
    const required=['ClntId','FinInstrmTp','TckrSymb','XpryDt','FininstrmActlXpryDt','NewBrdLotQty','OpnBuyTradgQty','OpnBuyTradgVal','OpnSellTradgQty','OpnSellTradgVal','SttlmPric','StrkPric','OptnTp'];
    const missing=required.filter(n=>idx(n)<0);
    if(missing.length){ alert('FO template is missing: '+missing.join(', ')); return; }

    const now=new Date();
    const date=now.getFullYear()+String(now.getMonth()+1).padStart(2,'0')+String(now.getDate()).padStart(2,'0');
    const output=[header];

    for(let i=0;i<records;i++){
        // Randomly select one of the four requested contract categories.
        const category=randomItem(available);
        const b=randomItem(category[1]);
        const c=randomItem(bulkClients);
        const row=Array(template.length).fill('');

        row[idx('Sgmt')]='FO';
        row[idx('Src')]='NCL';
        row[idx('RptgDt')]=date;
        row[idx('BizDt')]=date;
        row[idx('TradRegnOrgn')]='1';
        row[idx('ClrMmbId')]='6538';
        row[idx('BrkrOrCtdnPtcptId')]='6538';
        row[idx('ClntTp')]='C';
        row[idx('ClntId')]=c;

        row[idx('FinInstrmTp')]=b.FinInstrmTp||'';
        row[idx('ISIN')]=b.ISIN||'';
        row[idx('TckrSymb')]=b.TckrSymb||'';
        row[idx('XpryDt')]=String(b.XpryDt||'').replaceAll('-','');
        row[idx('FininstrmActlXpryDt')]=String(b.FininstrmActlXpryDt||b.XpryDt||'').replaceAll('-','');
        row[idx('StrkPric')]=b.StrkPric||'';
        row[idx('OptnTp')]=b.OptnTp||'';

        const lotSize=csvNumber(b.NewBrdLotQty)||1;
        const lots=1+Math.floor(Math.random()*4); // maximum 4 lots
        const qty=lotSize*lots;
        const price=csvNumber(b.SttlmPric||b.ClsPric||b.LastPric||b.LwPric||b.OpnPric);
        const value=qty*price;

        row[idx('NewBrdLotQty')]=String(lotSize);
        row[idx('OpngLngQty')]='0';
        row[idx('OpngLngVal')]='0';
        row[idx('OpngShrtQty')]='0';
        row[idx('OpngShrtVal')]='0';
        row[idx('OpnBuyTradgQty')]='0';
        row[idx('OpnBuyTradgVal')]='0';
        row[idx('OpnSellTradgQty')]='0';
        row[idx('OpnSellTradgVal')]='0';

        // Random Buy / Sell for every generated position.
        const isBuy = Math.random() < 0.5;
        if(isBuy){
            row[idx('OpnBuyTradgQty')]=String(qty);
            row[idx('OpnBuyTradgVal')]=String(value);
        }else{
            row[idx('OpnSellTradgQty')]=String(qty);
            row[idx('OpnSellTradgVal')]=String(value);
        }

        /* Position-file exercise fields: never leave these blank.
           The standard position format uses explicit zeroes for
           pre-exercise / exercise fields. Post-exercise fields carry
           the generated position, matching the existing manual logic. */
        [
            'PreExrcAssgndLngQty',
            'PreExrcAssgndLngVal',
            'PreExrcAssgndShrtQty',
            'PreExrcAssgndShrtVal',
            'ExrcdQty',
            'AssgndQty',
            'PstExrcAssgndLngQty',
            'PstExrcAssgndLngVal',
            'PstExrcAssgndShrtQty',
            'PstExrcAssgndShrtVal'
        ].forEach(n => {
            if (idx(n) >= 0) row[idx(n)] = '0';
        });

        /* Buy = post-exercise long, Sell = post-exercise short.
           This is the same convention already used by Manual Entry. */
        if (isBuy) {
            if(idx('PstExrcAssgndLngQty')>=0) row[idx('PstExrcAssgndLngQty')]=String(qty);
            if(idx('PstExrcAssgndLngVal')>=0) row[idx('PstExrcAssgndLngVal')]=String(value);
        } else {
            if(idx('PstExrcAssgndShrtQty')>=0) row[idx('PstExrcAssgndShrtQty')]=String(qty);
            if(idx('PstExrcAssgndShrtVal')>=0) row[idx('PstExrcAssgndShrtVal')]=String(value);
        }

        if(idx('PrmAmt')>=0) row[idx('PrmAmt')]='0';
        if(idx('DalyMrkToMktSettlmVal')>=0) row[idx('DalyMrkToMktSettlmVal')]='0';
        if(idx('FutrsFnlSttlmVal')>=0) row[idx('FutrsFnlSttlmVal')]='0';
        if(idx('ExrcAssgndVal')>=0) row[idx('ExrcAssgndVal')]='0';
        if(idx('SttlmPric')>=0) row[idx('SttlmPric')]=String(price);

        output.push(row.map(v=>String(v??'')).join(','));
    }

    const blob=new Blob([output.join('\n')+'\n'],{type:'text/csv;charset=utf-8;'});
    const a=document.createElement('a');
    a.href=URL.createObjectURL(blob);
    a.download='Position_NCL_FO_0_CM_4444_2026.csv';
    a.click();
    setTimeout(()=>URL.revokeObjectURL(a.href),1000);
}



/* =========================================================
   SET MODE (Sample Positions / Manual Entry / Bulk Generator)
========================================================= */

function setMode(mode) {

    currentMode = mode;


    document
        .querySelectorAll("#modeToggle input[type=radio]")
        .forEach(radio => {

            radio.checked =
                radio.dataset.mode === mode;

        });


    if (mode === "manual") {
        setBulkUiState(false);
        closeBulkDrawer();
        openManualPanel();
    }
    else if (mode === "bulk") {
        clearManualRows();
        closeManualDrawer();
        setBulkUiState(true);
        document.getElementById("bulkDrawer")?.classList.add("open");
    }
    else {
        setBulkUiState(false);
        clearManualRows();
        closeManualDrawer();
        closeBulkDrawer();
    }


    refreshManualReopenLink();


    setSegmentVisibility(
        document.getElementById("positionType").value
    );

}


/* =========================================================
   SHOW / HIDE THE "(edit)" REOPEN LINK
   Sits next to the Manual Entry radio label. Only
   needed (and only visible) once that mode is active,
   since clicking an already-checked radio fires no
   change event and wouldn't otherwise reopen the popup.
========================================================= */

function refreshManualReopenLink() {

    const link =
        document.getElementById("manualReopen");


    if (!link) {

        return;

    }


    if (currentMode !== "manual") {

        link.style.display = "none";

        return;

    }


    link.style.display = "inline";


    const n =
        getManualEntries().length;


    link.textContent =
        n ? `(edit — ${n} added)` : "(edit)";

}


/* =========================================================
   ADD ONE MANUAL SCRIP ROW
========================================================= */

function addManualRow() {

    const type =
        document.getElementById("positionType").value;


    const container =
        document.getElementById("manualContainer");


    const id =
        ++manualRowSeq;


    const options =
        (instrumentTypes[type] || [])
            .map(
                ([code, label]) =>
                    `<option value="${code}">${code} — ${label}</option>`
            )
            .join("");


    const row =
        document.createElement("div");


    row.className = "manual-row";

    row.id = "manualRow_" + id;


    row.innerHTML = `

        <div class="manual-grid">

            <div class="manual-field">
                <label>Symbol</label>
                <input type="text"
                       class="m-symbol"
                       placeholder="e.g. RELIANCE"
                       oninput="this.value=this.value.toUpperCase(); refreshManualNotice();">
            </div>

            <div class="manual-field">
                <label>Instrument</label>
                <select class="m-inst"
                        onchange="syncManualRow(${id})">
                    ${options}
                </select>
            </div>

            <div class="manual-field">
                <label>Expiry</label>
                <input type="date" class="m-expiry">
            </div>

            <div class="manual-field">
                <label>CE / PE</label>
                <select class="m-opt" disabled>
                    <option value="CE">CE</option>
                    <option value="PE">PE</option>
                </select>
            </div>

            <div class="manual-field">
                <label>Strike</label>
                <input type="number" class="m-strike" step="any" disabled>
            </div>

            <div class="manual-field">
                <label>Qty (Lot Size)</label>
                <input type="number" class="m-lot" min="1" step="any" value="1">
            </div>

            <div class="manual-field">
                <label>Price</label>
                <input type="number" class="m-price" step="any" value="0">
            </div>

        </div>

        <div class="manual-row-foot">
            <button type="button"
                    onclick="addManualRow()">
                + Add Symbol
            </button>
            <button type="button"
                    onclick="removeManualRow(${id})">
                Remove
            </button>
        </div>

    `;


    container.appendChild(row);


    openManualDrawer();


    syncManualRow(id);

    refreshManualNotice();

}


/* =========================================================
   ENABLE / DISABLE OPTION FIELDS
========================================================= */

function syncManualRow(id) {

    const row =
        document.getElementById("manualRow_" + id);


    if (!row) {

        return;

    }


    const inst =
        row.querySelector(".m-inst").value;


    const isOption =
        inst.endsWith("O");


    row.querySelector(".m-opt").disabled = !isOption;

    row.querySelector(".m-strike").disabled = !isOption;

}


/* =========================================================
   REMOVE MANUAL SCRIP ROW
========================================================= */

function removeManualRow(id) {

    const row =
        document.getElementById("manualRow_" + id);


    if (row) {

        row.remove();

    }


    refreshManualNotice();

}


/* =========================================================
   CLEAR ALL MANUAL ROWS
========================================================= */

function clearManualRows() {

    document.getElementById("manualContainer").innerHTML = "";

    refreshManualNotice();

}


/* =========================================================
   OPEN / CLOSE THE MANUAL SCRIP POPUP
========================================================= */

/*
 * Called from the Manual Entry radio (and the "(edit)"
 * reopen link). If there's nothing entered yet, this
 * adds a row straight away so the person doesn't have
 * to select Manual Entry and then + Add Symbol separately.
 */

function openManualPanel() {

    if (document.querySelectorAll(".manual-row").length === 0) {

        addManualRow();

    }

    else {

        openManualDrawer();

    }

}


function openManualDrawer() {

    document
        .getElementById("manualDrawer")
        .classList.add("open");

}


function closeManualDrawer() {

    document
        .getElementById("manualDrawer")
        .classList.remove("open");

}


/* =========================================================
   UPDATE TRIGGER STRIP COUNT
========================================================= */

function refreshManualCount() {

    const el =
        document.getElementById("manualCount");


    if (!el) {

        return;

    }


    const n =
        getManualEntries().length;


    el.textContent =
        n ? n + " added" : "";

}


/* =========================================================
   READ MANUAL ROWS
========================================================= */

function getManualEntries() {

    const rows =
        [...document.querySelectorAll(".manual-row")];


    const entries = [];


    rows.forEach(row => {

        const symbol =
            row.querySelector(".m-symbol")
                .value
                .trim()
                .toUpperCase();


        if (!symbol) {

            return;

        }


        const inst =
            row.querySelector(".m-inst").value;


        entries.push({

            symbol: symbol,

            inst: inst,

            isOption: inst.endsWith("O"),

            expiry: row.querySelector(".m-expiry").value,

            optType: row.querySelector(".m-opt").value,

            strike: row.querySelector(".m-strike").value.trim(),

            /*
             * Strike step is picked up automatically from
             * the known lookup table (strikeSteps). If the
             * symbol isn't in that table, additional strikes
             * are simply not generated for it.
             */

            step: strikeSteps[symbol] || 0,

            lot:
                parseFloat(
                    row.querySelector(".m-lot").value
                ) || 0,

            price:
                parseFloat(
                    row.querySelector(".m-price").value
                ) || 0

        });

    });


    return entries;

}


/* =========================================================
   MANUAL MODE NOTICE
========================================================= */

function refreshManualNotice() {

    refreshManualCount();

    refreshManualReopenLink();


    const notice =
        document.getElementById("manualNotice");


    if (!notice) {

        return;

    }


    const manualOn =
        currentMode === "manual";


    notice.style.display =
        manualOn
            ? "block"
            : "none";


    setSegmentVisibility(
        document.getElementById("positionType").value
    );

}


/* =========================================================
   SHOW / HIDE SEGMENT FIELDS
   Rows are shown/hidden purely based on the segment
   type, same as always. Manual Entry mode doesn't hide
   the Expiry field(s) or Additional Strikes — it greys
   them out instead, since every manual scrip carries its
   own expiry (and strike step) regardless.

   BUGFIX: these rows use the .form-row class, and its
   page-scoped rule forces "display: flex !important" so
   labels/inputs sit on one line. A !important stylesheet
   rule always beats a plain inline style, so the previous
   version here — which set element.style.display directly
   — was silently overridden, and these rows kept showing
   on every segment. Toggling the .hidden class (also
   !important, declared after) fixes this reliably.
========================================================= */

function setSegmentVisibility(type) {

    const manualOn =
        currentMode === "manual";


    const commonExpiry =
        document.getElementById("commonExpiryRow");


    const bseExpiry =
        document.getElementById("sensexBankexExpiryRow");


    const additionalStrike =
        document.getElementById("additionalStrikeRow");


    const symbolExpiry =
        document.getElementById("symbolExpiryContainer");


    commonExpiry.classList.toggle(
        "hidden",
        type !== "FO"
    );


    bseExpiry.classList.toggle(
        "hidden",
        type !== "FO"
    );


    additionalStrike.classList.toggle(
        "hidden",
        !(type === "FO" || type === "MCX")
    );


    symbolExpiry.style.display =
        type === "FO" ? "none" : "block";


    setExpiryDisabled(manualOn);

}


/* =========================================================
   GREY OUT / RE-ENABLE THE EXPIRY FIELD(S)
   AND ADDITIONAL STRIKES
========================================================= */

function setExpiryDisabled(disabled) {

    [
        "expiry",
        "sensexBankexExpiry",
        "additionalStrikes"
    ].forEach(id => {

        const el =
            document.getElementById(id);


        if (el) {

            el.disabled = disabled;

        }

    });


    document
        .querySelectorAll(
            "#symbolExpiryContainer input, " +
            "#symbolExpiryContainer select"
        )
        .forEach(el => {

            el.disabled = disabled;

        });


    document
        .querySelectorAll(
            "#commonExpiryRow label, " +
            "#sensexBankexExpiryRow label, " +
            "#additionalStrikeRow label, " +
            "#symbolExpiryContainer label"
        )
        .forEach(label => {

            label.classList.toggle("disabled", disabled);

        });

}


/* =========================================================
   BUILD CSV ROWS FROM MANUAL ENTRIES
========================================================= */

function buildManualRows(
    type,
    headerFields,
    client1,
    client2,
    currentDate,
    additionalStrikes
) {

    const entries =
        getManualEntries();


    /* =====================================================
       BASE ROW
       Every column that the user does not control
       (Sgmt, Src, member ids, etc.) is copied from
       the first template row of the segment.
    ===================================================== */

    const baseRow =
        rawCSVs[type]
            .trim()
            .split("\n")[1]
            .split(",");


    const idx = {};

    headerFields.forEach((h, i) => {

        idx[h.trim()] = i;

    });


    const set = (cols, name, value) => {

        if (idx[name] !== undefined) {

            cols[idx[name]] = value;

        }

    };


    const rows = [];


    for (const e of entries) {

        /* =================================================
           VALIDATION
        ================================================= */

        if (!e.expiry) {

            alert(
                "Please select an expiry for " +
                e.symbol +
                "."
            );

            return null;

        }


        if (!e.lot || e.lot <= 0) {

            alert(
                "Please enter a valid lot size for " +
                e.symbol +
                "."
            );

            return null;

        }


        if (e.isOption && e.strike === "") {

            alert(
                "Please enter a strike price for " +
                e.symbol +
                "."
            );

            return null;

        }


        const expiry =
            e.expiry.replaceAll("-", "");


        const qty =
            e.lot;


        const val =
            qty * e.price;


        /* =================================================
           STRIKE LIST
           Original strike + additional strikes
        ================================================= */

        const strikes = [];


        if (e.isOption) {

            const base =
                parseFloat(e.strike);


            strikes.push(base);


            if (e.step > 0 && additionalStrikes > 0) {

                for (let i = 1; i <= additionalStrikes; i++) {

                    strikes.push(
                        base + (e.step * i)
                    );

                }

            }

        }

        else {

            strikes.push(null);

        }


        /* =================================================
           BUILD ROWS
        ================================================= */

        strikes.forEach(strike => {

            const makeRow = (clientId, isBuy) => {

                const cols = [...baseRow];


                set(cols, "ClntId", clientId);

                set(cols, "FinInstrmTp", e.inst);

                set(cols, "TckrSymb", e.symbol);

                set(cols, "XpryDt", expiry);

                set(cols, "FininstrmActlXpryDt", expiry);

                set(cols, "RptgDt", currentDate);

                set(cols, "BizDt", currentDate);

                set(cols, "NewBrdLotQty", String(e.lot));


                set(
                    cols,
                    "StrkPric",
                    strike === null ? "" : String(strike)
                );


                set(
                    cols,
                    "OptnTp",
                    e.isOption ? e.optType : ""
                );


                /* opening positions */

                set(cols, "OpngLngQty", "0");

                set(cols, "OpngLngVal", "0");

                set(cols, "OpngShrtQty", "0");

                set(cols, "OpngShrtVal", "0");


                /* traded positions */

                set(cols, "OpnBuyTradgQty", isBuy ? String(qty) : "0");

                set(cols, "OpnBuyTradgVal", isBuy ? String(val) : "0");

                set(cols, "OpnSellTradgQty", isBuy ? "0" : String(qty));

                set(cols, "OpnSellTradgVal", isBuy ? "0" : String(val));


                /* pre exercise */

                set(cols, "PreExrcAssgndLngQty", "0");

                set(cols, "PreExrcAssgndLngVal", "0");

                set(cols, "PreExrcAssgndShrtQty", "0");

                set(cols, "PreExrcAssgndShrtVal", "0");


                set(cols, "ExrcdQty", "0");

                set(cols, "AssgndQty", "0");


                /* post exercise */

                set(cols, "PstExrcAssgndLngQty", isBuy ? String(qty) : "0");
				
				
                set(cols, "PstExrcAssgndLngVal", isBuy ? String(val) : "0");

                set(cols, "PstExrcAssgndShrtQty", isBuy ? "0" : String(qty));

                set(cols, "PstExrcAssgndShrtVal", isBuy ? "0" : String(val));


                /* settlement */

                set(cols, "SttlmPric", String(e.price));

                set(cols, "PrmAmt", "0");

                set(cols, "DalyMrkToMktSettlmVal", "0");

                set(cols, "FutrsFnlSttlmVal", "0");

                set(cols, "ExrcAssgndVal", "0");


                return cols.join(",");

            };


            if (uploadedClients.length > 0) {

                uploadedClients.forEach(c => {

                    rows.push(
                        makeRow(c, true)
                    );

                });

            }

            else {

                rows.push(
                    makeRow(client1, true)
                );


                if (client2) {

                    rows.push(
                        makeRow(client2, false)
                    );

                }

            }

        });

    }


    return rows;

}


/* =========================================================
   LOAD FILES
========================================================= */

function loadFiles() {

    const type =
        document.getElementById("positionType").value;


    const container =
        document.getElementById("checkboxContainer");


    const symbolExpiry =
        document.getElementById("symbolExpiryContainer");


    /* =========================
       RESET
    ========================== */

    symbolExpiry.style.display = "none";

    container.innerHTML = "";


    /*
     * Manual scrips are segment specific,
     * so they are cleared when the tab changes,
     * and the mode toggle resets back to Sample.
     */

    clearManualRows();

    closeManualDrawer();

    currentMode = "sample";


    document
        .querySelectorAll("#modeToggle input[type=radio]")
        .forEach(radio => {

            radio.checked =
                radio.dataset.mode === "sample";

        });


    refreshManualReopenLink();


    /* =====================================================
       FO
    ===================================================== */

    if (type === "FO") {

        container.classList.remove("hidden");


        container.innerHTML = `

            <div class="checkbox-item">

                <input
                    type="checkbox"
                    id="all"
                    onclick="toggleAll()">

                <label for="all">
                    Select All
                </label>

            </div>

        `;


        fileNames.FO.forEach(name => {

            container.innerHTML += `

                <div class="checkbox-item">

                    <input
                        type="checkbox"
                        class="f"
                        id="${name}"
                        value="${name}">

                    <label for="${name}">
                        ${shortFileLabel(name)}
                    </label>

                </div>

            `;

        });

    }


    /* =====================================================
       MCX / CD / NCDEX / NSECOM
    ===================================================== */

    else if (
        ["CD", "MCX", "NCDEX", "NSECOM"]
            .includes(type)
    ) {

        container.classList.add("hidden");


        buildExpiryInputs(type);

    }


    setSegmentVisibility(type);

}


/* =========================================================
   SELECT / UNSELECT ALL FO FILES
========================================================= */

function toggleAll() {

    const isChecked =
        document.getElementById("all").checked;


    document
        .querySelectorAll(".f")
        .forEach(cb => {

            cb.checked = isChecked;

        });

}


/* =========================================================
   RAW CSV TEMPLATES
========================================================= */

const rawCSVs = {


    /* =====================================================
       FO
    ===================================================== */

    FO: `Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4
FO,NCL,20260627,20260627,1,6538,6538,C,A101,STF,,SBIN,20250625,20250625,,,750,0,0,0,0,750,75000,0,0,0,0,0,0,0,0,750,75000,0,0,1000,,0,-3420,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,IDF,,BANKNIFTY,20250625,20250625,,,30,0,0,0,0,30,7500,0,0,0,0,0,0,0,0,30,7500,0,0,60000,,0,41880,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,IDO,,NIFTY,20250625,20250625,25000,CE,65,0,0,0,0,65,7500,0,0,0,0,0,0,0,0,65,7500,0,0,25000,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,STO,,TCS,20250625,20250625,3000,CE,225,0,0,0,0,225,63000,0,0,0,0,0,0,0,0,225,63000,0,0,3000,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,IDF,,SENSEX,20250625,20250625,,,20,0,0,0,0,100,7430000,0,0,0,0,0,0,0,0,100,7430000,0,0,77500,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,IDO,,SENSEX,20250625,20250625,78000,CE,20,0,0,0,0,100,1060447,0,0,0,0,0,0,0,0,100,1060447,0,0,1450,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A101,IDO,,BANKEX,20250625,20250625,60000,PE,30,0,0,0,0,150,107400,0,0,0,0,0,0,0,0,150,107400,0,0,716,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,STF,,SBIN,20250625,20250625,,,750,0,0,0,0,0,0,750,75000,0,0,0,0,0,0,0,0,750,75000,1000,,0,-3420,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,IDF,,BANKNIFTY,20250625,20250625,,,30,0,0,0,0,0,0,30,7500,0,0,0,0,0,0,0,0,30,7500,60000,,0,41880,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,IDO,,NIFTY,20250625,20250625,25000,CE,65,0,0,0,0,0,0,65,7500,0,0,0,0,0,0,0,0,65,7500,25000,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,STO,,TCS,20250625,20250625,3000,CE,225,0,0,0,0,0,0,225,63000,0,0,0,0,0,0,0,0,225,63000,3000,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,IDF,,SENSEX,20250625,20250625,,,20,0,0,0,0,0,0,100,7430000,0,0,0,0,0,0,0,0,100,7430000,77500,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,IDO,,SENSEX,20250625,20250625,78000,CE,20,0,0,0,0,0,0,100,1060447,0,0,0,0,0,0,0,0,100,1060447,1450,,0,0,0,0,,,,,
FO,NCL,20260627,20260627,1,6538,6538,C,A11,IDO,,BANKEX,20250625,20250625,60000,PE,30,0,0,0,0,0,0,150,107400,0,0,0,0,0,0,0,0,150,107400,716,,0,0,0,0,,,,,
`,



    /* =====================================================
       MCX
    ===================================================== */

    MCX: `Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,COF,,CRUDEOIL,30-06-2026,30-06-2026,,,1,0,0,0,0,1,654000,0,0,0,0,0,0,0,0,1,654000,0,0,6500,,0,0,,0,,,,,
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,COF,,SILVERM,25-06-2026,25-06-2026,,,1,0,0,0,0,1,1180525,0,0,0,0,0,0,0,0,1,1180525,0,0,230000,,0,0,,0,,,,,
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,FUO,,COPPER,23-06-2026,23-06-2026,1200,PE,1,0,0,0,0,1,15000,0,0,0,0,0,0,0,0,1,15000,0,0,6,,0,0,,0,,,,,
CO,MCXCCL,26-06-2025,26-06-2025,1,56645,56645,C,KS52,FUO,,NATURALGAS,30-06-2026,30-06-2026,300,CE,1,0,0,0,0,1,20000,0,0,0,0,0,0,0,0,1,20000,1,0,10,,0,0,,0,,,,,
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,COF,,CRUDEOIL,30-06-2026,30-06-2026,,,1,0,0,0,0,0,0,1,654000,0,0,0,0,0,0,0,0,1,654000,6500,,0,0,,0,,,,,
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,COF,,SILVERM,25-06-2026,25-06-2026,,,1,0,0,0,0,0,0,1,1180525,0,0,0,0,0,0,0,0,1,1180525,230000,,0,0,,0,,,,,
CO,MCXCCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,FUO,,COPPER,23-06-2026,23-06-2026,1200,PE,1,0,0,0,0,0,0,1,15000,0,0,0,0,0,0,0,0,1,15000,6,,0,0,,0,,,,,
CO,MCXCCL,26-06-2025,26-06-2025,1,56645,56645,C,KS53,FUO,,NATURALGAS,30-06-2026,30-06-2026,300,CE,1,0,0,0,0,0,0,1,20000,0,0,0,0,0,0,0,0,1,20000,10,,0,0,,0,,,,,
`,


    /* =====================================================
       CD
    ===================================================== */

    CD: `Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4
CD,NCL,09-05-2024,09-05-2024,1,M52040,90144,C,KS53,CDF,,GBPINR,12-06-2026,12-06-2026,,,1000,0,0,0,0,1,10000,0,0,0,0,0,0,0,0,1,10000,0,0,96,100.13,0,0,0,0,,,,,
CD,NCL,09-05-2024,09-05-2024,1,M52040,90144,C,KS53,CDO,,USDINR,12-06-2026,12-06-2026,94,CE,1000,0,0,0,0,1,1200,0,0,0,0,0,0,0,0,1,1200,0,0,96,100.13,0,0,0,0,,,,,
CD,NCL,09-05-2024,09-05-2024,1,M52040,90144,C,KS54,CDF,,GBPINR,12-06-2026,12-06-2026,,,1000,0,0,0,0,0,0,1,10000,0,0,0,0,0,0,0,0,1,10000,96,100.13,0,0,0,0,,,,,
CD,NCL,09-05-2024,09-05-2024,1,M52040,90144,C,KS54,CDO,,USDINR,12-06-2026,12-06-2026,94,CE,1000,0,0,0,0,0,0,1,1200,0,0,0,0,0,0,0,0,1,1200,96,100.13,0,0,0,0,,,,,
`,



    /* =====================================================
       NCDEX
    ===================================================== */

    NCDEX: `Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4
CO,NCCL,17-07-2026,17-07-2026,1,M51085,00094,C,KS52,COF,,COCUDAKL,20-08-2026,20-08-2026,0,,10,0,0,0,0,10,382100,0,0,0,0,0,0,0,0,10,382100,0,0,3805,,0,-1600,0,0,,,,,
CO,NCCL,17-07-2026,17-07-2026,1,M51085,00094,C,KS52,COF,,DHANIYA,20-08-2026,20-08-2026,0,,5,0,0,0,0,10,1559400,0,0,0,0,0,0,0,0,10,1559400,0,0,15886,,0,29200,0,0,,,,,
CO,NCCL,17-07-2026,17-07-2026,1,M51085,00094,C,KS53,COF,,COCUDAKL,20-08-2026,20-08-2026,0,,10,0,0,0,0,0,0,10,382100,0,0,0,0,0,0,0,0,10,382100,3805,,0,-1600,0,0,,,,,
CO,NCCL,17-07-2026,17-07-2026,1,M51085,00094,C,KS53,COF,,DHANIYA,20-08-2026,20-08-2026,0,,5,10,0,0,0,0,0,10,1559400,0,0,0,0,0,0,0,0,10,1559400,15886,,0,29200,0,0,,,,,

`,


    /* =====================================================
       NSE COM
    ===================================================== */

    NSECOM: `Sgmt,Src,RptgDt,BizDt,TradRegnOrgn,ClrMmbId,BrkrOrCtdnPtcptId,ClntTp,ClntId,FinInstrmTp,ISIN,TckrSymb,XpryDt,FininstrmActlXpryDt,StrkPric,OptnTp,NewBrdLotQty,OpngLngQty,OpngLngVal,OpngShrtQty,OpngShrtVal,OpnBuyTradgQty,OpnBuyTradgVal,OpnSellTradgQty,OpnSellTradgVal,PreExrcAssgndLngQty,PreExrcAssgndLngVal,PreExrcAssgndShrtQty,PreExrcAssgndShrtVal,ExrcdQty,AssgndQty,PstExrcAssgndLngQty,PstExrcAssgndLngVal,PstExrcAssgndShrtQty,PstExrcAssgndShrtVal,SttlmPric,RefRate,PrmAmt,DalyMrkToMktSettlmVal,FutrsFnlSttlmVal,ExrcAssgndVal,Rmks,Rsvd1,Rsvd2,Rsvd3,Rsvd4
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,COF,,CRUDEOIL,30-06-2026,30-06-2026,,,1,0,0,0,0,1,654000,0,0,0,0,0,0,0,0,1,654000,0,0,6500,,0,0,,0,,,,,
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,COF,,SILVERM,25-06-2026,25-06-2026,,,1,0,0,0,0,1,1180525,0,0,0,0,0,0,0,0,1,1180525,0,0,230000,,0,0,,0,,,,,
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS52,FUO,,COPPER,23-06-2026,23-06-2026,1200,PE,1,0,0,0,0,1,15000,0,0,0,0,0,0,0,0,1,15000,0,0,6,,0,0,,0,,,,,
CO,NCL,26-06-2025,26-06-2025,1,56645,56645,C,KS52,FUO,,NATURALGAS,30-06-2026,30-06-2026,300,CE,1,0,0,0,0,1,20000,0,0,0,0,0,0,0,0,1,20000,1,0,10,,0,0,,0,,,,,
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,COF,,CRUDEOIL,30-06-2026,30-06-2026,,,1,0,0,0,0,0,0,1,654000,0,0,0,0,0,0,0,0,1,654000,6500,,0,0,,0,,,,,
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,COF,,SILVERM,25-06-2026,25-06-2026,,,1,0,0,0,0,0,0,1,1180525,0,0,0,0,0,0,0,0,1,1180525,230000,,0,0,,0,,,,,
CO,NCL,22-01-2026,22-01-2026,1,56645,56645,C,KS53,FUO,,COPPER,23-06-2026,23-06-2026,1200,PE,1,0,0,0,0,0,0,1,15000,0,0,0,0,0,0,0,0,1,15000,6,,0,0,,0,,,,,
CO,NCL,26-06-2025,26-06-2025,1,56645,56645,C,KS53,FUO,,NATURALGAS,30-06-2026,30-06-2026,300,CE,1,0,0,0,0,0,0,1,20000,0,0,0,0,0,0,0,0,1,20000,10,,0,0,,0,,,,,
`

};


/* =========================================================
   CHANGE TAB
========================================================= */

function changeTab(type, btn) {

    document.getElementById("positionType").value = type;


    document
        .querySelectorAll(".tab")
        .forEach(t => {

            t.classList.remove("active");

        });


    btn.classList.add("active");


    loadFiles();

}


/* =========================================================
   GENERATE SEGMENT
========================================================= */

function generateSegment(
    type,
    client1,
    client2,
    expiryInput
) {

    const expiry =
        expiryInput
            ? expiryInput.replaceAll("-", "")
            : "";


    const rawCSV =
        rawCSVs[type];


    if (!rawCSV) {

        alert(
            type +
            " CSV template not found."
        );

        return;

    }


    const lines =
        rawCSV
            .trim()
            .split("\n");


    const header =
        lines[0];


    const data =
        lines.slice(1);


    const headerFields =
        header.split(",");


    const clntIdIdx =
        headerFields.indexOf("ClntId");


    const xpryIdx =
        headerFields.indexOf("XpryDt");


    const fxpryIdx =
        headerFields.indexOf(
            "FininstrmActlXpryDt"
        );


    const rptgIdx =
        headerFields.indexOf("RptgDt");


    const bizIdx =
        headerFields.indexOf("BizDt");


    const qtyIdx =
        headerFields.indexOf(
            "OpnBuyTradgQty"
        );


    const symbolIdx =
        headerFields.indexOf(
            "TckrSymb"
        );


    const strikeIdx =
        headerFields.indexOf(
            "StrkPric"
        );


    const optionTypeIdx =
        headerFields.indexOf(
            "OptnTp"
        );


    const now =
        new Date();


    const currentDate =
        now.getFullYear() +
        String(
            now.getMonth() + 1
        ).padStart(2, "0") +
        String(
            now.getDate()
        ).padStart(2, "0");


    const processedRows = [
        header
    ];


    /* =====================================================
       BSE EXPIRY
       SENSEX + BANKEX
    ===================================================== */

    const bseExpiryInput =
        document.getElementById(
            "sensexBankexExpiry"
        );


    const bseExpiry =
        bseExpiryInput &&
        bseExpiryInput.value
            ? bseExpiryInput.value
                .replaceAll("-", "")
            : "";


    /* =====================================================
       ADDITIONAL STRIKES
    ===================================================== */

    const additionalStrikes =
        parseInt(
            document.getElementById(
                "additionalStrikes"
            )?.value || "0",
            10
        );


    /* =====================================================
       MANUAL MODE

       If the user has entered any manual scrip,
       ONLY those scrips are written to the file and
       the built-in template positions are skipped.
    ===================================================== */

    /* =====================================================
       MANUAL MODE

       Driven by the Sample Positions / Manual Entry
       toggle at the top of the form. If Manual Entry is
       selected, ONLY the scrips entered in the popup are
       written to the file — the built-in template
       positions are skipped entirely.
    ===================================================== */

    const manualActive =
        currentMode === "manual";


    let manualRows = null;


    if (manualActive) {

        if (getManualEntries().length === 0) {

            alert(
                "Manual Entry is selected but no scrip " +
                "has been added yet. Open Manual Scrips " +
                "and add at least one, or switch back to " +
                "Sample Positions."
            );

            return;

        }


        manualRows =
            buildManualRows(
                type,
                headerFields,
                client1,
                client2,
                currentDate,
                additionalStrikes
            );


        /* validation failed inside buildManualRows */

        if (!manualRows) {

            return;

        }


        manualRows.forEach(r => {

            processedRows.push(r);

        });

    }


    /* =====================================================
       PROCESS EACH ROW
       (skipped entirely in manual mode)

       Wrapped in a function so it can be called once per
       uploaded client (each pass treating that client as
       "client1", buy-side only) without touching any of
       the logic inside — same tested behaviour, just run
       multiple times when a client list is loaded.
    ===================================================== */

    function processSampleRows(client1, client2) {

    for (const row of (manualActive ? [] : data)) {

        const cols =
            row.split(",");


        const qty =
            parseFloat(
                cols[qtyIdx]
            );


        /* =================================================
           CLIENT
        ================================================= */

        if (qty > 0) {

            cols[clntIdIdx] =
                client1;

        }

        else if (client2) {

            cols[clntIdIdx] =
                client2;

        }

        else {

            continue;

        }


        /* =================================================
           FO
        ================================================= */

        if (type === "FO") {

            const symbol =
                cols[symbolIdx];


            /* =============================================
               SENSEX + BANKEX
               BSE EXPIRY
            ============================================= */

            if (
                symbol === "SENSEX" ||
                symbol === "BANKEX"
            ) {

                if (!bseExpiry) {

                    continue;

                }


                cols[xpryIdx] =
                    bseExpiry;


                cols[fxpryIdx] =
                    bseExpiry;

            }


            /* =============================================
               OTHER FO
               NSE EXPIRY
            ============================================= */

            else {

                if (!expiry) {

                    continue;

                }


                cols[xpryIdx] =
                    expiry;


                cols[fxpryIdx] =
                    expiry;

            }


            /* =============================================
               FO OPTION STRIKE GENERATION
            ============================================= */

            const optionType =
                cols[optionTypeIdx];


            const originalStrike =
                parseFloat(
                    cols[strikeIdx]
                );


            const step =
                strikeSteps[symbol];


            const rowsToAdd = [
                cols
            ];


            if (
                ["CE", "PE"]
                    .includes(optionType) &&

                !isNaN(originalStrike) &&

                step &&

                additionalStrikes > 0
            ) {

                for (
                    let i = 1;
                    i <= additionalStrikes;
                    i++
                ) {

                    const newCols =
                        [...cols];


                    newCols[strikeIdx] =
                        String(
                            originalStrike +
                            (step * i)
                        );


                    rowsToAdd.push(
                        newCols
                    );

                }

            }


            /* =============================================
               ADD FO ROWS
            ============================================= */

            rowsToAdd.forEach(
                newCols => {

                    newCols[rptgIdx] =
                        currentDate;


                    newCols[bizIdx] =
                        currentDate;


                    processedRows.push(
                        newCols.join(",")
                    );

                }
            );

        }


        /* =================================================
           MCX / CD / NCDEX / NSECOM
        ================================================= */

        else {

            const symbol =
                cols[symbolIdx];


            /* =============================================
               SYMBOL-SPECIFIC EXPIRY
            ============================================= */

            const input =
                document.getElementById(
                    `exp_${type}_${symbol}`
                );


            if (!input) {

                continue;

            }


            if (!input.value) {

                continue;

            }


            const exp =
                input.value
                    .replaceAll("-", "");


            cols[xpryIdx] =
                exp;


            cols[fxpryIdx] =
                exp;


            cols[rptgIdx] =
                currentDate;


            cols[bizIdx] =
                currentDate;


            /* =============================================
               MCX ADDITIONAL STRIKES

               Only:
                 CE
                 PE

               Not:
                 COF
            ============================================= */

            if (type === "MCX") {

                const optionType =
                    cols[optionTypeIdx];


                const originalStrike =
                    parseFloat(
                        cols[strikeIdx]
                    );


                const step =
                    strikeSteps[symbol];


                const rowsToAdd = [
                    cols
                ];


                if (
                    ["CE", "PE"]
                        .includes(optionType) &&

                    !isNaN(originalStrike) &&

                    step &&

                    additionalStrikes > 0
                ) {

                    for (
                        let i = 1;
                        i <= additionalStrikes;
                        i++
                    ) {

                        const newCols =
                            [...cols];


                        newCols[strikeIdx] =
                            String(
                                originalStrike +
                                (step * i)
                            );


                        rowsToAdd.push(
                            newCols
                        );

                    }

                }


                /* =========================================
                   ADD MCX ROWS
                ========================================= */

                rowsToAdd.forEach(
                    newCols => {

                        processedRows.push(
                            newCols.join(",")
                        );

                    }
                );

            }


            /* =============================================
               OTHER SEGMENTS
               CD / NCDEX / NSECOM
            ============================================= */

            else {

                processedRows.push(
                    cols.join(",")
                );

            }

        }

    }


    }


    if (uploadedClients.length > 0) {

        uploadedClients.forEach(c => {

            processSampleRows(c, null);

        });

    }

    else {

        processSampleRows(client1, client2);

    }


    /* =====================================================
       FO FILE DOWNLOAD
    ===================================================== */

    if (type === "FO") {

        const selected =
            [
                ...document
                    .querySelectorAll(
                        ".f:checked"
                    )
            ];


        if (!selected.length) {

            alert(
                "Select at least one FO file."
            );

            return;

        }


        selected.forEach(cb => {

            const blob =
                new Blob(
                    [
                        processedRows.join(
                            "\n"
                        )
                    ],
                    {
                        type:
                            "text/csv"
                    }
                );


            const a =
                document.createElement(
                    "a"
                );


            a.href =
                URL.createObjectURL(
                    blob
                );


            a.download =
                cb.value +
                ".csv";


            a.click();


            URL.revokeObjectURL(
                a.href
            );

        });

    }


    /* =====================================================
       OTHER SEGMENTS
       MCX / CD / NCDEX / NSECOM
    ===================================================== */

    else {

        fileNames[type]
            .forEach(name => {

                const blob =
                    new Blob(
                        [
                            processedRows.join(
                                "\n"
                            )
                        ],
                        {
                            type:
                                "text/csv"
                        }
                    );


                const a =
                    document.createElement(
                        "a"
                    );


                a.href =
                    URL.createObjectURL(
                        blob
                    );


                a.download =
                    name +
                    ".csv";


                a.click();


                URL.revokeObjectURL(
                    a.href
                );

            });

    }

}


/* =========================================================
   BUILD SYMBOL EXPIRY INPUTS
========================================================= */

function buildExpiryInputs(type) {

    const div =
        document.getElementById(
            "symbolExpiryContainer"
        );


    div.innerHTML = "";


    const rawCSV =
        rawCSVs[type];


    if (!rawCSV) {

        return;

    }


    const lines =
        rawCSV
            .trim()
            .split("\n");


    const header =
        lines[0].split(",");


    const symbolIdx =
        header.indexOf(
            "TckrSymb"
        );


    const symbols =
        [
            ...new Set(
                lines
                    .slice(1)
                    .map(
                        r =>
                            r.split(",")[
                                symbolIdx
                            ]
                    )
            )
        ];


    symbols.forEach(
        symbol => {

            div.innerHTML += `

                <div class="form-row">

                    <label>

                        ${
                            symbol === "COPPER" ||
                            symbol === "NATURALGAS" ||
                            symbol === "USDINR"

                                ? `${symbol} (OPT):`

                                : `${symbol}:`
                        }

                    </label>

                    <input
                        type="date"
                        id="exp_${type}_${symbol}">

                </div>

            `;

        }
    );

}


/* =========================================================
   GENERATE
========================================================= */

function generate() {

    if (currentMode === "bulk") {
        generateBulk();
        return;
    }

    const client1 =
        document
            .getElementById(
                "client1"
            )
            .value
            .trim();


    const client2 =
        document
            .getElementById(
                "client2"
            )
            .value
            .trim();


    const nseExpiry =
        document
            .getElementById(
                "expiry"
            )
            .value;


    const bseExpiry =
        document
            .getElementById(
                "sensexBankexExpiry"
            )
            .value;


    const type =
        document
            .getElementById(
                "positionType"
            )
            .value;


    /* =====================================================
       CLIENT VALIDATION
    ===================================================== */

    if (uploadedClients.length === 0 && !client1) {

        alert(
            "Please provide Client 1, or upload a client list."
        );

        return;

    }


    /* =====================================================
       FO VALIDATION
    ===================================================== */

    if (type === "FO" && currentMode !== "manual") {

        if (!nseExpiry) {

            alert(
                "Please select NSE Expiry Date."
            );

            return;

        }


        if (!bseExpiry) {

            alert(
                "Please select BSE Expiry Date."
            );

            return;

        }

    }


    /* =====================================================
       GENERATE
    ===================================================== */

    generateSegment(
        type,
        client1,
        client2,
        nseExpiry
    );

}


/* =========================================================
   PAGE LOAD
========================================================= */

window.onload = function () {

    loadFiles();

};


				
