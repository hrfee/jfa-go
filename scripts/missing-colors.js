let parser = require("cheerio");
let fs = require("fs");
let path = require("path");
let pre = require("perl-regex");

const template = process.env.NOTEMPLATE != "1";

const hasDark = (item) => {
    let list = item.attr("class").split(/\s+/);
    for (let i = 0; i < list.length; i++) {
        if (list[i].substring(0,5) == "dark:") {
            return true;
        }
    }
    return false;
};

if (typeof String.prototype.replaceAll === "undefined") {
    String.prototype.replaceAll = function(match, replace) {
       return this.replace(new RegExp(match, 'g'), () => replace);
    }
}

function fixHTML(infile, outfile) {
    let f = fs.readFileSync(infile).toString();
    // Find all go template strings ({{ example }})
    let templateStrings = pre.exec(f, "(?s){{(?:(?!{{).)*?}}", "gi");
    if (template) {
        for (let i = 0; i < templateStrings.length; i++) {
            let s = templateStrings[i].replace(/\\/g, '');
            // let s = templateStrings[i];
            f = f.replaceAll(s, "<!--" + s.slice(3).slice(0, -3) + "-->");
        }
    }
    let doc = new parser.load(f);
    for (let item of ["badge", "chip", "shield", "input", "table", "button", "portal", "select", "aside", "card", "field", "textarea"]) {
        let items = doc("."+item);
        items.each((i, elem) => {
            let hasColor = false;
            for (let color of ["neutral", "positive", "urge", "warning", "info", "critical"]) {
                //console.log(color);
                if (doc(elem).hasClass("~"+color)) {
                    hasColor = true;
                    // console.log("adding to", items[i].classList)
                    if (!hasDark(doc(elem))) {
                        doc(elem).addClass("dark:~d_"+color);
                    }
                    break;
                }
            }
            if (!hasColor) {
                if (!hasDark(doc(elem))) {
                    // card without ~neutral look different than with.
                    if (item != "card") doc(elem).addClass("~neutral");
                    doc(elem).addClass("dark:~d_neutral");
                }
            }
            if (!doc(elem).hasClass("@low") && !doc(elem).hasClass("@high")) {
                doc(elem).addClass("@low");
            }
        });
    }
    let out = doc.html();
    // let out = f
    if (template) {
        for (let i = 0; i < templateStrings.length; i++) {
            let s = templateStrings[i].replace(/\\/g, '');
            out = out.replaceAll("<!--" + s.slice(3).slice(0, -3) + "-->", s);
        }
    out = out.replaceAll("&lt;!--", "{{");
    out = out.replaceAll("--&gt;", "}}");
    }
    fs.writeFileSync(outfile, out);
    console.log(infile, outfile);
};

let inpath = process.argv[process.argv.length-2];
let outpath = process.argv[process.argv.length-1];

if (fs.statSync(inpath).isDirectory()) {
    let files = fs.readdirSync(inpath);
    for (let i = 0; i < files.length; i++) {
        if (files[i].indexOf(".html")>=0) {
            fixHTML(path.join(inpath, files[i]), path.join(outpath, files[i]));
        }
    }
} else {
    fixHTML(inpath, outpath);
}
