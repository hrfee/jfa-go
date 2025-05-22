import { _get, _post, _delete, toDateString } from "../modules/common.js";
import { SearchConfiguration, QueryType, SearchableItem, SearchableItems, SearchableItemDataAttribute } from "../modules/search.js";
import { accountURLEvent } from "../modules/accounts.js";
import { inviteURLEvent } from "../modules/invites.js";
import { PaginatedList } from "./list.js";

declare var window: GlobalWindow;

const ACTIVITY_DEFAULT_SORT_FIELD = "time";
const ACTIVITY_DEFAULT_SORT_ASCENDING = false;

export interface activity {
    id: string; 
    type: string; 
    user_id: string; 
    source_type: string; 
    source: string; 
    invite_code: string; 
    value: string; 
    time: number; 
    username: string;
    source_username: string;
    ip: string;
}

var activityTypeMoods = {
    "creation": 1,
    "deletion": -1,
    "disabled": -1,
    "enabled": 1,
    "contactLinked": 1,
    "contactUnlinked": -1,
    "changePassword": 0,
    "resetPassword": 0,
    "createInvite": 1,
    "deleteInvite": -1
};

// window.lang doesn't exist at page load, so I made this a function that's invoked by activityList.
const queries = (): { [field: string]: QueryType } => { return {
    "id": {
        name: window.lang.strings("activityID"),
        getter: "id",
        bool: false,
        string: true,
        date: false
    },
    "title": {
        name: window.lang.strings("title"),
        getter: "title",
        bool: false,
        string: true,
        date: false
    },
    "user": {
        name: window.lang.strings("usersMentioned"),
        getter: "mentionedUsers",
        bool: false,
        string: true,
        date: false
    },
    "actor": {
        name: window.lang.strings("actor"),
        description: window.lang.strings("actorDescription"),
        getter: "actor",
        bool: false,
        string: true,
        date: false
    },
    "referrer": {
        name: window.lang.strings("referrer"),
        getter: "referrer",
        bool: true,
        string: true,
        date: false
    },
    "date": {
        name: window.lang.strings("date"),
        getter: "date",
        bool: false,
        string: false,
        date: true
    },
    "account-creation": {
        name: window.lang.strings("accountCreationFilter"),
        getter: "accountCreation",
        bool: true,
        string: false,
        date: false
    },
    "account-deletion": {
        name: window.lang.strings("accountDeletionFilter"),
        getter: "accountDeletion",
        bool: true,
        string: false,
        date: false
    },
    "account-disabled": {
        name: window.lang.strings("accountDisabledFilter"),
        getter: "accountDisabled",
        bool: true,
        string: false,
        date: false
    },
    "account-enabled": {
        name: window.lang.strings("accountEnabledFilter"),
        getter: "accountEnabled",
        bool: true,
        string: false,
        date: false
    },
    "contact-linked": {
        name: window.lang.strings("contactLinkedFilter"),
        getter: "contactLinked",
        bool: true,
        string: false,
        date: false
    },
    "contact-unlinked": {
        name: window.lang.strings("contactUnlinkedFilter"),
        getter: "contactUnlinked",
        bool: true,
        string: false,
        date: false
    },
    "password-change": {
        name: window.lang.strings("passwordChangeFilter"),
        getter: "passwordChange",
        bool: true,
        string: false,
        date: false
    },
    "password-reset": {
        name: window.lang.strings("passwordResetFilter"),
        getter: "passwordReset",
        bool: true,
        string: false,
        date: false
    },
    "invite-created": {
        name: window.lang.strings("inviteCreatedFilter"),
        getter: "inviteCreated",
        bool: true,
        string: false,
        date: false
    },
    "invite-deleted": {
        name: window.lang.strings("inviteDeletedFilter"),
        getter: "inviteDeleted",
        bool: true,
        string: false,
        date: false
    }
}};

// var moodColours = ["~warning", "~neutral", "~urge"];

export var activityReload = new CustomEvent("activity-reload");

export class Activity implements activity, SearchableItem {
    private _card: HTMLElement;
    private _title: HTMLElement;
    private _time: HTMLElement;
    private _timeUnix: number;
    private _sourceType: HTMLElement;
    private _source: HTMLElement;
    private _referrer: HTMLElement;
    private _expiryTypeBadge: HTMLElement;
    private _delete: HTMLElement;
    private _ip: HTMLElement;
    private _act: activity;

    _genUserText = (): string => {
        return `<span class="font-medium">${this._act.username || this._act.user_id.substring(0, 5)}</span>`;
    }

    _genSrcUserText = (): string => {
        return `<span class="font-medium">${this._act.source_username || this._act.source.substring(0, 5)}</span>`;
    }

    _genUserLink = (): string => {
        return `<a role="link" tabindex="0" class="hover:underline cursor-pointer activity-pseudo-link-user" data-id="${this._act.user_id}" href="${window.pages.Base}${window.pages.Admin}/accounts?user=${this._act.user_id}">${this._genUserText()}</a>`;
    }
        
    _genSrcUserLink = (): string => {
        return `<a role="link" tabindex="0" class="hover:underline cursor-pointer activity-pseudo-link-user" data-id="${this._act.user_id}" href="${window.pages.Base}${window.pages.Admin}/accounts?user=${this._act.source}">${this._genSrcUserText()}</a>`;
    }

    private _renderInvText = (): string => { return `<span class="font-medium font-mono">${this.value || this.invite_code || "???"}</span>`; }

    private _genInvLink = (): string => {
        return `<a role="link" tabindex="0" class="hover:underline cursor-pointer activity-pseudo-link-invite" data-id="${this.invite_code}" href="${window.pages.Base}${window.pages.Admin}/?invite=${this.invite_code}">${this._renderInvText()}</a>`;
    }


    get accountCreation(): boolean { return this.type == "creation"; }
    get accountDeletion(): boolean { return this.type == "deletion"; }
    get accountDisabled(): boolean { return this.type == "disabled"; }
    get accountEnabled(): boolean { return this.type == "enabled"; }
    get contactLinked(): boolean { return this.type == "contactLinked"; }
    get contactUnlinked(): boolean { return this.type == "contactUnlinked"; }
    get passwordChange(): boolean { return this.type == "changePassword"; }
    get passwordReset(): boolean { return this.type == "resetPassword"; }
    get inviteCreated(): boolean { return this.type == "createInvite"; }
    get inviteDeleted(): boolean { return this.type == "deleteInvite"; }

    get mentionedUsers(): string {
        return (this.username + " " + this.source_username).toLowerCase();
    }

    get actor(): string {
        let out = this.source_type + " ";
        if (this.source_type == "admin" || this.source_type == "user") out += this.source_username;
        return out.toLowerCase();
    }

    get referrer(): string {
        if (this.type != "creation" || this.source_type != "user") return "";
        return this.source_username.toLowerCase();
    }

    get type(): string { return this._act.type; }
    set type(v: string) {
        this._act.type = v;

        let mood = activityTypeMoods[v]; // 1 = positive, 0 = neutral, -1 = negative
        for (let el of [this._card, this._delete]) {
            el.classList.remove("~warning");
            el.classList.remove("~neutral");
            el.classList.remove("~urge");
            
            if (mood == -1) {
                el.classList.add("~warning");
            } else if (mood == 0) {
                el.classList.add("~neutral");
            } else if (mood == 1) {
                el.classList.add("~urge");
            }
        }

        /* for (let i = 0; i < moodColours.length; i++) {
            if (i-1 == mood) this._card.classList.add(moodColours[i]);
            else this._card.classList.remove(moodColours[i]);
        } */
       
        if (this.type == "changePassword" || this.type == "resetPassword") {
            let innerHTML = ``;
            if (this.type == "changePassword") innerHTML = window.lang.strings("accountChangedPassword");
            else innerHTML = window.lang.strings("accountResetPassword");
            innerHTML = innerHTML.replace("{user}", this._genUserLink());
            this._title.innerHTML = innerHTML;
        } else if (this.type == "contactLinked" || this.type == "contactUnlinked") {
            let platform = this.value;
            if (platform == "email") {
                platform = window.lang.strings("emailAddress");
            } else {
                platform = platform.charAt(0).toUpperCase() + platform.slice(1);
            }
            let innerHTML = ``;
            if (this.type == "contactLinked") innerHTML = window.lang.strings("accountLinked");
            else innerHTML = window.lang.strings("accountUnlinked");
            innerHTML = innerHTML.replace("{user}", this._genUserLink()).replace("{contactMethod}", platform);
            this._title.innerHTML = innerHTML;
        } else if (this.type == "creation") {
            this._title.innerHTML = window.lang.strings("accountCreated").replace("{user}", this._genUserLink());
            if (this.source_type == "user") {
                this._referrer.innerHTML = `<span class="supra mr-2">${window.lang.strings("referrer")}</span>${this._genSrcUserLink()}`;
            } else {
                this._referrer.textContent = ``;
            }
        } else if (this.type == "deletion") {
            if (this.source_type == "daemon") {
                this._title.innerHTML = window.lang.strings("accountExpired").replace("{user}", this._genUserText());
                this._expiryTypeBadge.classList.add("~critical");
                this._expiryTypeBadge.classList.remove("~info");
                this._expiryTypeBadge.textContent = window.lang.strings("deleted");
            } else {
                this._title.innerHTML = window.lang.strings("accountDeleted").replace("{user}", this._genUserText());
            }
        } else if (this.type == "enabled") {
            this._title.innerHTML = window.lang.strings("accountReEnabled").replace("{user}", this._genUserLink());
        } else if (this.type == "disabled") {
            if (this.source_type == "daemon") {
                this._title.innerHTML = window.lang.strings("accountExpired").replace("{user}", this._genUserLink());
                this._expiryTypeBadge.classList.add("~info");
                this._expiryTypeBadge.classList.remove("~critical");
                this._expiryTypeBadge.textContent = window.lang.strings("disabled");
            } else {
                this._title.innerHTML = window.lang.strings("accountDisabled").replace("{user}", this._genUserLink());
            }
        } else if (this.type == "createInvite") {
            this._title.innerHTML = window.lang.strings("inviteCreated").replace("{invite}", this._genInvLink());
        } else if (this.type == "deleteInvite") {
            let innerHTML = ``;
            if (this.source_type == "daemon") {
                innerHTML = window.lang.strings("inviteExpired");
            } else {
                innerHTML = window.lang.strings("inviteDeleted");
            }

            this._title.innerHTML = innerHTML.replace("{invite}", this._renderInvText());
        }
    }

    get time(): number { return this._timeUnix; }
    set time(v: number) {
        this._timeUnix = v;
        this._time.textContent = toDateString(new Date(v*1000));
    }

    get source_type(): string { return this._act.source_type; }
    set source_type(v: string) {
        this._act.source_type = v;
        if ((this.source_type == "anon" || this.source_type == "user") && this.type == "creation") {
            this._sourceType.textContent = window.lang.strings("fromInvite");
        } else if (this.source_type == "admin") {
            this._sourceType.textContent = window.lang.strings("byAdmin");
        } else if (this.source_type == "user" && this.type != "creation") {
            this._sourceType.textContent = window.lang.strings("byUser");
        } else if (this.source_type == "daemon") {
            this._sourceType.textContent = window.lang.strings("byJfaGo");
        }
    }

    get ip(): string { return this._act.ip; }
    set ip(v: string) {
        this._act.ip = v;
        if (v) {
            this._ip.innerHTML = `<span class="supra mr-2">IP</span><span class="font-mono bg-inherit">${v}</span>`;
        } else {
            this._ip.textContent = ``;
        }
    }

    get invite_code(): string { return this._act.invite_code; }
    set invite_code(v: string) {
        this._act.invite_code = v;
    }

    get value(): string { return this._act.value; }
    set value(v: string) {
        this._act.value = v;
    }

    get source(): string { return this._act.source; }
    set source(v: string) {
        this._act.source = v;
        if ((this.source_type == "anon" || this.source_type == "user") && this.type == "creation") {
            this._source.innerHTML = this._genInvLink();
        } else if ((this.source_type == "admin" || this.source_type == "user") && this._act.source != "" && this._act.source_username != "") {
            this._source.innerHTML = this._genSrcUserLink();
        }
    }

    get id(): string { return this._act.id; }
    set id(v: string) {
        this._act.id = v;
        this._card.setAttribute(SearchableItemDataAttribute, v);
    }

    get user_id(): string { return this._act.user_id; }
    set user_id(v: string) { this._act.user_id = v; }

    get username(): string { return this._act.username; }
    set username(v: string) { this._act.username = v; }

    get source_username(): string { return this._act.source_username; }
    set source_username(v: string) { this._act.source_username = v; }

    get title(): string { return this._title.textContent; }

    matchesSearch = (query: string): boolean => {
        // console.log(this.title, "matches", query, ":", this.title.includes(query));
        return (
            this.title.toLowerCase().includes(query) ||
            this.username.toLowerCase().includes(query) ||
            this.source_username.toLowerCase().includes(query)
        );
    }

    constructor(act: activity) {
        this._card = document.createElement("div");

        this._card.classList.add("card", "@low", "my-2");

        this._card.innerHTML = `
        <div class="flex flex-col md:flex-row justify-between mb-2">
            <span class="heading truncate flex-initial md:text-2xl text-xl activity-title"></span>
            <div class="flex flex-col flex-none ml-0 md:ml-2">
                <span class="font-medium md:text-sm text-xs activity-time" aria-label="${window.lang.strings("date")}"></span>
                <span class="activity-expiry-type badge self-start md:self-end mt-1"></span>
            </div>
        </div>
        <div class="flex flex-row justify-between items-end">
            <div class="flex flex-col md:flex-row gap-2">
                <div>    
                    <span class="content supra mr-2 activity-source-type"></span><span class="activity-source"></span>
                </div>
                <span class="content activity-referrer"></span>
                <span class="content activity-ip"></span>
            </div>
            <div>
                <button class="button @low hover:~critical rounded-full px-1 py-px activity-delete" aria-label="${window.lang.strings("delete")}"><i class="ri-close-line"></i></button>
            </div>
        </div>
        `;

        this._title = this._card.querySelector(".activity-title");
        this._time = this._card.querySelector(".activity-time");
        this._sourceType = this._card.querySelector(".activity-source-type");
        this._source = this._card.querySelector(".activity-source");
        this._ip = this._card.querySelector(".activity-ip");
        this._referrer = this._card.querySelector(".activity-referrer");
        this._expiryTypeBadge = this._card.querySelector(".activity-expiry-type");
        this._delete = this._card.querySelector(".activity-delete");

        document.addEventListener("timefmt-change", () => {
            this.time = this.time;
        });

        this._delete.addEventListener("click", this.delete);

        this.update(act);

        const pseudoUsers = this._card.getElementsByClassName("activity-pseudo-link-user") as HTMLCollectionOf<HTMLAnchorElement>;
        const pseudoInvites = this._card.getElementsByClassName("activity-pseudo-link-invite") as HTMLCollectionOf<HTMLAnchorElement>;

        for (let i = 0; i < pseudoUsers.length; i++) {
            /*const navigate = (event: Event) => {
                event.preventDefault()
                window.tabs.switch("accounts");
                document.dispatchEvent(accountURLEvent(pseudoUsers[i].getAttribute("data-id")));
                window.history.pushState(null, document.title, pseudoUsers[i].getAttribute("data-href"));
            }
            pseudoUsers[i].onclick = navigate;
            pseudoUsers[i].onkeydown = navigate;*/
        }
        for (let i = 0; i < pseudoInvites.length; i++) {
            /*const navigate = (event: Event) => {
                event.preventDefault();
                window.invites.reload(() => {
                    window.tabs.switch("invites");
                    document.dispatchEvent(inviteURLEvent(pseudoInvites[i].getAttribute("data-id")));
                    window.history.pushState(null, document.title, pseudoInvites[i].getAttribute("data-href"));
                });
            }
            pseudoInvites[i].onclick = navigate;
            pseudoInvites[i].onkeydown = navigate;*/
        }
    }

    update = (act: activity) => {
        this._act = act;
        this.source_type = act.source_type;
        this.invite_code = act.invite_code;
        this.time = act.time;
        this.source = act.source;
        this.value = act.value;
        this.type  = act.type;
        this.ip = act.ip;
    }

    delete = () => _delete("/activity/" + this._act.id, null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        if (req.status == 200) {
            window.notifications.customSuccess("activityDeleted", window.lang.notif("activityDeleted"));
        }
        document.dispatchEvent(activityReload);
    });

    asElement = () => { return this._card; };
}

interface ActivitiesReqDTO extends PaginatedReqDTO {
    type: string[];
};

interface ActivitiesDTO extends paginatedDTO {
    activities: activity[];
}

export class activityList extends PaginatedList {
    protected _container: HTMLElement;
    // protected _sortingByButton = document.getElementById("activity-sort-by-field") as HTMLButtonElement;
    protected _sortDirection = document.getElementById("activity-sort-direction") as HTMLButtonElement;

    protected _ascending: boolean;
    
    get activities(): { [id: string]: Activity } { return this._search.items as { [id: string]: Activity }; }
    // set activities(v: { [id: string]: Activity }) { this._search.items = v as SearchableItems; }
    
    constructor() {
        super({
            loader: document.getElementById("activity-loader"),
            loadMoreButton: document.getElementById("activity-load-more") as HTMLButtonElement,
            loadAllButton: document.getElementById("activity-load-all") as HTMLButtonElement,
            refreshButton: document.getElementById("activity-refresh") as HTMLButtonElement,
            filterArea: document.getElementById("activity-filter-area"),
            searchOptionsHeader: document.getElementById("activity-search-options-header"),
            searchBox: document.getElementById("activity-search") as HTMLInputElement,
            recordCounter: document.getElementById("activity-record-counter"),
            totalEndpoint: "/activity/count",
            getPageEndpoint: "/activity",
            itemsPerPage: 20,
            maxItemsLoadedForSearch: 200,
            newElementsFromPage: (resp: paginatedDTO) => {
                let ordering: string[] = this._search.ordering;
                for (let act of ((resp as ActivitiesDTO).activities || [])) {
                    this.activities[act.id] = new Activity(act);
                    ordering.push(act.id);
                }
            },
            updateExistingElementsFromPage: (resp: paginatedDTO) => {
                // FIXME: Implement updates to existing elements!
                for (let id of Object.keys(this.activities)) {
                    delete this.activities[id];
                }
                this._search.setOrdering([], this._c.defaultSortField, this.ascending);
                this._c.newElementsFromPage(resp);
            },
            defaultSortField: ACTIVITY_DEFAULT_SORT_FIELD,
            defaultSortAscending: ACTIVITY_DEFAULT_SORT_ASCENDING,
            pageLoadCallback: (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                if (req.status != 200) {
                    window.notifications.customError("loadActivitiesError", window.lang.notif("errorLoadActivities"));
                    return;
                }
            }
        });

        this._container = document.getElementById("activity-card-list")
        document.addEventListener("activity-reload", this.reload);

        let searchConfig: SearchConfiguration = {
            filterArea: this._c.filterArea,
            // Exclude this: We only sort by date, and don't want to show a redundant header indicating so.
            // sortingByButton: this._sortingByButton,
            searchOptionsHeader: this._c.searchOptionsHeader,
            notFoundPanel: document.getElementById("activity-not-found"),
            notFoundLocallyText: document.getElementById("activity-no-local-results"),
            search: this._c.searchBox,
            clearSearchButtonSelector: ".activity-search-clear",
            serverSearchButtonSelector: ".activity-search-server",
            queries: queries(),
            setVisibility: this.setVisibility,
            filterList: document.getElementById("activity-filter-list"),
            // notFoundCallback: this._notFoundCallback,
            onSearchCallback: null,
            searchServer: null,
            clearServerSearch: null,
        }

        this.initSearch(searchConfig);

        this.ascending = this._c.defaultSortAscending;
        this._sortDirection.addEventListener("click", () => this.ascending = !this.ascending);
    }

    reload = () => {
        this._reload();
    }

    loadMore = (callback?: () => void, loadAll: boolean = false) => {
        this._loadMore(
            loadAll,
            (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                if (req.status != 200) return;
                if (callback) callback();
            }
        );
    }

    get ascending(): boolean { return this._ascending; }
    set ascending(v: boolean) {
        this._ascending = v;
        this._sortDirection.innerHTML = `${window.lang.strings("sortDirection")} <i class="ri-arrow-${v ? "up" : "down"}-s-line ml-2"></i>`;
        // FIXME?: We don't actually re-sort the list here, instead just use setOrdering to apply this.ascending before a reload.
        this._search.setOrdering(this._search.ordering, this._c.defaultSortField, this.ascending);
        if (this._hasLoaded) {
            this.reload();
        }
    }

    /*private _notFoundCallback = (notFound: boolean) => {
        if (notFound) this._loadMoreButton.classList.add("unfocused");
        else this._loadMoreButton.classList.remove("unfocused");

        if (notFound && !this._lastPage) {
            this._keepSearchingButton.classList.remove("unfocused");
            this._keepSearchingDescription.classList.remove("unfocused");
        } else {
            this._keepSearchingButton.classList.add("unfocused");
            this._keepSearchingDescription.classList.add("unfocused");
        }
    };*/

}
