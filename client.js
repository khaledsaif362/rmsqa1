const BASE_PAN = "ABCDE1234Q";

let generatedFiles = {};

function generatePAN(index){

    const prefix = BASE_PAN.substring(0,5);

    const num = 1000 + index;

    const suffix = String.fromCharCode(65 + (index % 26));

    return prefix + String(num).padStart(4,'0') + suffix;
}

function getClientCodes(){

    const input = document.getElementById("clientCodes").value;

    return input
        .split("\n")
        .map(x => x.trim())
        .filter(x => x !== "");
}

function buildFiles(){

    const clientCodes = getClientCodes();

    if(clientCodes.length === 0){

        alert("Enter Client Codes");

        return null;
    }

    const date = new Date().toISOString().split("T")[0].replace(/-/g,'');

    let clientRows = [];

    let productRows = [];

    let exchangeRows = [];

    clientCodes.forEach((code,index)=>{

        const pan = generatePAN(index);

        // CLIENT FILE
        clientRows.push(
`${code}|TEST CLIENT ${index+1}|${code}|HO|||KS@gm.com|Borivali|Mumbai||1234567890||IN123456|NSDL|IN12345671234567||${pan}|NI|HDFC123456|560016|HDFC BANK|ANDHERI|A|Y|Y||||E||||24/12/1985||Q106|R16|IN12345678901234|A|B|C|7|Y|A123456789012345|||`
        );

        // PRODUCT FILE
        productRows.push(
`${code}|${code}|MIS|NRML|CNC|MTF|CO|BO`
        );

        // EXCHANGE FILE
const exchanges = [
    ["NSE", "111111111111"],
    ["BSEEQ", "1111111111111"],
    ["CDS", "111111111111"],
    ["NSEFO", "111111111111"],
    ["BCR", "1111111111111"],
    ["BSEFO", "1111111111111"],
    ["MCX", "111111111111"],
    ["NSEMF", "111111111111"],
    ["BSEMF", "1111111111111"],
    ["NCDX", "111111111111"],
    ["BSECOM", "1111111111111"]
];

exchanges.forEach(([ex, loc]) => {
    exchangeRows.push(`${code}||${ex}|||${loc}`);
});

});   // <-- ADD THIS LINE

generatedFiles = {

        "Client.txt":
`RUPEE|CLT|${date}
${clientRows.join("\n")}`,

        "Product_Allowed.txt":
`RUPEE|PROD_ALW|${date}
${productRows.join("\n")}`,

        "Exchange_Allowed.txt":
`RUPEE|EXCH_ALW|${date}
${exchangeRows.join("\n")}`
    };

    return generatedFiles;
}

function previewFiles(){

    const files = buildFiles();

    if(!files) return;

    let previewText = "";

    Object.keys(files).forEach(fileName => {

        previewText += `========== ${fileName} ==========\n\n`;

        previewText += files[fileName];

        previewText += "\n\n\n";

    });

    document.getElementById("preview").textContent = previewText;
}

async function generateAllFiles(){

    const files = buildFiles();

    if(!files) return;

    for(const fileName in files){

        const blob = new Blob(
            [files[fileName]],
            {type:"text/plain"}
        );

        const link = document.createElement("a");

        link.href = URL.createObjectURL(blob);

        link.download = fileName;

        document.body.appendChild(link);

        link.click();

        document.body.removeChild(link);

        await new Promise(resolve => setTimeout(resolve,200));
    }

    alert("All Files Generated Successfully");
}

function clearAll(){

    document.getElementById("clientCodes").value = "";

    document.getElementById("preview").textContent = "";

    generatedFiles = {};
}
