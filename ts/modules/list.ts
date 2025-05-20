import { _get, _post, addLoader, removeLoader } from "./common";
import { Search, SearchConfiguration, SearchableItems } from "./search";

declare var window: GlobalWindow;

export class RecordCounter {
    private _container: HTMLElement;
    private _totalRecords: HTMLElement;
    private _loadedRecords: HTMLElement;
    private _shownRecords: HTMLElement;
    private _selectedRecords: HTMLElement;
    private _total: number;
    private _loaded: number;
    private _shown: number;
    private _selected: number;
    constructor(container: HTMLElement) {
        this._container = container;
        this._container.innerHTML = `
            <span class="records-total"></span>
            <span class="records-loaded"></span>
            <span class="records-shown"></span>
            <span class="records-selected"></span>
            `;
        this._totalRecords = document.getElementsByClassName("records-total")[0] as HTMLElement;
        this._loadedRecords = document.getElementsByClassName("records-loaded")[0] as HTMLElement;
        this._shownRecords = document.getElementsByClassName("records-shown")[0] as HTMLElement;
        this._selectedRecords = document.getElementsByClassName("records-selected")[0] as HTMLElement;
        this.total = 0;
        this.loaded = 0;
        this.shown = 0;
    }

    reset() {
        this.total = 0;
        this.loaded = 0;
        this.shown = 0;
        this.selected = 0;
    }

    // Sets the total using a PageCountDTO-returning API endpoint.
    getTotal(endpoint: string) {
        _get(endpoint, null, (req: XMLHttpRequest) => {
            if (req.readyState != 4 || req.status != 200) return;
            this.total = req.response["count"] as number;
        });
    }

    get total(): number { return this._total; }
    set total(v: number) {
        this._total = v;
        this._totalRecords.textContent = window.lang.var("strings", "totalRecords", `${v}`);
    }
    
    get loaded(): number { return this._loaded; }
    set loaded(v: number) {
        this._loaded = v;
        this._loadedRecords.textContent = window.lang.var("strings", "loadedRecords", `${v}`);
    }
    
    get shown(): number { return this._shown; }
    set shown(v: number) {
        this._shown = v;
        this._shownRecords.textContent = window.lang.var("strings", "shownRecords", `${v}`);
    }
    
    get selected(): number { return this._selected; }
    set selected(v: number) {
        this._selected = v;
        if (v == 0) this._selectedRecords.textContent = ``;
        else this._selectedRecords.textContent = window.lang.var("strings", "selectedRecords", `${v}`);
    }
}

export interface PaginatedListConfig {
    loader: HTMLElement;
    loadMoreButton: HTMLButtonElement;
    loadAllButton: HTMLButtonElement;
    refreshButton: HTMLButtonElement;
    keepSearchingDescription: HTMLElement;
    keepSearchingButton: HTMLElement;
    notFoundPanel: HTMLElement;
    filterArea: HTMLElement;
    searchOptionsHeader: HTMLElement;
    searchBox: HTMLInputElement;
    recordCounter: HTMLElement;
    totalEndpoint: string;
    getPageEndpoint: string;
    limit: number;
    newElementsFromPage: (resp: paginatedDTO) => void;
    updateExistingElementsFromPage: (resp: paginatedDTO) => void;
    defaultSortField: string;
    pageLoadCallback?: (req: XMLHttpRequest) => void;
}

export abstract class PaginatedList {
    protected _c: PaginatedListConfig;
    
    protected _search: Search;
    
    protected _counter: RecordCounter;
    
    protected _hasLoaded: boolean;
    protected _lastLoad: number;
    protected _page: number = 0;
    protected _lastPage: boolean;
    get lastPage(): boolean { return this._lastPage };
    set lastPage(v: boolean) {
        this._lastPage = v;
        if (v) {
            this._c.loadAllButton.classList.add("unfocused");
            this._c.loadMoreButton.textContent = window.lang.strings("noMoreResults");
            this._c.loadMoreButton.disabled = true;
        } else {
            this._c.loadMoreButton.textContent = window.lang.strings("loadMore");
            this._c.loadMoreButton.disabled = false;
            if (this._search.inSearch) {
                this._c.loadAllButton.classList.remove("unfocused");
            }
        }
    }

    protected _previousPageSize = 0;

    // Stores a PaginatedReqDTO-implementing thing.
    // A standard PaginatedReqDTO will be overridden entirely,
    // but a ServerSearchDTO will keep it's fields.
    protected _searchParams: PaginatedReqDTO;
    defaultParams = (): PaginatedReqDTO => {
        return {
            limit: 0,
            page: 0,
            sortByField: "",
            ascending: false
        };
    }

    constructor(c: PaginatedListConfig) {
        this._c = c;
        this._counter = new RecordCounter(this._c.recordCounter);
        this._hasLoaded = false;
       
        this._c.loadMoreButton.onclick = () => this.loadMore(null, false);
        this._c.loadAllButton.onclick = () => {
            addLoader(this._c.loadAllButton, true);
            this.loadMore(null, true);
        };
        /* this._keepSearchingButton.onclick = () => {
            addLoader(this._keepSearchingButton, true);
            this.loadMore(() => removeLoader(this._keepSearchingButton, true));
        }; */
        // Since this.reload doesn't exist, we need an arrow function to wrap it.
        // FIXME: Make sure it works though!
        this._c.refreshButton.onclick = () => this.reload();
    }

    initSearch = (searchConfig: SearchConfiguration) => {
        const previousCallback = searchConfig.onSearchCallback;
        searchConfig.onSearchCallback = (visibleCount: number, newItems: boolean, loadAll: boolean) => {
            this._counter.shown = visibleCount;

            // if (this._search.inSearch && !this.lastPage) this._c.loadAllButton.classList.remove("unfocused");
            // else this._c.loadAllButton.classList.add("unfocused");
           
            // FIXME: Figure out why this makes sense and make it clearer.
            if ((visibleCount < this._c.limit && !this.lastPage) || loadAll) {
                if (!newItems ||
                    this._previousPageSize != visibleCount ||
                    (visibleCount == 0 && !this.lastPage) ||
                    loadAll
                   ) {
                    this.loadMore(() => {}, loadAll);
                }
            }
            this._previousPageSize = visibleCount;
            if (previousCallback) previousCallback(visibleCount, newItems, loadAll);
        };
        const previousServerSearch = searchConfig.searchServer;
        searchConfig.searchServer = (params: PaginatedReqDTO, newSearch: boolean) => {
            this._searchParams = params;
            if (newSearch) this.reload();
            else this.loadMore(null, false);

            if (previousServerSearch) previousServerSearch(params, newSearch);
        };
        searchConfig.clearServerSearch = () => {
            this._page = 0;
            this.reload();
        }
        this._search = new Search(searchConfig);
        this._search.generateFilterList();
        this.lastPage = false;
    };

    // Sets the elements with "name"s in "elements" as visible or not.
    public abstract setVisibility: (elements: string[], visible: boolean) => void;

    // Removes all elements, and reloads the first page.
    public abstract reload: () => void;
    protected _reload = (
        callback?: (req: XMLHttpRequest) => void
    ) => {
        this._lastLoad = Date.now();
        this.lastPage = false;

        this._counter.reset();
        this._counter.getTotal(this._c.totalEndpoint);

        // Reload all currently visible elements, i.e. Load a new page of size (limit*(page+1)).
        let limit = this._c.limit;
        if (this._page != 0) {
            limit *= this._page+1;
        }

        let params = this._search.inServerSearch ? this._searchParams : this.defaultParams();
        params.limit = limit;
        params.page = 0;
        if (params.sortByField == "") {
            params.sortByField = this._c.defaultSortField;
            params.ascending = true;
        }

        _post(this._c.getPageEndpoint, params, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
                if (callback) callback(req);
                return;
            }

            this._hasLoaded = true;
            // Allow refreshes every 15s
            this._c.refreshButton.disabled = true;
            setTimeout(() => this._c.refreshButton.disabled = false, 15000);

            let resp = req.response as paginatedDTO;
            
            this.lastPage = resp.last_page;
            
            this._c.updateExistingElementsFromPage(resp);
            
            this._counter.loaded = this._search.ordering.length;
            
            this._search.onSearchBoxChange(true);
            if (this._search.inSearch) {
                // this._c.loadAllButton.classList.remove("unfocused");
            } else {
                this._counter.shown = this._counter.loaded;
                this.setVisibility(this._search.ordering, true);
                this._c.loadAllButton.classList.add("unfocused");
                this._c.notFoundPanel.classList.add("unfocused");
            }
            if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
            if (callback) callback(req);
        }, true);
    }

    // Loads the next page. If "loadAll", all pages will be loaded until the last is reached.
    public abstract loadMore: (callback: () => void, loadAll: boolean) => void;
    protected _loadMore = (
        loadAll: boolean = false,
        callback?: (req: XMLHttpRequest) => void
    ) => {
        this._lastLoad = Date.now();
        this._c.loadMoreButton.disabled = true;
        const timeout = setTimeout(() => {
            this._c.loadMoreButton.disabled = false;
        }, 1000);
        this._page += 1;

        let params = this._search.inServerSearch ? this._searchParams : this.defaultParams();
        params.limit = this._c.limit;
        params.page = this._page;
        if (params.sortByField == "") {
            params.sortByField = this._c.defaultSortField;
            params.ascending = true;
        }

        _post(this._c.getPageEndpoint, params, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
                if (callback) callback(req);
                return;
            }

            let resp = req.response as paginatedDTO;
           
            // Check before setting this.lastPage so we have a chance to cancel the timeout.
            if (resp.last_page) {
                clearTimeout(timeout);
                removeLoader(this._c.loadAllButton);
            }

            this.lastPage = resp.last_page;

            this._c.newElementsFromPage(resp);
           
            this._counter.loaded = this._search.ordering.length;
            
            if (this._search.inSearch || loadAll) {
                if (this.lastPage) {
                    loadAll = false;
                }
                this._search.onSearchBoxChange(true, loadAll);
            } else {
                this.setVisibility(this._search.ordering, true);
                this._c.notFoundPanel.classList.add("unfocused");
            }
            if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
            if (callback) callback(req);
        }, true)
    }
   
    // Should be assigned to window.onscroll whenever the list is in view.
    detectScroll = () => {
        if (!this._hasLoaded || this.lastPage) return;
        // console.log(window.innerHeight + document.documentElement.scrollTop, document.scrollingElement.scrollHeight);
        if (Math.abs(window.innerHeight + document.documentElement.scrollTop - document.scrollingElement.scrollHeight) < 50) {
            // window.notifications.customSuccess("scroll", "Reached bottom.");
            // Wait .5s between loads
            if (this._lastLoad + 500 > Date.now()) return;
            this.loadMore(null, false);
        }
    }

}


