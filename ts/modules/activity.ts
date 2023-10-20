import { _get, toDateString } from "../modules/common.js";

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
    private _timeUnix: number;
    private _sourceType: HTMLElement;
    private _source: HTMLElement;
    private _referrer: HTMLElement;
    private _expiryTypeBadge: HTMLElement;
    private _act: activity;

    get type(): string { return this._act.type; }
    set type(v: string) {
        this._act.type = v;

        let mood = activityTypeMoods[v]; // 1 = positive, 0 = neutral, -1 = negative
        
        for (let i = 0; i < moodColours.length; i++) {
            if (i-1 == mood) this._card.classList.add(moodColours[i]);
            else this._card.classList.remove(moodColours[i]);
        }
       
        if (this.type == "changePassword" || this.type == "resetPassword") {
            let innerHTML = ``;
            if (this.type == "changePassword") innerHTML = window.lang.strings("accountChangedPassword");
            else innerHTML = window.lang.strings("accountResetPassword");
            innerHTML = innerHTML.replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
            this._title.innerHTML = innerHTML;
        } else if (this.type == "contactLinked" || this.type == "contactUnlinked") {
            let platform = this._act.type;
            if (platform == "email") {
                platform = window.lang.strings("emailAddress");
            } else {
                platform = platform.charAt(0).toUpperCase() + platform.slice(1);
            }
            let innerHTML = ``;
            if (this.type == "contactLinked") innerHTML = window.lang.strings("accountLinked");
            else innerHTML = window.lang.strings("accountUnlinked");
            innerHTML = innerHTML.replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`).replace("{contactMethod}", platform);
            this._title.innerHTML = innerHTML;
        } else if (this.type == "creation") {
            this._title.innerHTML = window.lang.strings("accountCreated").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
            if (this.source_type == "user") {
                this._referrer.innerHTML = `<span class="supra mr-2">${window.lang.strings("referrer")}</span><a href="/accounts/${this._source}">FIXME</a>`;
            } else {
                this._referrer.textContent = ``;
            }
        } else if (this.type == "deletion") {
            if (this.source_type == "daemon") {
                this._title.innerHTML = window.lang.strings("accountExpired").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
                this._expiryTypeBadge.classList.add("~critical");
                this._expiryTypeBadge.classList.remove("~warning");
                this._expiryTypeBadge.textContent = window.lang.strings("deleted");
            } else {
                this._title.innerHTML = window.lang.strings("accountDeleted").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
            }
        } else if (this.type == "enabled") {
            this._title.innerHTML = window.lang.strings("accountReEnabled").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
        } else if (this.type == "disabled") {
            if (this.source_type == "daemon") {
                this._title.innerHTML = window.lang.strings("accountExpired").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
                this._expiryTypeBadge.classList.add("~warning");
                this._expiryTypeBadge.classList.remove("~critical");
                this._expiryTypeBadge.textContent = window.lang.strings("disabled");
            } else {
                this._title.innerHTML = window.lang.strings("accountDisabled").replace("{user}", `<a href="/accounts/user/${this._act.user_id}">FIXME</a>`);
            }
        } else if (this.type == "createInvite") {
            this._title.innerHTML = window.lang.strings("inviteCreated").replace("{invite}", `<a href="/accounts/user/${this.invite_code}">${this.value || this.invite_code}</a>`);
        } else if (this.type == "deleteInvite") {

            this._title.innerHTML = window.lang.strings("inviteDeleted").replace("{invite}", this.value || this.invite_code);
        }

        /*} else if (this.source_type == "admin") {
            // FIXME: Handle contactLinked/Unlinked, creation/deletion, enable/disable, createInvite/deleteInvite
        } else if (this.source_type == "anon") {
            this._referrer.innerHTML = ``;
        } else if (this.source_type == "daemon") {
            // FIXME: Handle deleteInvite, disabled, deletion
        }*/
    }

    get time(): number { return this._timeUnix; }
    set time(v: number) {
        this._timeUnix = v;
        this._time.textContent = toDateString(new Date(v*1000));
    }

    get source_type(): string { return this._act.source_type; }
    set source_type(v: string) {
        this._act.source_type = v;
    }

    get invite_code(): string { return this._act.invite_code; }
    set invite_code(v: string) {
        this._act.invite_code = v;
    }

    get value(): string { return this._act.value; }
    set value(v: string) {
        this._act.value = v;
    }

    constructor(act: activity) {
        this._card = document.createElement("div");

        this._card.classList.add("card", "@low");
        this._card.innerHTML = `
        <div class="flex justify-between mb-2">
            <span class="heading text-2xl activity-title"></span><span class="activity-expiry-type badge"></span>
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
        this._expiryTypeBadge = this._card.querySelector(".activity-expiry-type");

        this.update(act);
    }

    update = (act: activity) => {
        // FIXME
        this._act = act;
        this.source_type = act.source_type;
        this.invite_code = act.invite_code;
        this.value = act.value;
        this.type  = act.type;
    }

    asElement = () => { return this._card; };
}

interface ActivitiesDTO {
    activities: activity[];
}

export class activityList {
    private _activityList: HTMLElement;

    reload = () => {
        _get("/activity", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                window.notifications.customError("loadActivitiesError", window.lang.notif("errorLoadActivities"));
                return;
            }

            let resp = req.response as ActivitiesDTO;
            this._activityList.textContent = ``;

            for (let act of resp.activities) {
                const activity = new Activity(act);
                this._activityList.appendChild(activity.asElement());
            }
        });
    }

    constructor() {
        this._activityList = document.getElementById("activity-card-list");
    }
}
