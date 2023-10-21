const dateParser = require("any-date-parser");

export interface QueryType {
    name: string;
    getter: string;
    bool: boolean;
    string: boolean;
    date: boolean;
    dependsOnElement?: string; // Format for querySelector
    show?: boolean;
}

export interface SearchConfiguration {
    filterArea: HTMLElement;
    sortingByButton: HTMLButtonElement;
    searchOptionsHeader: HTMLElement;
    notFoundPanel: HTMLElement;
    clearSearchButtonSelector: string;
    search: HTMLInputElement;
    queries: { [field: string]: QueryType };
    setVisibility: (items: string[], visible: boolean) => void;
    onSearchCallback: () => void;
}

export interface SearchableItem {
    matchesSearch: (query: string) => boolean;
}

export class Search {
    private _c: SearchConfiguration;
    private _ordering: string[] = [];
    private _items: { [id: string]: SearchableItem };
    inSearch: boolean;

    search = (query: String): string[] => {
        this._c.filterArea.textContent = "";

        query = query.toLowerCase();

        let result: string[] = [...this._ordering];
        let words: string[] = [];
        
        let quoteSymbol = ``;
        let queryStart = -1;
        let lastQuote = -1;
        for (let i = 0; i < query.length; i++) {
            if (queryStart == -1 && query[i] != " " && query[i] != `"` && query[i] != `'`) {
                queryStart = i;
            }
            if ((query[i] == `"` || query[i] == `'`) && (quoteSymbol == `` || query[i] == quoteSymbol)) {
                if (lastQuote != -1) {
                    lastQuote = -1;
                    quoteSymbol = ``;
                } else {
                    lastQuote = i;
                    quoteSymbol = query[i];
                }
            }

            if (query[i] == " " || i == query.length-1) {
                if (lastQuote != -1) {
                    continue;
                } else {
                    let end = i+1;
                    if (query[i] == " ") {
                        end = i;
                        while (i+1 < query.length && query[i+1] == " ") {
                            i += 1;
                        }
                    }
                    words.push(query.substring(queryStart, end).replace(/['"]/g, ""));
                    console.log("pushed", words);
                    queryStart = -1;
                }
            }
        }

        query = "";
        for (let word of words) {
            if (!word.includes(":")) {
                let cachedResult = [...result];
                for (let id of cachedResult) {
                    const u = this._items[id];
                    if (!u.matchesSearch(word)) {
                        result.splice(result.indexOf(id), 1);
                    }
                }
                continue;
            }
            const split = [word.substring(0, word.indexOf(":")), word.substring(word.indexOf(":")+1)];
            
            if (!(split[0] in this._c.queries)) continue;

            const queryFormat = this._c.queries[split[0]];

            if (queryFormat.bool) {
                let isBool = false;
                let boolState = false;
                if (split[1] == "true" || split[1] == "yes" || split[1] == "t" || split[1] == "y") {
                    isBool = true;
                    boolState = true;
                } else if (split[1] == "false" || split[1] == "no" || split[1] == "f" || split[1] == "n") {
                    isBool = true;
                    boolState = false;
                }
                if (isBool) {
                    const filterCard = document.createElement("span");
                    filterCard.ariaLabel = window.lang.strings("clickToRemoveFilter");
                    filterCard.classList.add("button", "~" + (boolState ? "positive" : "critical"), "@high", "center", "mx-2", "h-full");
                    filterCard.innerHTML = `
                    <span class="font-bold mr-2">${queryFormat.name}</span>
                    <i class="text-2xl ri-${boolState? "checkbox" : "close"}-circle-fill"></i>
                    `;

                    filterCard.addEventListener("click", () => {
                        for (let quote of [`"`, `'`, ``]) {
                            this._c.search.value = this._c.search.value.replace(split[0] + ":" + quote + split[1] + quote, "");
                        }
                        this._c.search.oninput((null as Event));
                    })

                    this._c.filterArea.appendChild(filterCard);

                    // console.log("is bool, state", boolState);
                    // So removing elements doesn't affect us
                    let cachedResult = [...result];
                    for (let id of cachedResult) {
                        const u = this._items[id];
                        const value = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u);
                        // console.log("got", queryFormat.getter + ":", value);
                        // Remove from result if not matching query
                        if (!((value && boolState) || (!value && !boolState))) {
                            // console.log("not matching, result is", result);
                            result.splice(result.indexOf(id), 1);
                        }
                    }
                    continue
                }
            }
            if (queryFormat.string) {
                const filterCard = document.createElement("span");
                filterCard.ariaLabel = window.lang.strings("clickToRemoveFilter");
                filterCard.classList.add("button", "~neutral", "@low", "center", "mx-2", "h-full");
                filterCard.innerHTML = `
                <span class="font-bold mr-2">${queryFormat.name}:</span> "${split[1]}"
                `;

                filterCard.addEventListener("click", () => {
                    for (let quote of [`"`, `'`, ``]) {
                        let regex = new RegExp(split[0] + ":" + quote + split[1] + quote, "ig");
                        this._c.search.value = this._c.search.value.replace(regex, "");
                    }
                    this._c.search.oninput((null as Event));
                })

                this._c.filterArea.appendChild(filterCard);

                let cachedResult = [...result];
                for (let id of cachedResult) {
                    const u = this._items[id];
                    const value = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u);
                    if (!(value.includes(split[1]))) {
                        result.splice(result.indexOf(id), 1);
                    }
                }
                continue;
            }
            if (queryFormat.date) {
                // -1 = Before, 0 = On, 1 = After, 2 = No symbol, assume 0
                let compareType = (split[1][0] == ">") ? 1 : ((split[1][0] == "<") ? -1 : ((split[1][0] == "=") ? 0 : 2));
                let unmodifiedValue = split[1];
                if (compareType != 2) {
                    split[1] = split[1].substring(1);
                }
                if (compareType == 2) compareType = 0;

                let attempt: { year?: number, month?: number, day?: number, hour?: number, minute?: number } = dateParser.attempt(split[1]);
                // Month in Date objects is 0-based, so make our parsed date that way too
                if ("month" in attempt) attempt.month -= 1;

                let date: Date = (Date as any).fromString(split[1]) as Date;
                console.log("Read", attempt, "and", date);
                if ("invalid" in (date as any)) continue;

                const filterCard = document.createElement("span");
                filterCard.ariaLabel = window.lang.strings("clickToRemoveFilter");
                filterCard.classList.add("button", "~neutral", "@low", "center", "m-2", "h-full");
                filterCard.innerHTML = `
                <span class="font-bold mr-2">${queryFormat.name}:</span> ${(compareType == 1) ? window.lang.strings("after")+" " : ((compareType == -1) ? window.lang.strings("before")+" " : "")}${split[1]}
                `;
                
                filterCard.addEventListener("click", () => {
                    for (let quote of [`"`, `'`, ``]) {
                        let regex = new RegExp(split[0] + ":" + quote + unmodifiedValue + quote, "ig");
                        this._c.search.value = this._c.search.value.replace(regex, "");
                    }
                    
                    this._c.search.oninput((null as Event));
                })
                
                this._c.filterArea.appendChild(filterCard);

                let cachedResult = [...result];
                for (let id of cachedResult) {
                    const u = this._items[id];
                    const unixValue = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u);
                    if (unixValue == 0) {
                        result.splice(result.indexOf(id), 1);
                        continue;
                    }
                    let value = new Date(unixValue*1000);
                    
                    const getterPairs: [string, () => number][] = [["year", Date.prototype.getFullYear], ["month", Date.prototype.getMonth], ["day", Date.prototype.getDate], ["hour", Date.prototype.getHours], ["minute", Date.prototype.getMinutes]];

                    // When doing > or < <time> with no date, we need to ignore the rest of the Date object
                    if (compareType != 0 && Object.keys(attempt).length == 2 && "hour" in attempt && "minute" in attempt) { 
                        const temp = new Date(date.valueOf());
                        temp.setHours(value.getHours(), value.getMinutes());
                        value = temp;
                        console.log("just hours/minutes workaround, value set to", value);
                    }


                    let match = true;
                    if (compareType == 0) {
                        for (let pair of getterPairs) {
                            if (pair[0] in attempt) {
                                if (compareType == 0 && attempt[pair[0]] != pair[1].call(value)) {
                                    match = false;
                                    break;
                                }
                            }
                        }
                    } else if (compareType == -1) {
                        match = (value < date);
                    } else if (compareType == 1) {
                        match = (value > date);
                    }
                    if (!match) {
                        result.splice(result.indexOf(id), 1);
                    }
                }
            }
        }
        return result;
    }
    
    showHideSearchOptionsHeader = () => {
        const sortingBy = !(this._c.sortingByButton.parentElement.classList.contains("hidden"));
        const hasFilters = this._c.filterArea.textContent != "";
        console.log("sortingBy", sortingBy, "hasFilters", hasFilters);
        if (sortingBy || hasFilters) {
            this._c.searchOptionsHeader.classList.remove("hidden");
        } else {
            this._c.searchOptionsHeader.classList.add("hidden");
        }
    }


    get items(): { [id: string]: SearchableItem } { return this._items; }
    set items(v: { [id: string]: SearchableItem }) {
        this._items = v;
    }

    get ordering(): string[] { return this._ordering; }
    set ordering(v: string[]) { this._ordering = v; }

    onSearchBoxChange = () => {
        const query = this._c.search.value;
        if (!query) {
            this.inSearch = false;
        } else {
            this.inSearch = true;
        }
        const results = this.search(query);
        this._c.setVisibility(results, true);
        this._c.onSearchCallback();
        this.showHideSearchOptionsHeader();
        if (results.length == 0) {
            this._c.notFoundPanel.classList.remove("unfocused");
        } else {
            this._c.notFoundPanel.classList.add("unfocused");
        }
    }

    constructor(c: SearchConfiguration) {
        this._c = c;

        this._c.search.oninput = this.onSearchBoxChange;

        const clearSearchButtons = Array.from(document.querySelectorAll(this._c.clearSearchButtonSelector)) as Array<HTMLSpanElement>;
        for (let b of clearSearchButtons) {
            b.addEventListener("click", () => {
                this._c.search.value = "";
                this.onSearchBoxChange();
            });
        }
    }
}
