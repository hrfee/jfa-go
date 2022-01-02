let fs = require('fs');

let content = fs.readFileSync("out/index.html", 'utf8');

fs.writeFileSync("out/index.html", content.replace('<script src="main.js"></script>', '<script src="main.js"></script>'+process.env.INJECT));
