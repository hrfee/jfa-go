import { _get, _post, addLoader, removeLoader, throttle } from "./common";
import { Search, SearchConfiguration } from "./search";
import "@af-utils/scrollend-polyfill";

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
        this._totalRecords = this._container.getElementsByClassName("records-total")[0] as HTMLElement;
        this._loadedRecords = this._container.getElementsByClassName("records-loaded")[0] as HTMLElement;
        this._shownRecords = this._container.getElementsByClassName("records-shown")[0] as HTMLElement;
        this._selectedRecords = this._container.getElementsByClassName("records-selected")[0] as HTMLElement;
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
    appendNewItems: (resp: paginatedDTO) => void;
    replaceWithNewItems: (resp: paginatedDTO) => void;
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
    // Infinite-scroll related data.
    // Implementation partially based on this blog post, thank you Miina Lervik:
    // https://www.bekk.christmas/post/2021/02/how-to-lazy-render-large-data-tables-to-up-performance
    protected _scroll = {
        rowHeight: 0,
        screenHeight: 0,
        // Render this many screen's worth of content below the viewport.
        renderNExtraScreensWorth: 3,
        rendered: 0,
        initialRenderCount: 0,
        scrollLoading: false,
        // Used to calculate scroll speed, so more pages are loaded when scrolling fast.
        lastScrollY: 0,
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
            this._c.loadMoreButton.disabled = true;
        } else {
            this._c.loadMoreButton.textContent = window.lang.strings("loadMore");
            this._c.loadMoreButton.disabled = false;
            this._c.loadAllButton.classList.remove("unfocused");
        }
        this.autoSetServerSearchButtonsDisabled();
    }

    protected _previousVisibleItemCount = 0;

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
       
        this._c.loadMoreButton.onclick = () => this.loadMore(false);
        this._c.loadAllButton.onclick = () => {
            addLoader(this._c.loadAllButton, true);
            this.loadMore(true);
        };
        /* this._keepSearchingButton.onclick = () => {
            addLoader(this._keepSearchingButton, true);
            this.loadMore(() => removeLoader(this._keepSearchingButton, true));
        }; */
        // Since this.reload doesn't exist, we need an arrow function to wrap it.
        this._c.refreshButton.onclick = () => this.reload();
    }

    autoSetServerSearchButtonsDisabled = () => {
        const serverSearchSortChanged = this._search.inServerSearch && (this._searchParams.sortByField != this._search.sortField || this._searchParams.ascending != this._search.ascending);
        if (this._search.inServerSearch) {
            if (serverSearchSortChanged) {
                this._search.setServerSearchButtonsDisabled(false);
            } else {
                this._search.setServerSearchButtonsDisabled(this.lastPage);
            }
            return;
        }
        if (!this._search.inSearch && this._search.sortField == this._c.defaultSortField && this._search.ascending == this._c.defaultSortAscending) {
            this._search.setServerSearchButtonsDisabled(true);
            return;
        }
        this._search.setServerSearchButtonsDisabled(false);
    }

    initSearch = (searchConfig: SearchConfiguration) => {
        const previousCallback = searchConfig.onSearchCallback;
        searchConfig.onSearchCallback = (newItems: boolean, loadAll: boolean) => {
            // if (this._search.inSearch && !this.lastPage) this._c.loadAllButton.classList.remove("unfocused");
            // else this._c.loadAllButton.classList.add("unfocused");

            this.autoSetServerSearchButtonsDisabled();

            // FIXME: Figure out why this makes sense and make it clearer.
            if ((this._visible.length < this._c.itemsPerPage && this._counter.loaded < this._c.maxItemsLoadedForSearch && !this.lastPage) || loadAll) {
                if (!newItems ||
                    this._previousVisibleItemCount != this._visible.length ||
                    (this._visible.length == 0 && !this.lastPage) ||
                    loadAll
                   ) {
                    this.loadMore(loadAll);
                }
            }
            this._previousVisibleItemCount = this._visible.length;
            if (previousCallback) previousCallback(newItems, loadAll);
        };
        const previousServerSearch = searchConfig.searchServer;
        searchConfig.searchServer = (params: PaginatedReqDTO, newSearch: boolean) => {
            this._searchParams = params;
            if (newSearch) this.reload();
            else this.loadMore(false);

            if (previousServerSearch) previousServerSearch(params, newSearch);
        };
        searchConfig.clearServerSearch = () => {
            console.trace("Clearing server search");
            this._page = 0;
            this.reload();
        }
        searchConfig.setVisibility = this.setVisibility;
        this._search = new Search(searchConfig);
        this._search.generateFilterList();
        this.lastPage = false;
    };

    // Sets the elements with "name"s in "elements" as visible or not.
    // setVisibilityNaive = (elements: string[], visible: boolean) => {
    //     let timer = this._search.timeSearches ? performance.now() : null;
    //     if (visible) this._visible = elements;
    //     else this._visible = this._search.ordering.filter(v => !elements.includes(v));
    //     const frag = document.createDocumentFragment()
    //     for (let i = 0; i < this._visible.length; i++) {
    //         frag.appendChild(this._search.items[this._visible[i]].asElement())
    //     }
    //     this._container.replaceChildren(frag);
    //     if (this._search.timeSearches) {
    //         const totalTime = performance.now() - timer;
    //         console.log(`setVisibility took ${totalTime}ms`);
    //     }
    // }

    // FIXME: Might have broken _counter.shown!
    // Sets the elements with "name"s in "elements" as visible or not.
    // appendedItems==true implies "elements" is the previously rendered elements plus some new ones on the end. Knowing this means the page's infinite scroll doesn't have to be reset.
    setVisibility = (elements: string[], visible: boolean, appendedItems: boolean = false) => {
        let timer = this._search.timeSearches ? performance.now() : null;
        if (visible) this._visible = elements;
        else this._visible = this._search.ordering.filter(v => !elements.includes(v));
        // console.log(elements.length, visible, this._visible.length);
        this._counter.shown = this._visible.length;
        if (this._visible.length == 0) {
            this._container.textContent = ``;
            return;
        }

        if (!appendedItems) {
            // Wipe old elements and render 1 new one, so we can take the element height.
            this._container.replaceChildren(this._search.items[this._visible[0]].asElement())
        }

        this._computeScrollInfo();

        // Initial render of min(_visible.length, max(rowsOnPage*renderNExtraScreensWorth, itemsPerPage)), skipping 1 as we already did it.
        this._scroll.initialRenderCount = Math.floor(Math.min(
            this._visible.length,
            Math.max(
                ((this._scroll.renderNExtraScreensWorth+1)*this._scroll.screenHeight)/this._scroll.rowHeight,
                this._c.itemsPerPage)
        ));

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
            console.debug(`setVisibility took ${totalTime}ms`);
        }
    }

    // Computes required scroll info, requiring one on-DOM item. Should be computed on page resize and this._visible change.
    _computeScrollInfo = () => {
        if (this._visible.length == 0) return;

        this._scroll.screenHeight = Math.max(
            document.documentElement.clientHeight,
            window.innerHeight || 0
        );

        this._scroll.rowHeight = this._search.items[this._visible[0]].asElement().offsetHeight;
    }

    // returns the item index to render up to for the given scroll position.
    // might return a value greater than this._visible.length, indicating a need for a page load.
    maximumItemsToRender = (scrollY: number): number => {
        const bottomScroll = scrollY + ((this._scroll.renderNExtraScreensWorth+1)*this._scroll.screenHeight);
        const bottomIdx = Math.floor(bottomScroll / this._scroll.rowHeight);
        return bottomIdx;
    }

    private _load = (
        itemLimit: number,
        page: number,
        appendFunc: (resp: paginatedDTO) => void, // Function to append/put items in storage.
        pre?: (resp: paginatedDTO) => void,
        post?: (resp: paginatedDTO) => void,
        failCallback?: (req: XMLHttpRequest) => void
    ) => {
        this._lastLoad = Date.now();
        let params = this._search.inServerSearch ? this._searchParams : this.defaultParams();
        params.limit = itemLimit;
        params.page = page;
        if (params.sortByField == "") {
            params.sortByField = this._c.defaultSortField;
            params.ascending = this._c.defaultSortAscending;
        }

        _post(this._c.getPageEndpoint, params, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
                if (failCallback) failCallback(req);
                return;
            }
            this._hasLoaded = true;

            let resp = req.response as paginatedDTO;
            if (pre) pre(resp);
            
            this.lastPage = resp.last_page;

            appendFunc(resp);
            
            this._counter.loaded = this._search.ordering.length;
            
            if (post) post(resp);

            if (this._c.pageLoadCallback) this._c.pageLoadCallback(req);
        }, true);
    }
    
    // Removes all elements, and reloads the first page.
    public abstract reload: (callback?: (resp: paginatedDTO) => void) => void;
    protected _reload = (callback?: (resp: paginatedDTO) => void) => {
        this.lastPage = false;
        this._counter.reset();
        this._counter.getTotal(this._c.totalEndpoint);
        // Reload all currently visible elements, i.e. Load a new page of size (limit*(page+1)).
        let limit = this._c.itemsPerPage;
        if (this._page != 0) {
            limit *= this._page+1;
        }
        this._load(
            limit,
            0,
            this._c.replaceWithNewItems,
            (_0: paginatedDTO) => {
                // Allow refreshes every 15s
                this._c.refreshButton.disabled = true;
                setTimeout(() => this._c.refreshButton.disabled = false, 15000);
            },
            (resp: paginatedDTO) => {
                this._search.onSearchBoxChange(true, false, false);
                if (this._search.inSearch) {
                    // this._c.loadAllButton.classList.remove("unfocused");
                } else {
                    this._counter.shown = this._counter.loaded;
                    this.setVisibility(this._search.ordering, true);
                    // this._search.showHideNotFoundPanel(false);
                }
                if (callback) callback(resp);
            },
        );
    }

    // Loads the next page. If "loadAll", all pages will be loaded until the last is reached.
    public abstract loadMore: (loadAll?: boolean, callback?: () => void) => void;
    protected _loadMore = (loadAll: boolean = false, callback?: (resp: paginatedDTO) => void) => {
        this._c.loadMoreButton.disabled = true;
        const timeout = setTimeout(() => {
            this._c.loadMoreButton.disabled = false;
        }, 1000);
        this._page += 1;

        this._load(
            this._c.itemsPerPage,
            this._page,
            this._c.appendNewItems,
            (resp: paginatedDTO) => {
                // Check before setting this.lastPage so we have a chance to cancel the timeout.
                if (resp.last_page) {
                    clearTimeout(timeout);
                    removeLoader(this._c.loadAllButton);
                }
            },
            (resp: paginatedDTO) => {
                if (this._search.inSearch || loadAll) {
                    if (this.lastPage) {
                        loadAll = false;
                    }
                    this._search.onSearchBoxChange(true, true, loadAll);
                } else {
                    // Since results come to us ordered already, we can assume "ordering"
                    // will be identical to pre-page-load but with extra elements at the end,
                    // allowing infinite scroll to continue
                    this.setVisibility(this._search.ordering, true, true);
                    this._search.setNotFoundPanelVisibility(false);
                }
                if (callback) callback(resp);
            },
        );
    }

    loadNItems = (n: number) => {
        const cb = () => {
            if (this._counter.loaded > n) return;
            this.loadMore(false, cb);
        }
        cb();
    }

    // As reloading can disrupt long-scrolling, this function will only do it if you're at the top of the page, essentially.
    public reloadIfNotInScroll = () => {
        if (this._visible.length == 0 || this.maximumItemsToRender(window.scrollY) < this._scroll.initialRenderCount) {
            return this.reload();
        }
    }


    _detectScroll = () => {
        if (!this._hasLoaded || this._scroll.scrollLoading || this._visible.length == 0) return;
        const scrollY = window.scrollY;
        const scrollSpeed = scrollY - this._scroll.lastScrollY;
        this._scroll.lastScrollY = scrollY;
        // If you've scrolled back up, do nothing
        if (scrollSpeed < 0) return;
        let endIdx = this.maximumItemsToRender(scrollY);
     
        // Throttling this function means we might not catch up in time if the user scrolls fast,
        // so we calculate the scroll speed (in rows/call) from the previous scrollY value.
        // This still might not be enough, so hackily we'll just scale it up.
        // With onscrollend, this is less necessary, but with both I wasn't able to hit the bottom of the page on my mouse.
        const rowsPerScroll = Math.round((scrollSpeed / this._scroll.rowHeight));
        // Render extra pages depending on scroll speed
        endIdx += rowsPerScroll*2;
        
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
                    // FIXME: This causes scroll-to-top when in search.
                    this.loadMore(false, cb);
                    return;
                }

                this._scroll.scrollLoading = false;
                this._detectScroll();
            };
            cb();
            return;
        }
    }

    detectScroll = throttle(this._detectScroll, 200);

    computeScrollInfo = throttle(this._computeScrollInfo, 200);

    redrawScroll = this.computeScrollInfo;

    // bindPageEvents binds window event handlers for when this list/tab containing it is visible.
    bindPageEvents = () => {
        window.addEventListener("scroll", this.detectScroll);
        // Not available on safari, we include a polyfill though.
        window.addEventListener("scrollend", this.detectScroll);
        window.addEventListener("resize", this.redrawScroll);
    };

    unbindPageEvents = () => {
        window.removeEventListener("scroll", this.detectScroll);
        window.removeEventListener("scrollend", this.detectScroll);
        window.removeEventListener("resize", this.redrawScroll);
    }
}


