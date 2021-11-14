// equiv to npx inline-source --root data data/crash.html out-cli.html

const { inlineSource } = require('inline-source');
const fs = require('fs');
const path = require('path');

if (process.argv.length < 4) {
    console.log(`Usage
    ${process.argv[0]} ${process.argv[1]} [root <rootdir>] in.html out.html`);
    process.exit(1);
}

let htmlpath = path.resolve(process.argv[2]);
let root = path.resolve('.');
let out = path.resolve(process.argv[3]);
if (process.argv[2] == 'root') {
    root = path.resolve(process.argv[3]);
    htmlpath = path.resolve(process.argv[4]);
    out = path.resolve(process.argv[5]);
}

inlineSource(htmlpath, {
    compress: true,
    rootpath: root,
}).then((html) => {
    fs.writeFile(out, html, (err) => {
        if (err) {
            console.log("Failed:", err);
            process.exit(1);
        }
    });
}).catch((err) => {
    console.log("Failed:", err);
    process.exit(1);
});
