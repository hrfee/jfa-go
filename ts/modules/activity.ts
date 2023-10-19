export interface activity {
    id: string;
    type: string;
    user_id: string;
    source_type: string;
    source: string;
    invite_code: string;
    value: string;
    time: number;
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

var moodColours = ["~warning", "~neutral", "~urge"];

export class Activity { // FIXME: Add "implements"
    private _card: HTMLElement;
    private _title: HTMLElement;
    private _time: HTMLElement;
    private _sourceType: HTMLElement;
    private _source: HTMLElement;
    private _referrer: HTMLElement;
    private _act: activity;

    get type(): string { return this._act.type; }
    set type(v: string) {
        this._act.type = v;

        let mood = activityTypeMoods[v]; // 1 = positive, 0 = neutral, -1 = negative
        
        for (let i = 0; i < moodColours.length; i++) {
            if (i-1 == mood) this._card.classList.add(moodColours[i]);
            else this._card.classList.remove(moodColours[i]);
        }
    }

    get source_type(): string { return this._act.source_type; }
    set source_type(v: string) {
        this._act.source_type = v;
        if (v == "user") {
            if (this.type == "creation") {
                this._referrer.innerHTML = `<span class="supra mr-2">${window.lang.strings("referrer")}</span><a href="/accounts/${this._source}">FIXME</a>`;
            } else if (this.type == "contactLinked" || this.type == "contactUnlinked" || this.type == "changePassword" || this.type == "resetPassword") {
                // FIXME: Reflect in title
            }
        } else if (v == "admin") {
            // FIXME: Handle contactLinked/Unlinked, creation/deletion, enable/disable, createInvite/deleteInvite
        } else if (v == "anon") {
            this._referrer.innerHTML = ``;
        } else if (v == "daemon") {
            // FIXME: Handle deleteInvite, disabled, deletion
        }
    }

    constructor(act: activity) {
        this._card = document.createElement("div");

        this._card.classList.add("card", "@low");
        this._card.innerHTML = `
        <div class="flex justify-between mb-2">
            <span class="heading text-2xl activity-title"></span>
            <span class="text-sm font-medium activity-time" aria-label="${window.lang.strings("date")}"></span>
        </div>
        <div class="flex justify-between">
            <div>
                <span class="content supra mr-2 activity-source-type"></span><span class="activity-source"></span>
            </div>
            <div>
                <span class="content activity-referrer"></span>
            </div>
        </div>
        `;

        this._title = this._card.querySelector(".activity-title");
        this._time = this._card.querySelector(".activity-time");
        this._sourceType = this._card.querySelector(".activity-source-type");
        this._source = this._card.querySelector(".activity-source");
        this._referrer = this._card.querySelector(".activity-referrer");

        this.update(act);
    }

    update = (act: activity) => {
        // FIXME
        this._act = act;
        this.type  = act.type;
    }

    asElement = () => { return this._card; };
}

