export class Tabs implements Tabs {
    private _current: string = "";
    tabs: Array<Tab>;
   
    constructor() {
        this.tabs = [];
    }

    addTab = (tabID: string, preFunc = () => void {}, postFunc = () => void {}) => {
        let tab = {} as Tab;
        tab.tabID = tabID;
        tab.tabEl = document.getElementById("tab-" + tabID) as HTMLDivElement;
        tab.buttonEl = document.getElementById("button-tab-" + tabID) as HTMLSpanElement;
        tab.buttonEl.onclick = () => { this.switch(tabID); };
        tab.preFunc = preFunc;
        tab.postFunc = postFunc;
        this.tabs.push(tab);
    }

    get current(): string { return this._current; }
    set current(tabID: string) { this.switch(tabID); }

    switch = (tabID: string, noRun: boolean = false) => {
        this._current = tabID;
        for (let t of this.tabs) {
            if (t.tabID == tabID) {
                t.buttonEl.classList.add("active", "~urge");
                if (t.preFunc && !noRun) { t.preFunc(); }
                t.tabEl.classList.remove("unfocused");
                if (t.postFunc && !noRun) { t.postFunc(); }
                document.dispatchEvent(new CustomEvent("tab-change", { detail: tabID }));
            } else {
                t.buttonEl.classList.remove("active");
                t.buttonEl.classList.remove("~urge");
                t.tabEl.classList.add("unfocused");
            }
        }
    }
}
