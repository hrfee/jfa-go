import { ActivityQueries } from "./activity";
import { HTTPMethod, _http } from "./common";
import { PageCountDTO } from "./list";
import { Search, ServerFilterReqDTO } from "./search";

// FIXME: These should be language keys, not literals!
export type Unit = string | [string, string]; // string or [singular, plural]
export const UnitUsers: Unit = ["User", "Users"];
export const UnitRecords: Unit = ["Record", "Records"];
export const UnitOccurences: Unit = ["Occurrence", "Occurrences"];
export const UnitInvites: Unit = ["Invite", "Invites"];

export const EventNamespace = "stats-";
export const GlobalReload = EventNamespace + "reload-all";

export interface StatCard {
    name: string;
    value: any;
    reload: () => void;
    asElement: () => HTMLElement;
    // Event the card will listen to for reloads
    eventName: string;
};

export abstract class NumberCard implements StatCard {
    protected _name: string;
    protected _nameEl: HTMLElement;
    get name(): string { return this._name; }
    set name(v: string) {
        this._name = v;
        this._nameEl.textContent = v;
    }
    
    protected _unit: Unit | null = null;
    protected _unitEl: HTMLElement;
    get unit(): Unit { return this._unit; }
    set unit(v: Unit) {
        this._unit = v;
        // re-load value to set units correctly
        this.value = this.value;
    }

    protected _value: number;
    protected _valueEl: HTMLElement;
    get value(): number { return this._value; }
    set value(v: number) {
        this._value = v;
        if (v == -1) {
            this._valueEl.textContent = "";
        } else {
            this._valueEl.textContent = ""+this._value;
        }

        if (this._unit === null) return;

        if (typeof this._unit === "string") this._unitEl.textContent = this._unit;
        else this._unitEl.textContent = (v == 1) ? this._unit[0] : this._unit[1];
    }

    protected _url: string;
    protected _method: HTTPMethod;
    protected _container: HTMLElement;

    // generates data to be passed in the HTTP request.
    abstract params: () => any;
    // returns value from HTTP response.
    abstract handler: (req: XMLHttpRequest) => number;

    // Name of a custom event that will trigger this specific card to reload.
    readonly eventName: string;

    public reload = () => {
        let params = this.params();
        _http(this._method, this._url, params, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                this.value = -1;
            } else {
                this.value = this.handler(req);
            }
        });
    }

    protected onReloadRequest = () => this.reload();

    asElement = () => { return this._container; }

    constructor(name: string, url: string, method: HTTPMethod, unit?: Unit) {
        this._url = url;
        this._method = method;
        
        this._container = document.createElement("div");
        this._container.classList.add("card", "@low", "dark:~d_neutral", "flex", "flex-col", "gap-4");
        this._container.innerHTML = `
        <p class="text-xl italic number-card-name"></p>
        <p class="text-2xl font-bold"><span class="number-card-value"></span> <span class="number-card-unit"></span></p>
        `;
        this._nameEl = this._container.querySelector(".number-card-name");
        this._valueEl = this._container.querySelector(".number-card-value");
        this._unitEl = this._container.querySelector(".number-card-unit");

        this.name = name;
        if (unit) this.unit = unit;
        this.value = -1;

        this.eventName = this.name;
        
        document.addEventListener(EventNamespace+this.eventName, this.onReloadRequest);
        document.addEventListener(GlobalReload, this.onReloadRequest);
    }
}

export class CountCard extends NumberCard {
    params = () => {};
    handler = (req: XMLHttpRequest): number => {
        return (req.response as PageCountDTO).count;
    };

    constructor(name: string, url: string, unit?: Unit) {
        super(name, url, "GET", unit);
    }
}

export class FilteredCountCard extends CountCard {
    private _params: ServerFilterReqDTO;
    params = (): ServerFilterReqDTO => { return this._params };
    
    constructor(name: string, url: string, params: ServerFilterReqDTO, unit?: Unit) {
        super(name, url, unit);
        this._method = "POST";
        this._params = params;
    }
}

// FIXME: Make a page and load some of these!

export class StatsPanel {
    private _container: HTMLElement;
    private _cards: Map<string, StatCard>;
    private _order: string[];

    private _loaded = false;

    public static DefaultLayout(): StatCard[] {
        return [
            new CountCard("Number of users", "/users/count", UnitUsers),
            new CountCard("Number of invites", "/invites/count", UnitInvites),
            new FilteredCountCard(
                "Users created through jfa-go",
                "/activity/count",
                Search.DTOFromString("account-creation:true", ActivityQueries()),
                UnitUsers
            )
        ];
    }

    addCards = (...cards: StatCard[]) => {
        for (const card of cards) {
            this._cards.set(card.name, card);
            this._order.push(card.name);
        }
    }

    deleteCards = (...cards: StatCard[]) => {
        for (const card of cards) {
            this._cards.delete(card.name);
            let idx = this._order.indexOf(card.name);
            if (idx != -1) this._order.splice(idx, 1);
        }
    };

    reflow = () => {
        const temp = document.createDocumentFragment();
        for (const name of this._order) {
            temp.appendChild(this._cards.get(name).asElement());
        }
        this._container.replaceChildren(temp);
    }

    reload = () => {
        const hasLoaded = this._loaded;
        this._loaded = true;
        document.dispatchEvent(new CustomEvent(GlobalReload));
        if (!hasLoaded) {
            this.reflow();
        }
    }

    bindPageEvents = () => {};
    unbindPageEvents = () => {};

    constructor(container: HTMLElement) {
        this._container = container;
        this._container.classList.add("flex", "flex-row", "gap-2", "flex-wrap");

        this._cards = new Map<string, StatCard>();
        this._order = [];
    }
}
