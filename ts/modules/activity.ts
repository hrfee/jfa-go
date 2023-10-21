import { _post, _delete, toDateString } from "../modules/common.js";

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

export var activityReload = new CustomEvent("activity-reload");

export class Activity { // FIXME: Add "implements"
    private _card: HTMLElement;
    private _title: HTMLElement;
    private _time: HTMLElement;
    private _timeUnix: number;
    private _sourceType: HTMLElement;
    private _source: HTMLElement;
    private _referrer: HTMLElement;
    private _expiryTypeBadge: HTMLElement;
    private _delete: HTMLElement;
    private _act: activity;

    _genUserText = (): string => {
        return `<span class="font-medium">${this._act.username || this._act.user_id.substring(0, 5)}</span>`;
    }

    _genSrcUserText = (): string => {
        return `<span class="font-medium">${this._act.source_username || this._act.source.substring(0, 5)}</span>`;
    }

    _genUserLink = (): string => {
        return `<a class="hover:underline" href="/accounts/user/${this._act.user_id}">${this._genUserText()}</a>`;
    }
        
    _genSrcUserLink = (): string => {
        return `<a class="hover:underline" href="/accounts/user/${this._act.source}">${this._genSrcUserText()}</a>`;
    }

    private _renderInvText = (): string => { return `<span class="font-medium font-mono">${this.value || this.invite_code || "???"}</span>`; }

    private _genInvLink = (): string => {
        return `<a class="hover:underline" href="/accounts/invites/${this.invite_code}">${this._renderInvText()}</a>`;
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
        <div class="flex flex-col md:flex-row justify-between">
            <div>
                <span class="content supra mr-2 activity-source-type"></span><span class="activity-source"></span>
            </div>
            <div>
                <span class="content activity-referrer"></span>
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
        this._referrer = this._card.querySelector(".activity-referrer");
        this._expiryTypeBadge = this._card.querySelector(".activity-expiry-type");
        this._delete = this._card.querySelector(".activity-delete");

        document.addEventListener("timefmt-change", () => {
            this.time = this.time;
        });

        this._delete.addEventListener("click", this.delete);

        this.update(act);
    }

    update = (act: activity) => {
        // FIXME
        this._act = act;
        this.source_type = act.source_type;
        this.invite_code = act.invite_code;
        this.time = act.time;
        this.source = act.source;
        this.value = act.value;
        this.type  = act.type;
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

interface ActivitiesDTO {
    activities: activity[];
}

export class activityList {
    private _activityList: HTMLElement;

    reload = () => {
        let send = {
            "type": [],
            "limit": 60,
            "page": 0,
            "ascending": false
        }
        _post("/activity", send, (req: XMLHttpRequest) => {
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
        }, true);
    }

    constructor() {
        this._activityList = document.getElementById("activity-card-list");
        document.addEventListener("activity-reload", this.reload);
    }
}
