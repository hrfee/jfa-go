const dateParser = require("any-date-parser");

declare var window: GlobalWindow;

export enum QueryOperator {
    Greater = ">",
    Lower = "<",
    Equal = "="
}

export function QueryOperatorToDateText(op: QueryOperator): string {
    switch (op) {
        case QueryOperator.Greater:
            return window.lang.strings("after");
        case QueryOperator.Lower:
            return window.lang.strings("before");
        default:
            return "";
    }
}

export interface QueryType {
    name: string;
    description?: string;
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
    notFoundCallback?: (notFound: boolean) => void;
    filterList: HTMLElement;
    clearSearchButtonSelector: string;
    search: HTMLInputElement;
    queries: { [field: string]: QueryType };
    setVisibility: (items: string[], visible: boolean) => void;
    onSearchCallback: (visibleCount: number, newItems: boolean, loadAll: boolean) => void;
    loadMore?: () => void;
}

export abstract class Query {
    protected _subject: QueryType;
    protected _operator: QueryOperator;
    protected _card: HTMLElement;

    constructor(subject: QueryType, operator: QueryOperator) {
        this._subject = subject;
        this._operator = operator;
        this._card = document.createElement("span");
        this._card.ariaLabel = window.lang.strings("clickToRemoveFilter");
    }

    set onclick(v: () => void) {
        this._card.addEventListener("click", v);
    }

    asElement(): HTMLElement { return this._card; }
}


export class BoolQuery extends Query {
    protected _value: boolean;
    constructor(subject: QueryType, value: boolean) {
        super(subject, QueryOperator.Equal);
        this._value = value;
        this._card.classList.add("button", "~" + (this._value ? "positive" : "critical"), "@high", "center", "mx-2", "h-full");
        this._card.innerHTML = `
        <span class="font-bold mr-2">${subject.name}</span>
        <i class="text-2xl ri-${this._value? "checkbox" : "close"}-circle-fill"></i>
        `;
    }

    public static paramsFromString(valueString: string): [boolean, boolean] {
        let isBool = false;
        let boolState = false;
        if (valueString == "true" || valueString == "yes" || valueString == "t" || valueString == "y") {
            isBool = true;
            boolState = true;
        } else if (valueString == "false" || valueString == "no" || valueString == "f" || valueString == "n") {
            isBool = true;
            boolState = false;
        }
        return [boolState, isBool]
    }

    get value(): boolean { return this._value; }

    // Ripped from old code. Why it's like this, I don't know
    public compare(subjectBool: boolean): boolean {
        return ((subjectBool && this._value) || (!subjectBool && !this._value))
    }
}

export class StringQuery extends Query {
    protected _value: string;
    constructor(subject: QueryType, value: string) {
        super(subject, QueryOperator.Equal);
        this._value = value;
        this._card.classList.add("button", "~neutral", "@low", "center", "mx-2", "h-full");
        this._card.innerHTML = `
        <span class="font-bold mr-2">${subject.name}:</span> "${this._value}"
        `;
    }

    get value(): string { return this._value; }
}

export interface DateAttempt {
    year?: number;
    month?: number;
    day?: number;
    hour?: number;
    minute?: number
}

export interface ParsedDate {
    attempt: DateAttempt;
    date: Date;
    text: string;
};
    
const dateGetters: Map<string, () => number> = (() => {
    let m = new Map<string, () => number>();
    m.set("year", Date.prototype.getFullYear);
    m.set("month", Date.prototype.getMonth);
    m.set("day", Date.prototype.getDate);
    m.set("hour", Date.prototype.getHours);
    m.set("minute", Date.prototype.getMinutes);
    return m;
})();
const dateSetters: Map<string, (v: number) => void> = (() => {
    let m = new Map<string, (v: number) => void>();
    m.set("year", Date.prototype.setFullYear);
    m.set("month", Date.prototype.setMonth);
    m.set("day", Date.prototype.setDate);
    m.set("hour", Date.prototype.setHours);
    m.set("minute", Date.prototype.setMinutes);
    return m;
})();

export class DateQuery extends Query {
    protected _value: ParsedDate;

    constructor(subject: QueryType, operator: QueryOperator, value: ParsedDate) {
        super(subject, operator);
        this._value = value;
        console.log("op:", operator, "date:", value);
        this._card.classList.add("button", "~neutral", "@low", "center", "m-2", "h-full");
        let dateText = QueryOperatorToDateText(operator);
        this._card.innerHTML = `
        <span class="font-bold mr-2">${subject.name}:</span> ${dateText != "" ? dateText+" " : ""}${value.text}
        `;
    }
    public static paramsFromString(valueString: string): [ParsedDate, QueryOperator, boolean] {
        // FIXME: Validate this!
        let op = QueryOperator.Equal;
        if ((Object.values(QueryOperator) as string[]).includes(valueString.charAt(0))) {
            op = valueString.charAt(0) as QueryOperator;
            // Trim the operator from the string
            valueString = valueString.substring(1);
        }

        let out: ParsedDate = {
            text: valueString,
            // Used just to tell use what fields the user passed.
            attempt: dateParser.attempt(valueString),
            // note Date.fromString is also provided by dateParser.
            date: (Date as any).fromString(valueString) as Date
        };
        // Month in Date objects is 0-based, so make our parsed date that way too
        if ("month" in out.attempt) out.attempt.month -= 1;
        let isValid = true;
        if ("invalid" in (out.date as any)) { isValid = false; };
        
        return [out, op, isValid];
    }

    get value(): ParsedDate { return this._value; }

    public compare(subjectDate: Date): boolean {
        // We want to compare only the fields given in this._value,
        // so we copy subjectDate and apply on those fields from this._value.
        const temp = new Date(subjectDate.valueOf());
        for (let [field] of dateGetters) {
            if (field in this._value.attempt) {
                dateSetters.get(field).call(
                    temp,
                    dateGetters.get(field).call(this._value.date)
                );
            }
        }

        if (this._operator == QueryOperator.Equal) {
            return subjectDate.getTime() == temp.getTime();
        } else if (this._operator == QueryOperator.Lower) {
            return subjectDate < temp;
        }
        return subjectDate > temp;
    }
}


// FIXME: Continue taking stuff from search function, making XQuery classes!



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

            let formattedQuery = []

            if (queryFormat.bool) {
                let [boolState, isBool] = BoolQuery.paramsFromString(split[1]);
                if (isBool) {
                    let q = new BoolQuery(queryFormat, boolState);
                    q.onclick = () => {
                        for (let quote of [`"`, `'`, ``]) {
                            this._c.search.value = this._c.search.value.replace(split[0] + ":" + quote + split[1] + quote, "");
                        }
                        this._c.search.oninput((null as Event));
                    };

                    this._c.filterArea.appendChild(q.asElement());

                    // console.log("is bool, state", boolState);
                    // So removing elements doesn't affect us
                    let cachedResult = [...result];
                    for (let id of cachedResult) {
                        const u = this._items[id];
                        const value = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u);
                        // console.log("got", queryFormat.getter + ":", value);
                        // Remove from result if not matching query
                        if (!q.compare(value)) {
                            // console.log("not matching, result is", result);
                            result.splice(result.indexOf(id), 1);
                        }
                    }
                    continue
                }
            }
            if (queryFormat.string) {
                const q = new StringQuery(queryFormat, split[1]);

                q.onclick = () => {
                    for (let quote of [`"`, `'`, ``]) {
                        let regex = new RegExp(split[0] + ":" + quote + split[1] + quote, "ig");
                        this._c.search.value = this._c.search.value.replace(regex, "");
                    }
                    this._c.search.oninput((null as Event));
                }

                this._c.filterArea.appendChild(q.asElement());

                let cachedResult = [...result];
                for (let id of cachedResult) {
                    const u = this._items[id];
                    const value = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u).toLowerCase();
                    if (!(value.includes(split[1]))) {
                        result.splice(result.indexOf(id), 1);
                    }
                }
                continue;
            }
            if (queryFormat.date) {
                let [parsedDate, op, isDate] = DateQuery.paramsFromString(split[1]);
                if (!isDate) continue;
                const q = new DateQuery(queryFormat, op, parsedDate);
                
                q.onclick = () => {
                    for (let quote of [`"`, `'`, ``]) {
                        let regex = new RegExp(split[0] + ":" + quote + split[1] + quote, "ig");
                        this._c.search.value = this._c.search.value.replace(regex, "");
                    }
                    
                    this._c.search.oninput((null as Event));
                }
                
                this._c.filterArea.appendChild(q.asElement());

                let cachedResult = [...result];
                for (let id of cachedResult) {
                    const u = this._items[id];
                    const unixValue = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(u), queryFormat.getter).get.call(u);
                    if (unixValue == 0) {
                        result.splice(result.indexOf(id), 1);
                        continue;
                    }
                    let value = new Date(unixValue*1000);

                    let match = q.compare(value);
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

    onSearchBoxChange = (newItems: boolean = false, loadAll: boolean = false) => {
        const query = this._c.search.value;
        if (!query) {
            this.inSearch = false;
        } else {
            this.inSearch = true;
        }
        const results = this.search(query);
        this._c.setVisibility(results, true);
        this._c.onSearchCallback(results.length, newItems, loadAll);
        this.showHideSearchOptionsHeader();
        if (results.length == 0) {
            this._c.notFoundPanel.classList.remove("unfocused");
        } else {
            this._c.notFoundPanel.classList.add("unfocused");
        }
        if (this._c.notFoundCallback) this._c.notFoundCallback(results.length == 0);
    }

    fillInFilter = (name: string, value: string, offset?: number) => {
        this._c.search.value = name + ":" + value + " " + this._c.search.value;
        this._c.search.focus();
        let newPos = name.length + 1 + value.length;
        if (typeof offset !== 'undefined')
            newPos += offset;
        this._c.search.setSelectionRange(newPos, newPos);
        this._c.search.oninput(null as any);
    };
    


    generateFilterList = () => {
        // Generate filter buttons
        for (let queryName of Object.keys(this._c.queries)) {
            const query = this._c.queries[queryName];
            if ("show" in query && !query.show) continue;
            if ("dependsOnElement" in query && query.dependsOnElement) {
                const el = document.querySelector(query.dependsOnElement);
                if (el === null) continue;
            }

            const container = document.createElement("span") as HTMLSpanElement;
            container.classList.add("button", "button-xl", "~neutral", "@low", "mb-1", "mr-2", "align-bottom");
            container.innerHTML = `
            <div class="flex flex-col mr-2">
                <span>${query.name}</span>
                <span class="support">${query.description || ""}</span>
            </div>
            `;
            if (query.bool) {
                const pos = document.createElement("button") as HTMLButtonElement;
                pos.type = "button";
                pos.ariaLabel = `Filter by "${query.name}": True`;
                pos.classList.add("button", "~positive", "ml-2");
                pos.innerHTML = `<i class="ri-checkbox-circle-fill"></i>`;
                pos.addEventListener("click", () => this.fillInFilter(queryName, "true"));
                const neg = document.createElement("button") as HTMLButtonElement;
                neg.type = "button";
                neg.ariaLabel = `Filter by "${query.name}": False`;
                neg.classList.add("button", "~critical", "ml-2");
                neg.innerHTML = `<i class="ri-close-circle-fill"></i>`;
                neg.addEventListener("click", () => this.fillInFilter(queryName, "false"));

                container.appendChild(pos);
                container.appendChild(neg);
            }
            if (query.string) {
                const button = document.createElement("button") as HTMLButtonElement;
                button.type = "button";
                button.classList.add("button", "~urge", "ml-2");
                button.innerHTML = `<i class="ri-equal-line mr-2"></i>${window.lang.strings("matchText")}`;

                // Position cursor between quotes
                button.addEventListener("click", () => this.fillInFilter(queryName, `""`, -1));
                
                container.appendChild(button);
            }
            if (query.date) {
                const onDate = document.createElement("button") as HTMLButtonElement;
                onDate.type = "button";
                onDate.classList.add("button", "~urge", "ml-2");
                onDate.innerHTML = `<i class="ri-calendar-check-line mr-2"></i>On Date`;
                onDate.addEventListener("click", () => this.fillInFilter(queryName, `"="`, -1));

                const beforeDate = document.createElement("button") as HTMLButtonElement;
                beforeDate.type = "button";
                beforeDate.classList.add("button", "~urge", "ml-2");
                beforeDate.innerHTML = `<i class="ri-calendar-check-line mr-2"></i>Before Date`;
                beforeDate.addEventListener("click", () => this.fillInFilter(queryName, `"<"`, -1));

                const afterDate = document.createElement("button") as HTMLButtonElement;
                afterDate.type = "button";
                afterDate.classList.add("button", "~urge", "ml-2");
                afterDate.innerHTML = `<i class="ri-calendar-check-line mr-2"></i>After Date`;
                afterDate.addEventListener("click", () => this.fillInFilter(queryName, `">"`, -1));
                
                container.appendChild(onDate);
                container.appendChild(beforeDate);
                container.appendChild(afterDate);
            }

            this._c.filterList.appendChild(container);
        }
    }

    constructor(c: SearchConfiguration) {
        this._c = c;

        this._c.search.oninput = () => this.onSearchBoxChange();

        const clearSearchButtons = Array.from(document.querySelectorAll(this._c.clearSearchButtonSelector)) as Array<HTMLSpanElement>;
        for (let b of clearSearchButtons) {
            b.addEventListener("click", () => {
                this._c.search.value = "";
                this.onSearchBoxChange();
            });
        }
    }
}
