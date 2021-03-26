const removeMd = require("remove-markdown");

function stripAltText(md: string): string {
	let altStart = -1; // Start of alt text (between '[' & ']')
	let urlStart = -1; // Start of url (between '(' & ')')
	let urlEnd = -1;
	let prevURLEnd = -2;
	let out = "";
    for (let i = 0; i < md.length; i++) {
		if (altStart != -1 && urlStart != -1 && md.charAt(i) == ')') {
			urlEnd = i - 1;
            out += md.substring(prevURLEnd+2, altStart-1) + md.substring(urlStart, urlEnd+1);
			prevURLEnd = urlEnd;
			altStart = -1;
            urlStart = -1;
            urlEnd = -1;
			continue;
		}
		if (md.charAt(i) == '[' && altStart == -1) {
			altStart = i + 1
			if (i > 0 && md.charAt(i-1) == '!') {
				altStart--
			}
		}
		if (i > 0 && md.charAt(i-1) == ']' && md.charAt(i) == '(' && urlStart == -1) {
			urlStart = i + 1
		}
	}
    if (prevURLEnd + 1 != md.length - 1) {
        out += md.substring(prevURLEnd+2)
    }
    if (out == "") {
        return md
    }
	return out
}

export function stripMarkdown(md: string): string {
    return removeMd(stripAltText(md));
}
