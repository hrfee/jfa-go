export interface Page {
    name: string;
    title: string;
    url: string;
    show: () => boolean;
    hide: () => boolean;
    shouldSkip: () => boolean;
    index?: number;
};

export interface PageConfig {
    hideOthersOnPageShow: boolean;
    defaultName: string;
    defaultTitle: string;
}

export class PageManager {
    pages: Map<string, Page>;
    pageList: string[];
    hideOthers: boolean;
    defaultName: string = "";
    defaultTitle: string = "";

    private _overridePushState = () => {
        const pushState = window.history.pushState;
        window.history.pushState = function (data: any, __: string, _: string | URL) {
            pushState.apply(window.history, arguments);
            let ev = { state: data as string } as PopStateEvent;
            window.onpopstate(ev);
        };
    }

    private _onpopstate = (event: PopStateEvent) => {
        let name = event.state;
        if (!this.pages.has(event.state)) {
            name = this.pageList[0]
        }
        let success = this.pages.get(name).show();
        if (!success) {
            return;
        }
        if (!(this.hideOthers)) {
            return;
        }
        for (let k of this.pageList) {
            if (name != k) {
                this.pages.get(k).hide();
            }
        }
    }

    constructor(c: PageConfig) {
        this.pages = new Map<string, Page>;
        this.pageList = [];
        this.hideOthers = c.hideOthersOnPageShow;
        this.defaultName = c.defaultName;
        this.defaultTitle = c.defaultTitle;
    
        this._overridePushState();
        window.onpopstate = this._onpopstate;
    }

    setPage(p: Page) {
        p.index = this.pageList.length;
        this.pages.set(p.name, p);
        this.pageList.push(p.name);
    }

    load(name: string = "") {
        if (!this.pages.has(name)) return window.history.pushState(name || this.defaultName, this.defaultTitle, "")
        const p = this.pages.get(name);
        this.loadPage(p);
    }

    loadPage (p: Page) {
        window.history.pushState(p.name || this.defaultName, p.title, p.url + window.location.search);
    }

    prev(name: string = "") {
        if (!this.pages.has(name)) return console.error(`previous page ${name} not found`);
        let p = this.pages.get(name);
        let shouldSkip = true;
        while (shouldSkip && p.index > 0) {
            p = this.pages.get(this.pageList[p.index-1]);
            shouldSkip = p.shouldSkip();
        }
        this.loadPage(p);
    } 
    
    next(name: string = "") {
        if (!this.pages.has(name)) return console.error(`previous page ${name} not found`);
        let p = this.pages.get(name);
        let shouldSkip = true;
        while (shouldSkip && p.index < this.pageList.length) {
            p = this.pages.get(this.pageList[p.index+1]);
            shouldSkip = p.shouldSkip();
        }
        this.loadPage(p);
    } 
};
