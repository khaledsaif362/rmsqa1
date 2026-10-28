
const foPositions = [
    {
        client: "A101",
        positions: [
            { instrument: "FUTSTK", symbol: "SBIN",     strike: "0",     option: "XX", qty: 750 },
            { instrument: "FUTIDX", symbol: "BANKNIFTY",strike: "0",     option: "XX", qty: 30 },
			{ instrument: "FUTIDX", symbol: "SENSEX",   strike: "0",     option: "XX", qty: 100 },
            { instrument: "OPTIDX", symbol: "NIFTY",    strike: "25000", option: "CE", qty: 65 },
            { instrument: "OPTSTK", symbol: "TCS",      strike: "3000",  option: "CE", qty: 175 },          
            { instrument: "OPTIDX", symbol: "SENSEX",   strike: "78000", option: "CE", qty: 100 },
            { instrument: "OPTIDX", symbol: "BANKEX",   strike: "60000", option: "PE", qty: 150 }
        ]
    },
    {
        client: "A11",
        positions: [
            { instrument: "FUTSTK", symbol: "SBIN",     strike: "0",     option: "XX", qty: 750 },
            { instrument: "FUTIDX", symbol: "BANKNIFTY",strike: "0",     option: "XX", qty: 30 },
			{ instrument: "FUTIDX", symbol: "SENSEX",   strike: "0",     option: "XX", qty: 100 },
            { instrument: "OPTIDX", symbol: "NIFTY",    strike: "25000", option: "CE", qty: 65 },
            { instrument: "OPTSTK", symbol: "TCS",      strike: "3000",  option: "CE", qty: 175 },           
            { instrument: "OPTIDX", symbol: "SENSEX",   strike: "78000", option: "CE", qty: 100 },
            { instrument: "OPTIDX", symbol: "BANKEX",   strike: "60000", option: "PE", qty: 150 }
        ]
    }
];

function displayName(position) {
    if (position.option === "CE" || position.option === "PE") {
        return `${position.symbol} ${position.strike} ${position.option}`;
    }
    return position.symbol;
}

function buildPriceInputs() {
    const area = document.getElementById("priceArea");
    const same = document.getElementById("sameAvg").checked;

    area.innerHTML = "";

    if (same) {
        const section = document.createElement("div");
        section.className = "client-section";

        const title = document.createElement("div");
        title.className = "client-title";
        title.textContent = "AVG COST PRICE — BOTH CLIENTS";
        section.appendChild(title);

        section.appendChild(createPriceRow(foPositions[0].positions, "BOTH"));
        area.appendChild(section);
        return;
    }

    foPositions.forEach(client => {
        const section = document.createElement("div");
        section.className = "client-section";

        const title = document.createElement("div");
        title.className = "client-title";
        title.textContent = client.client;
        section.appendChild(title);

        section.appendChild(createPriceRow(client.positions, client.client));
        area.appendChild(section);
    });
}

function createPriceRow(positions, clientKey) {
    const row = document.createElement("div");
    row.className = "price-row";

    positions.forEach((position, index) => {
        const box = document.createElement("div");
        box.className = "position-box";

        const label = document.createElement("div");
        label.className = "position-symbol";
        label.textContent = displayName(position);

        const input = document.createElement("input");
        input.className = "position-price";
        input.type = "number";
        input.step = "any";
        input.min = "0";
        input.placeholder = "Avg Price";
        input.dataset.client = clientKey;
        input.dataset.index = index;

        box.appendChild(label);
        box.appendChild(input);
        row.appendChild(box);
    });

    return row;
}

function toggleSamePrice() {
    buildPriceInputs();
}

function formatDate(dateValue) {
    if (!dateValue) return "";
    const parts = dateValue.split("-");
    if (parts.length !== 3) return "";
    return `${parts[1]}/${parts[2]}/${parts[0]}`;
}

function generateAvgTradePrice() {
    const nseExpiry = formatDate(document.getElementById("nseExpiry").value);
    const bseExpiry = formatDate(document.getElementById("bseExpiry").value);
    const same = document.getElementById("sameAvg").checked;

    if (!nseExpiry || !bseExpiry) {
        alert("Please select NSE Expiry and BSE Expiry.");
        return;
    }

    const lines = [
        "UCC|Instrument_Type|Symbol|Expiry_Date|Strike_Price|Option_Type|Price|Quantity"
    ];

    if (same) {
        const inputs = document.querySelectorAll('#priceArea input[data-client="BOTH"]');

        inputs.forEach((input, index) => {
            const price = input.value.trim();
            if (price === "") return;

            foPositions.forEach(client => {
                const p = client.positions[index];
                const expiry = (p.symbol === "SENSEX" || p.symbol === "BANKEX") ? bseExpiry : nseExpiry;

                lines.push([
                    client.client,
                    p.instrument,
                    p.symbol,
                    expiry,
                    p.strike,
                    p.option,
                    price,
                    p.qty
                ].join("|"));
            });
        });
    } else {
        foPositions.forEach(client => {
            const inputs = document.querySelectorAll(`#priceArea input[data-client="${client.client}"]`);

            inputs.forEach((input, index) => {
                const price = input.value.trim();
                if (price === "") return;

                const p = client.positions[index];
                const expiry = (p.symbol === "SENSEX" || p.symbol === "BANKEX") ? bseExpiry : nseExpiry;

                lines.push([
                    client.client,
                    p.instrument,
                    p.symbol,
                    expiry,
                    p.strike,
                    p.option,
                    price,
                    p.qty
                ].join("|"));
            });
        });
    }

    if (lines.length === 1) {
        alert("Enter at least one AVG Price.");
        return;
    }

    const blob = new Blob([lines.join("\n") + "\n"], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "AVG_TRD_PRICE.txt";
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
}

window.addEventListener("DOMContentLoaded", buildPriceInputs);
