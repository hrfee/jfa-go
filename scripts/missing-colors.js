let parser = require("jsdom");
let fs = require("fs");
let path = require("path");

const fixHTML = (infile, outfile) => {
    console.log(infile, outfile)
    let doc = new parser.JSDOM(fs.readFileSync(infile));
    for (let item of ["badge", "chip", "shield", "input", "table", "button", "portal", "select", "aside", "card"]) {
        let items = doc.window.document.body.querySelectorAll("."+item);
        for (let i = 0; i < items.length; i++) {
            let hasColor = false;
            for (let color of ["neutral", "positive", "urge", "warning", "info", "critical"]) {
                //console.log(color);
                if (items[i].classList.contains("~"+color)) {
                    hasColor = true;
                    // console.log("adding to", items[i].classList)
                    items[i].classList.add("dark:~d_"+color);
                    break;
                }
            }
            if (!hasColor) {
                items[i].classList.add("~neutral");
                items[i].classList.add("dark:~d_neutral");
            }
            if (!items[i].classList.contains("@low") && !items[i].classList.contains("@high")) {
                items[i].classList.add("@low");
            }
        }
    }
    fs.writeFileSync(outfile, doc.window.document.documentElement.outerHTML); 
};

let inpath = process.argv[process.argv.length-2];
let outpath = process.argv[process.argv.length-1];

let files = fs.readdirSync(inpath);
for (let i = 0; i < files.length; i++) {
    if (files[i].indexOf(".html")>=0) {
        fixHTML(path.join(inpath, files[i]), path.join(outpath, files[i]));
    }
}
