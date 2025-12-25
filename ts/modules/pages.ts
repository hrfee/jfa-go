export interface PageConfig {
    hideOthersOnPageShow: boolean;
    defaultName: string;
    defaultTitle: string;
}

export class PageManager implements Pages {
    pages: Map<string, Page>;
    pageList: string[];
    hideOthers: boolean;
    defaultName: string = "";
    defaultTitle: string = "";

    private _listeners: Map<string, { params: string[]; func: (qp: URLSearchParams) => void }> = new Map();
    private _previousParams = new URLSearchParams();

    private _overridePushState = () => {
        const pushState = window.history.pushState;
        window.history.pushState = function (data: any, __: string, _: string | URL) {
            console.debug("Pushing state", arguments);
            pushState.apply(window.history, arguments);
            let ev = { state: data as string } as PopStateEvent;
            window.onpopstate(ev);
        };
    };

    private _onpopstate = (event: PopStateEvent) => {
        const prevParams = this._previousParams;
        this._previousParams = new URLSearchParams(window.location.search);
        let name = event.state;
        if (name == null) {
            // Attempt to use hash from URL, if it isn't there, try the last part of the URL.
            if (window.location.hash && window.location.hash.charAt(0) == "#") {
                name = window.location.hash.substring(1);
            } else {
                name = window.location.pathname.split("/").filter(Boolean).at(-1);
            }
        }
        if (!this.pages.has(name)) {
            name = this.pageList[0];
        }
        let success = this.pages.get(name).show();
        if (!success) {
            return;
        }
        if (this._listeners.has(name)) {
            for (let qp of this._listeners.get(name).params) {
                if (prevParams.get(qp) != this._previousParams.get(qp)) {
                    this._listeners.get(name).func(this._previousParams);
                    break;
                }
            }
        }
        if (!this.hideOthers) {
            return;
        }
        for (let k of this.pageList) {
            if (name != k) {
                this.pages.get(k).hide();
            }
        }
    };

    constructor(c: PageConfig) {
        this.pages = new Map<string, Page>();
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
        name = decodeURI(name);
        if (!this.pages.has(name)) return window.history.pushState(name || this.defaultName, this.defaultTitle, "");
        const p = this.pages.get(name);
        this.loadPage(p);
    }

    loadPage(p: Page) {
        let url = p.url;
        // Fix ordering of query params and hash
        if (url.includes("#")) {
            let split = url.split("#");
            url = split[0] + window.location.search + "#" + split[1];
        } else {
            url = url + window.location.search;
        }
        window.history.pushState(p.name || this.defaultName, p.title, url);
    }

    prev(name: string = "") {
        if (!this.pages.has(name)) return console.error(`previous page ${name} not found`);
        let p = this.pages.get(name);
        let shouldSkip = true;
        while (shouldSkip && p.index > 0) {
            p = this.pages.get(this.pageList[p.index - 1]);
            shouldSkip = p.shouldSkip();
        }
        this.loadPage(p);
    }

    next(name: string = "") {
        if (!this.pages.has(name)) return console.error(`previous page ${name} not found`);
        let p = this.pages.get(name);
        let shouldSkip = true;
        while (shouldSkip && p.index < this.pageList.length) {
            p = this.pages.get(this.pageList[p.index + 1]);
            shouldSkip = p.shouldSkip();
        }
        this.loadPage(p);
    }

    // FIXME: Make PageManager global.

    // registerParamListener allows registering a listener which will be called when one or many of the given query param names are changed. It will only be called once per navigation.
    registerParamListener(pageName: string, func: (qp: URLSearchParams) => void, ...qps: string[]) {
        const p: { params: string[]; func: (qp: URLSearchParams) => void } = this._listeners.get(pageName) || {
            params: [],
            func: null,
        };
        if (func) p.func = func;
        p.params.push(...qps);
        this._listeners.set(pageName, p);
    }
}
