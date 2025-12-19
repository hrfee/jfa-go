import { PageManager, Page } from "../modules/pages.js";

export interface Tab {
    page: Page;
    tabEl: HTMLDivElement;
    buttonEl: HTMLSpanElement;
    preFunc?: () => void;
    postFunc?: () => void;
}

export class Tabs implements Tabs {
    private _current: string = "";
    private _baseOffset = -1;
    tabs: Map<string, Tab>;
    pages: PageManager;

    constructor() {
        this.tabs = new Map<string, Tab>();
        this.pages = new PageManager({
            hideOthersOnPageShow: true,
            defaultName: "invites",
            defaultTitle: document.title,
        });
    }

    addTab = (
        tabID: string,
        url: string,
        preFunc = () => void {},
        postFunc = () => void {},
        unloadFunc = () => void {},
    ) => {
        let tab: Tab = {
            page: null,
            tabEl: document.getElementById("tab-" + tabID) as HTMLDivElement,
            buttonEl: document.getElementById("button-tab-" + tabID) as HTMLButtonElement,
            preFunc: preFunc,
            postFunc: postFunc,
        };
        if (this._baseOffset == -1) {
            this._baseOffset = tab.buttonEl.offsetLeft;
        }
        const order = Array.from(this.tabs.keys());
        let scrollTo: () => number = (): number => tab.buttonEl.offsetLeft - this._baseOffset;
        if (order.length > 0) {
            scrollTo = (): number =>
                tab.buttonEl.offsetLeft - (tab.buttonEl.parentElement.offsetWidth - tab.buttonEl.offsetWidth) / 2;
        }

        tab.page = {
            name: tabID,
            title: document.title /*FIXME: Get actual names from translations*/,
            url: url,
            show: () => {
                tab.buttonEl.classList.add("active", "~urge");
                tab.tabEl.classList.remove("unfocused");
                tab.buttonEl.parentElement.scrollTo({
                    left: scrollTo(),
                    top: 0,
                    behavior: "auto",
                });
                document.dispatchEvent(new CustomEvent("tab-change", { detail: tabID }));
                return true;
            },
            hide: () => {
                tab.buttonEl.classList.remove("active");
                tab.buttonEl.classList.remove("~urge");
                tab.tabEl.classList.add("unfocused");
                if (unloadFunc) unloadFunc();
                return true;
            },
            shouldSkip: () => false,
        };
        this.pages.setPage(tab.page);
        tab.buttonEl.onclick = () => {
            this.switch(tabID);
        };
        this.tabs.set(tabID, tab);
    };

    get current(): string {
        return this._current;
    }
    set current(tabID: string) {
        this.switch(tabID);
    }

    switch = (tabID: string, noRun: boolean = false) => {
        let t = this.tabs.get(tabID);
        if (t == undefined) {
            [t] = this.tabs.values();
        }

        this._current = t.page.name;

        if (t.preFunc && !noRun) {
            t.preFunc();
        }
        this.pages.load(tabID);
        if (t.postFunc && !noRun) {
            t.postFunc();
        }
    };
}
