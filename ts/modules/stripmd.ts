const removeMd = require("remove-markdown");

export function stripMarkdown(md: string): string {
    let foundOpenSquare = false;
    let openSquare = -1;
	let openBracket = -1;
	let closeBracket = -1;
	let openSquares: number[] = [];
	let closeBrackets: number[] = [];
	let links: string[] = [];
	let foundOpen = false;
    for (let i = 0; i < md.length; i++) {
        const c = md.charAt(i);
		if (!foundOpenSquare && !foundOpen && c != '[' && c != ']') {
			continue;
		}
		if (c == '[' && md.charAt(i-1) != '!') {
			foundOpenSquare = true;
			openSquare = i;
		} else if (c == ']') {
			if (md.charAt(i+1) == '(') {
				foundOpenSquare = false;
				foundOpen = true;
				openBracket = i + 1;
				continue;
			}
		} else if (c == ')') {
			closeBracket = i;
			openSquares.push(openSquare);
			closeBrackets.push(closeBracket);
			links.push(md.slice(openBracket+1, closeBracket))
			openBracket = -1;
			closeBracket = -1;
			openSquare = -1;
			foundOpenSquare = false;
			foundOpen = false;
		}
	}
	let fullLinks: string[] = new Array(openSquares.length);
	for (let i = 0; i < openSquares.length; i++) {
		if (openSquares[i] != -1 && closeBrackets[i] != -1) {
			fullLinks[i] = md.slice(openSquares[i], closeBrackets[i]+1)
		}
	}
	for (let i = 0; i < openSquares.length; i++) {
        md = md.replace(fullLinks[i], links[i]);
	}
    return removeMd(md);
}
