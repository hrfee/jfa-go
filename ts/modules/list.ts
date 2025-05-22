import { _get, _post, addLoader, removeLoader, throttle } from "./common";
import { Search, SearchConfiguration } from "./search";

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
    filterArea: HTMLElement;
    searchOptionsHeader: HTMLElement;
    searchBox: HTMLInputElement;
    recordCounter: HTMLElement;
    totalEndpoint: string;
    getPageEndpoint: string;
    itemsPerPage: number;
    maxItemsLoadedForSearch: number;
    newElementsFromPage: (resp: paginatedDTO) => void;
    updateExistingElementsFromPage: (resp: paginatedDTO) => void;
    defaultSortField: string;
    defaultSortAscending: boolean;
    pageLoadCallback?: (req: XMLHttpRequest) => void;
}

export abstract class PaginatedList {
    protected _c: PaginatedListConfig;
   
    // Container to append items to.
    protected _container: HTMLElement;
    // List of visible IDs (i.e. those set with setVisibility).
    protected _visible: string[];
    protected _scroll = {
        rowHeight: 0,
        screenHeight: 0,
        // Render this many screen's worth of content below the viewport.
        renderNExtraScreensWorth: 3,
        rowsOnPage: 0,
        rendered: 0,
        initialRenderCount: 0,
        scrollLoading: false
    };

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
            this._search.setServerSearchButtonsDisabled(this._search.inServerSearch);
            this._c.loadMoreButton.disabled = true;
        } else {
            this._c.loadMoreButton.textContent = window.lang.strings("loadMore");
            this._c.loadMoreButton.disabled = false;
            this._search.setServerSearchButtonsDisabled(false);
            this._c.loadAllButton.classList.remove("unfocused");
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
        this._c.refreshButton.onclick = () => this.reload();
    }

    initSearch = (searchConfig: SearchConfiguration) => {
        const previousCallback = searchConfig.onSearchCallback;
        searchConfig.onSearchCallback = (visibleCount: number, newItems: boolean, loadAll: boolean) => {
            this._counter.shown = visibleCount;

            // if (this._search.inSearch && !this.lastPage) this._c.loadAllButton.classList.remove("unfocused");
            // else this._c.loadAllButton.classList.add("unfocused");
          
            if (this._search.sortField == this._c.defaultSortField && this._search.ascending == this._c.defaultSortAscending) {
                this._search.setServerSearchButtonsDisabled(!this._search.inSearch)
            } else {
                this._search.setServerSearchButtonsDisabled(false)
            }

            // FIXME: Figure out why this makes sense and make it clearer.
            if ((visibleCount < this._c.itemsPerPage && this._counter.loaded < this._c.maxItemsLoadedForSearch && !this.lastPage) || loadAll) {
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
            console.log("Clearing server search");
            this._page = 0;
            this.reload();
        }
        this._search = new Search(searchConfig);
        this._search.generateFilterList();
        this.lastPage = false;
    };

    // Sets the elements with "name"s in "elements" as visible or not.
    setVisibilityNaive = (elements: string[], visible: boolean) => {
        let timer = this._search.timeSearches ? performance.now() : null;
        if (visible) this._visible = elements;
        else this._visible = this._search.ordering.filter(v => !elements.includes(v));
        const frag = document.createDocumentFragment()
        for (let i = 0; i < this._visible.length; i++) {
            frag.appendChild(this._search.items[this._visible[i]].asElement())
        }
        this._container.replaceChildren(frag);
        if (this._search.timeSearches) {
            const totalTime = performance.now() - timer;
            console.log(`setVisibility took ${totalTime}ms`);
        }
    }
   
    // FIXME: Call on window resize/zoom
    // FIXME: On reload, load enough pages to fill required space.
    // FIXME: Might have broken _counter.shown!
    // Sets the elements with "name"s in "elements" as visible or not.
    // appendedItems==true implies "elements" is the previously rendered elements plus some new ones on the end. Knowing this means the page's infinite scroll doesn't have to be reset.
    setVisibility = (elements: string[], visible: boolean, appendedItems: boolean = false) => {
        let timer = this._search.timeSearches ? performance.now() : null;
        if (visible) this._visible = elements;
        else this._visible = this._search.ordering.filter(v => !elements.includes(v));
        if (this._visible.length == 0) return;

        this._scroll.screenHeight = Math.max(
            document.documentElement.clientHeight,
            window.innerHeight || 0
        );

        if (!appendedItems) {
            // Wipe old elements and render 1 new one, so we can take the element height.
            this._container.replaceChildren(this._search.items[this._visible[0]].asElement())
        }

        this.computeScrollInfo();

        let baseIndex = 1;
        if (appendedItems) {
            baseIndex = this._scroll.rendered;
        }
        const frag = document.createDocumentFragment()
        for (let i = baseIndex; i < this._scroll.initialRenderCount; i++) {
            frag.appendChild(this._search.items[this._visible[i]].asElement())
        }
        this._scroll.rendered = Math.max(baseIndex, this._scroll.initialRenderCount);
        // appendChild over replaceChildren because there's already elements on the DOM 
        this._container.appendChild(frag);

        if (this._search.timeSearches) {
            const totalTime = performance.now() - timer;
            console.log(`setVisibility took ${totalTime}ms`);
        }
    }

    // Computes required scroll info, requiring one on-DOM item. Should be computed on page resize and this._visible change.
    computeScrollInfo = () => {
        this._scroll.rowHeight = this._search.items[this._visible[0]].asElement().offsetHeight;

        // We want to have _scroll.renderNScreensWorth*_scroll.screenHeight or more elements rendered always.
        this._scroll.rowsOnPage = Math.floor(this._scroll.screenHeight / this._scroll.rowHeight);

        // Initial render of min(_visible.length, max(rowsOnPage*renderNExtraScreensWorth, itemsPerPage)), skipping 1 as we already did it.
        this._scroll.initialRenderCount = Math.min(this._visible.length, Math.max((this._scroll.renderNExtraScreensWorth+1)*this._scroll.rowsOnPage, this._c.itemsPerPage));
    }

    // returns the item index to render up to for the given scroll position.
    // might return a value greater than this._visible.length, indicating a need for a page load.
    maximumItemsToRender = (scrollY: number): number => {
        const bottomScroll = scrollY + ((this._scroll.renderNExtraScreensWorth+1)*this._scroll.screenHeight);
        const bottomIdx = Math.floor(bottomScroll / this._scroll.rowsOnPage);
        return bottomIdx;
    }

    // Removes all elements, and reloads the first page.
    // FIXME: Share more code between reload and loadMore, and go over the logic, it's messy.
    public abstract reload: () => void;
    protected _reload = (
        callback?: (req: XMLHttpRequest) => void
    ) => {
        this._lastLoad = Date.now();
        this.lastPage = false;

        this._counter.reset();
        this._counter.getTotal(this._c.totalEndpoint);

        // Reload all currently visible elements, i.e. Load a new page of size (limit*(page+1)).
        let limit = this._c.itemsPerPage;
        if (this._page != 0) {
            limit *= this._page+1;
        }

        let params = this._search.inServerSearch ? this._searchParams : this.defaultParams();
        params.limit = limit;
        params.page = 0;
        if (params.sortByField == "") {
            params.sortByField = this._c.defaultSortField;
            params.ascending = this._c.defaultSortAscending;
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
                // this._search.showHideNotFoundPanel(false);
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
        params.limit = this._c.itemsPerPage;
        params.page = this._page;
        if (params.sortByField == "") {
            params.sortByField = this._c.defaultSortField;
            params.ascending = this._c.defaultSortAscending;
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
                // Since results come to us ordered already, we can assume "ordering"
                // will be identical to pre-page-load but with extra elements at the end,
                // allowing infinite scroll to continue
                this.setVisibility(this._search.ordering, true, true);
                this._search.setNotFoundPanelVisibility(false);
            }
            if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
            if (callback) callback(req);
        }, true)
    }

    loadNItems = (n: number) => {
        const cb = () => {
            if (this._counter.loaded > n) return;
            this.loadMore(cb, false);
        }
        cb();
    }

    // As reloading can disrupt long-scrolling, this function will only do it if you're at the top of the page, essentially.
    public reloadIfNotInScroll = () => {
        if (this.maximumItemsToRender(window.scrollY) < this._scroll.initialRenderCount) {
            return this.reload();
        }
    }

    _detectScroll = () => {
        if (!this._hasLoaded || this._scroll.scrollLoading) return;
        if (this._visible.length == 0) return;
        const endIdx = this.maximumItemsToRender(window.scrollY);
        // If you've scrolled back up, do nothing
        if (endIdx <= this._scroll.rendered) return;
        
        const realEndIdx = Math.min(endIdx, this._visible.length);
        const frag = document.createDocumentFragment();
        for (let i = this._scroll.rendered; i < realEndIdx; i++) {
            frag.appendChild(this._search.items[this._visible[i]].asElement());
        }
        this._scroll.rendered = realEndIdx;
        this._container.appendChild(frag);
        
        if (endIdx >= this._visible.length) {
            if (this.lastPage || this._lastLoad + 500 > Date.now()) return;
            this._scroll.scrollLoading = true;
            const cb = () => {
                if (this._visible.length < endIdx && !this.lastPage) {
                    this.loadMore(cb, false)
                    return;
                }
                this._scroll.scrollLoading = false;
                this._detectScroll();
            };
            cb();
            return;
        }
    }

    // Should be assigned to window.onscroll whenever the list is in view.
    detectScroll = throttle(this._detectScroll, 200);
}


