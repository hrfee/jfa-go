import { _get, _post, _delete, toClipboard, toggleLoader, toDateString } from "../modules/common.js";
import { DiscordUser, newDiscordSearch } from "../modules/discord.js";

class DOMInvite implements Invite {
    updateNotify = (checkbox: HTMLInputElement) => {
        let state: { [code: string]: { [type: string]: boolean } } = {};
        let revertChanges: () => void;
        if (checkbox.classList.contains("inv-notify-expiry")) {
            revertChanges = () => { this.notifyExpiry = !this.notifyExpiry };
            state[this.code] = { "notify-expiry": this.notifyExpiry };
        } else {
            revertChanges = () => { this.notifyCreation = !this.notifyCreation };
            state[this.code] = { "notify-creation": this.notifyCreation };
        }
        _post("/invites/notify", state, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && !(req.status == 200 || req.status == 204)) {
                revertChanges();
            }
        });
    }

    delete = () => _delete("/invites", { "code": this.code }, (req: XMLHttpRequest) => {
        if (req.readyState == 4 && (req.status == 200 || req.status == 204)) {
            this.remove();
            const inviteDeletedEvent = new CustomEvent("inviteDeletedEvent", { "detail": this.code });
            document.dispatchEvent(inviteDeletedEvent);
        }
    })

    private _label: string = "";
    get label(): string { return this._label; }
    set label(label: string) {
        this._label = label;
        const linkEl = this._codeArea.querySelector("a") as HTMLAnchorElement;
        if (label == "") {
            linkEl.textContent = this.code.replace(/-/g, '-');
        } else {
            linkEl.textContent = label;
        }
    }

    private _userLabel: string = "";
    get user_label(): string { return this._userLabel; }
    set user_label(label: string) {
        this._userLabel = label;
        const labelLabel = this._middle.querySelector(".user-label-label");
        const value = this._middle.querySelector(".user-label");
        if (label) {
            labelLabel.textContent = window.lang.strings("userLabel");
            value.textContent = label;
            value.classList.remove("unfocused");
        } else {
            labelLabel.textContent = "";
            value.textContent = "";
            value.classList.add("unfocused");
        }
    }

    private _code: string = "None";
    get code(): string { return this._code; }
    set code(code: string) {
        this._code = code;
        let codeLink = window.location.href;
        for (let split of ["#", "?"]) {
            codeLink = codeLink.split(split)[0];
        }
        if (codeLink.slice(-1) != "/") { codeLink += "/"; }
        this._codeLink = codeLink + "invite/" + code;
        const linkEl = this._codeArea.querySelector("a") as HTMLAnchorElement;
        if (this.label == "") {
            linkEl.textContent = code.replace(/-/g, '-');
        }
        linkEl.href = this._codeLink;
    }
    private _codeLink: string;

    private _expiresIn: string;
    get expiresIn(): string { return this._expiresIn }
    set expiresIn(expiry: string) {
        this._expiresIn = expiry;
        this._codeArea.querySelector("span.inv-duration").textContent = expiry;
    }

    private _userExpiry: string;
    get userExpiryTime(): string { return this._userExpiry; }
    set userExpiryTime(d: string) {
        const expiry = this._middle.querySelector("span.user-expiry") as HTMLSpanElement;
        if (!d) {
            expiry.textContent = "";
        } else {
            expiry.textContent = window.lang.strings("userExpiry");
        }
        this._userExpiry = d;
        this._middle.querySelector("strong.user-expiry-time").textContent = d;
    }

    private _remainingUses: string = "1";
    get remainingUses(): string { return this._remainingUses; }
    set remainingUses(remaining: string) {
        this._remainingUses = remaining;
        this._middle.querySelector("strong.inv-remaining").textContent = remaining;
    }

    private _send_to: string = "";
    get send_to(): string { return this._send_to };
    set send_to(address: string) {
        this._send_to = address;
        const container = this._infoArea.querySelector(".tooltip") as HTMLDivElement;
        const icon = container.querySelector("i");
        const chip = container.querySelector("span.inv-email-chip");
        const tooltip = container.querySelector("span.content") as HTMLSpanElement;
        if (address == "") {
            icon.classList.remove("ri-mail-line");
            icon.classList.remove("ri-mail-close-line");
            chip.classList.remove("~neutral");
            chip.classList.remove("~critical");
            chip.classList.remove("button");
            chip.parentElement.classList.remove("h-full");
        } else {
            chip.classList.add("button");
            chip.parentElement.classList.add("h-full");
            if (address.includes("Failed")) {
                icon.classList.remove("ri-mail-line");
                icon.classList.add("ri-mail-close-line");
                chip.classList.remove("~neutral");
                chip.classList.add("~critical");
            } else {
                address = "Sent to " + address;
                icon.classList.remove("ri-mail-close-line");
                icon.classList.add("ri-mail-line");
                chip.classList.remove("~critical");
                chip.classList.add("~neutral");
            }
        }
        tooltip.textContent = address;
    }

    private _usedBy: { [name: string]: number };
    get usedBy(): { [name: string]: number } { return this._usedBy; }
    set usedBy(uB: { [name: string]: number }) {
        this._usedBy = uB;
        if (Object.keys(uB).length == 0) {
            this._right.classList.add("empty");
            this._userTable.innerHTML = `<p class="content">${window.lang.strings("inviteNoUsersCreated")}</p>`;
            return;
        }
        this._right.classList.remove("empty");
        let innerHTML = `
        <table class="table inv-table table-p-0">
            <thead>
                <tr>
                    <th>${window.lang.strings("name")}</th>
                    <th class="w-2"></th>
                    <th>${window.lang.strings("date")}</th>
                </tr>
            </thead>
            <tbody>
        `;
        for (let username in uB) {
            innerHTML += `
                <tr>
                    <td>${username}</td>
                    <td class="w-2"></td>
                    <td>${toDateString(new Date(uB[username] * 1000))}</td>
                </tr>
            `;
        }
        innerHTML += `
            </tbody>
        </table>
        `;
        this._userTable.innerHTML = innerHTML;
    }

    private _createdUnix: number;
    get created(): number { return this._createdUnix; }
    set created(unix: number) {
        this._createdUnix = unix;
        const el = this._middle.querySelector("strong.inv-created");
        if (unix == 0) {
            el.textContent = window.lang.strings("unknown");
        } else {
            el.textContent = toDateString(new Date(unix*1000));
        }
    }
    
    private _notifyExpiry: boolean = false;
    get notifyExpiry(): boolean { return this._notifyExpiry }
    set notifyExpiry(state: boolean) {
        this._notifyExpiry = state;
        (this._left.querySelector("input.inv-notify-expiry") as HTMLInputElement).checked = state;
    }

    private _notifyCreation: boolean = false;
    get notifyCreation(): boolean { return this._notifyCreation }
    set notifyCreation(state: boolean) {
        this._notifyCreation = state;
        (this._left.querySelector("input.inv-notify-creation") as HTMLInputElement).checked = state;
    }

    private _profile: string;
    get profile(): string { return this._profile; }
    set profile(profile: string) { this.loadProfiles(profile); }
    loadProfiles = (selected?: string) => {
        const select = this._left.querySelector("select") as HTMLSelectElement;
        let noProfile = false;
        if (selected === "") {
            noProfile = true; 
        } else {
            selected = selected || select.value;
        }
        let innerHTML = `<option value="noProfile" ${noProfile ? "selected" : ""}>${window.lang.strings("inviteNoProfile")}</option>`;
        for (let profile of window.availableProfiles) {
            innerHTML += `<option value="${profile}" ${((profile == selected) && !noProfile) ? "selected" : ""}>${profile}</option>`;
        }
        select.innerHTML = innerHTML;
        this._profile = selected;
    };
    updateProfile = () => {
        const select = this._left.querySelector("select") as HTMLSelectElement;
        const previous = this.profile;
        let profile = select.value;
        if (profile == "noProfile") { profile = ""; }
        _post("/invites/profile", { "invite": this.code, "profile": profile }, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (!(req.status == 200 || req.status == 204)) {
                    select.value = previous || "noProfile";
                } else {
                    this._profile = profile;
                }
            }
        });
    }

    private _container: HTMLDivElement;

    private _header: HTMLDivElement;
    private _codeArea: HTMLDivElement;
    private _infoArea: HTMLDivElement;

    private _details: HTMLDivElement;
    private _left: HTMLDivElement;
    private _middle: HTMLDivElement;
    private _right: HTMLDivElement;
    private _userTable: HTMLDivElement;

    // whether the details card is expanded.
    get expanded(): boolean {
        return this._details.classList.contains("focused");
    }
    set expanded(state: boolean) {
        const toggle = (this._infoArea.querySelector("input.inv-toggle-details") as HTMLInputElement);
        if (state) {
            this._details.classList.remove("unfocused");
            this._details.classList.add("focused");
            toggle.previousElementSibling.classList.add("rotated");
            toggle.previousElementSibling.classList.remove("not-rotated");
        } else {
            this._details.classList.add("unfocused");
            this._details.classList.remove("focused");
            toggle.previousElementSibling.classList.remove("rotated");
            toggle.previousElementSibling.classList.add("not-rotated");
        }
    }

    focus = () => this._container.scrollIntoView({ behavior: "smooth", block: "center" });

    constructor(invite: Invite) {
        // first create the invite structure, then use our setter methods to fill in the data.
        this._container = document.createElement('div') as HTMLDivElement;
        this._container.classList.add("inv", "overflow-visible");

        this._header = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._header);
        this._header.classList.add("card", "dark:~d_neutral", "@low", "inv-header", "flex", "flex-row", "justify-between", "mt-2", "overflow-visible", "gap-2");

        this._codeArea = document.createElement('div') as HTMLDivElement;
        this._header.appendChild(this._codeArea);
        this._codeArea.classList.add("flex", "flex-row", "flex-wrap", "justify-between", "w-full", "items-baseline", "gap-2", "truncate");
        this._codeArea.innerHTML = `
        <div class="flex items-baseline gap-x-4 gap-y-2 truncate">
            <a class="invite-link text-black dark:text-white font-mono bg-inherit truncate" href=""></a>
            <span class="button ~info @low" title="${window.lang.strings("copy")}"><i class="ri-file-copy-line"></i></span>
        </div>
        <span class="inv-duration"></span>
        `;
        const copyButton = this._codeArea.querySelector("span.button") as HTMLSpanElement;
        copyButton.onclick = () => { 
            toClipboard(this._codeLink);
            const icon = copyButton.children[0];
            icon.classList.remove("ri-file-copy-line");
            icon.classList.add("ri-check-line");
            copyButton.classList.remove("~info");
            copyButton.classList.add("~positive");
            setTimeout(() => {
                icon.classList.remove("ri-check-line");
                icon.classList.add("ri-file-copy-line");
                copyButton.classList.remove("~positive");
                copyButton.classList.add("~info");
            }, 800);
        };

        this._infoArea = document.createElement('div') as HTMLDivElement;
        this._header.appendChild(this._infoArea);
        this._infoArea.classList.add("inv-infoarea", "flex", "flex-row", "items-baseline", "gap-2");
        this._infoArea.innerHTML = `
        <div class="tooltip below darker" tabindex="0">
            <span class="inv-email-chip h-full"><i></i></span>
            <span class="content sm p-1"></span>
        </div>
        <span class="button ~critical @low inv-delete h-full">${window.lang.strings("delete")}</span>
        <label>
            <i class="icon px-2.5 py-2 ri-arrow-down-s-line not-rotated"></i>
            <input class="inv-toggle-details unfocused" type="checkbox">
        </label>
        `;
        
        (this._infoArea.querySelector(".inv-delete") as HTMLSpanElement).onclick = this.delete;

        const toggle = (this._infoArea.querySelector("input.inv-toggle-details") as HTMLInputElement);
        toggle.onchange = () => { this.expanded = !this.expanded; };
        const toggleDetails = (event: Event) => { 
            if (event.target == this._header || event.target == this._codeArea || event.target == this._infoArea) {
                this.expanded = !this.expanded; 
            }
        };
        this._header.onclick = toggleDetails;


        this._details = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._details);
        this._details.classList.add("card", "~neutral", "@low", "mt-2", "inv-details");
        const detailsInner = document.createElement('div') as HTMLDivElement;
        this._details.appendChild(detailsInner);
        detailsInner.classList.add("inv-row", "flex", "flex-row", "flex-wrap", "justify-between", "gap-4");

        this._left = document.createElement('div') as HTMLDivElement;
        this._left.classList.add("flex", "flex-row", "flex-wrap", "gap-4", "min-w-full", "sm:min-w-fit", "whitespace-nowrap");
        detailsInner.appendChild(this._left);
        const leftLeft = document.createElement("div") as HTMLDivElement;
        this._left.appendChild(leftLeft);
        leftLeft.classList.add("inv-profilearea", "min-w-full", "sm:min-w-fit");
        let innerHTML = `
        <p class="supra mb-2 top">${window.lang.strings("profile")}</p>
        <div class="select ~neutral @low inv-profileselect min-w-full inline-block mb-2">
            <select>
                <option value="noProfile" selected>${window.lang.strings("inviteNoProfile")}</option>
            </select>
        </div>
        `;
        if (window.notificationsEnabled) {
            innerHTML += `
            <p class="label supra mb-2">${window.lang.strings("notifyEvent")}</p>
            <label class="switch block">
                <input class="inv-notify-expiry" type="checkbox">
                <span>${window.lang.strings("notifyInviteExpiry")}</span>
            </label>
            <label class="switch block">
                <input class="inv-notify-creation" type="checkbox">
                <span>${window.lang.strings("notifyUserCreation")}</span>
            </label>
            `;
        }
        leftLeft.innerHTML = innerHTML;
        (this._left.querySelector("select") as HTMLSelectElement).onchange = this.updateProfile;
        
        if (window.notificationsEnabled) {
            const notifyExpiry = this._left.querySelector("input.inv-notify-expiry") as HTMLInputElement;
            notifyExpiry.onchange = () => { this._notifyExpiry = notifyExpiry.checked; this.updateNotify(notifyExpiry); };

            const notifyCreation = this._left.querySelector("input.inv-notify-creation") as HTMLInputElement;
            notifyCreation.onchange = () => { this._notifyCreation = notifyCreation.checked; this.updateNotify(notifyCreation); };
        }

        this._middle = document.createElement('div') as HTMLDivElement;
        this._left.appendChild(this._middle);
        this._middle.classList.add("flex", "flex-col", "justify-between");
        this._middle.innerHTML = `
        <p class="supra top">${window.lang.strings("inviteDateCreated")} <strong class="inv-created"></strong></p>
        <p class="supra">${window.lang.strings("inviteRemainingUses")} <strong class="inv-remaining"></strong></p>
        <p class="supra"><span class="user-expiry"></span> <strong class="user-expiry-time"></strong></p>
        <p class="flex items-center"><span class="user-label-label supra mr-2"></span> <span class="user-label chip ~blue unfocused"></span></p>
        `;

        this._right = document.createElement('div') as HTMLDivElement;
        detailsInner.appendChild(this._right);
        this._right.classList.add("card", "~neutral", "@low", "inv-created-users", "min-w-full", "sm:min-w-fit", "whitespace-nowrap");
        this._right.innerHTML = `<span class="supra table-header">${window.lang.strings("inviteUsersCreated")}</span>`;
        this._userTable = document.createElement('div') as HTMLDivElement;
        this._userTable.classList.add("text-sm", "mt-1", );
        this._right.appendChild(this._userTable);


        this.expanded = false;
        this.update(invite);

        document.addEventListener("profileLoadEvent", () => { this.loadProfiles(); }, false);
        document.addEventListener("timefmt-change", () => {
            this.created = this.created;
            this.usedBy = this.usedBy;
        });
    }

    update = (invite: Invite) => {
        this.code = invite.code;
        this.created = invite.created;
        this.send_to = invite.send_to;
        this.expiresIn = invite.expiresIn;
        if (window.notificationsEnabled) {
            this.notifyCreation = invite.notifyCreation;
            this.notifyExpiry = invite.notifyExpiry;
        }
        this.profile = invite.profile;
        this.remainingUses = invite.remainingUses;
        this.usedBy = invite.usedBy;
        if (invite.label) {
            this.label = invite.label;
        }
        if (invite.user_label) {
            this.user_label = invite.user_label;
        }
        this.userExpiryTime = invite.userExpiryTime || "";
    }

    asElement = (): HTMLDivElement => { return this._container; }

    remove = () => { this._container.remove(); }
}

export class inviteList implements inviteList {
    private _list: HTMLDivElement;
    private _empty: boolean;
    // since invite reload sends profiles, this event it broadcast so the createInvite object can load them.
    private _profileLoadEvent = new CustomEvent("profileLoadEvent");

    invites: { [code: string]: DOMInvite };

    focusInvite = (inviteCode: string, errorMsg: string = window.lang.notif("errorInviteNoLongerExists")) => {
        for (let code of Object.keys(this.invites)) {
            this.invites[code].expanded = code == inviteCode;
        }
        if (inviteCode in this.invites) this.invites[inviteCode].focus();
        else window.notifications.customError("inviteDoesntExistError", errorMsg);
    };

    public static readonly _inviteURLEvent = "invite-url";
    registerURLListener = () => document.addEventListener(inviteList._inviteURLEvent, (event: CustomEvent) => {
        this.focusInvite(event.detail);
    })

    isInviteURL = () => { return window.location.pathname.startsWith(window.URLBase + "/invites/"); }

    loadInviteURL = () => {
        let inviteCode = window.location.pathname.split(window.URLBase + "/invites/")[1].split("?lang")[0];
        this.focusInvite(inviteCode, window.lang.notif("errorInviteNotFound"));
    }

    constructor() {
        this._list = document.getElementById('invites') as HTMLDivElement;
        this.empty = true;
        this.invites = {};
        document.addEventListener("newInviteEvent", () => { this.reload(); }, false);
        document.addEventListener("inviteDeletedEvent", (event: CustomEvent) => {
            const code = event.detail;
            const length = Object.keys(this.invites).length - 1; // store prior as Object.keys is undefined when there are no keys
            delete this.invites[code];
            if (length == 0) {
                this.empty = true;
            }
        }, false);

        this.registerURLListener();
    }

    get empty(): boolean { return this._empty; }
    set empty(state: boolean) {
        this._empty = state;
        if (state) {
            this.invites = {};
            this._list.classList.add("empty");
            this._list.innerHTML = `
            <div class="inv inv-empty">
                <div class="card dark:~d_neutral @low inv-header mt-2">
                    <div class="justify-start">
                        <span class="text-black dark:text-white font-mono bg-inherit">${window.lang.strings("inviteNoInvites")}</span>
                    </div>
                </div>
            </div>
            `;
        } else {
            this._list.classList.remove("empty");
            if (this._list.querySelector(".inv-empty")) {
                this._list.textContent = '';
            }
        }
    }

    add = (invite: Invite) => {
        let domInv = new DOMInvite(invite);
        this.invites[invite.code] = domInv;
        if (this.empty) { this.empty = false; }
        this._list.appendChild(domInv.asElement());
    }

    reload = (callback?: () => void) => _get("/invites", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            let data = req.response;
            if (req.status == 200) {
                window.availableProfiles = data["profiles"];
                document.dispatchEvent(this._profileLoadEvent);
            }
            if (data["invites"] === undefined || data["invites"] == null || data["invites"].length == 0) {
                this.empty = true;
                return;
            }
            // get a list of all current inv codes on dom
            // every time we find a match in resp, delete from list
            // at end delete all remaining in list from dom
            let invitesOnDOM: { [code: string]: boolean } = {};
            for (let code in this.invites) { invitesOnDOM[code] = true; }
            for (let inv of (data["invites"] as Array<any>)) {
                const invite = parseInvite(inv);
                if (invite.code in this.invites) {
                    this.invites[invite.code].update(invite);
                    delete invitesOnDOM[invite.code];
                } else {
                    this.add(invite);
                }
            }
            for (let code in invitesOnDOM) {
                this.invites[code].remove();
                delete this.invites[code];
            }

            if (callback) callback();
        }
    })
}

export const inviteURLEvent = (id: string) => { return new CustomEvent(inviteList._inviteURLEvent, {"detail": id}) };

function parseInvite(invite: { [f: string]: string | number | { [name: string]: number } | boolean }): Invite {
    let parsed: Invite = {};
    parsed.code = invite["code"] as string;
    parsed.send_to = invite["send_to"] as string || "";
    parsed.label = invite["label"] as string || "";
    parsed.user_label = invite["user_label"] as string || "";
    let time = "";
    let userExpiryTime = "";
    const fields = ["months", "days", "hours", "minutes"];
    let prefixes = [""];
    if (invite["user-expiry"] as boolean) { prefixes.push("user-"); }
    for (let i = 0; i < fields.length; i++) {
        for (let j = 0; j < prefixes.length; j++) {
            if (invite[prefixes[j]+fields[i]]) {
                let abbreviation = fields[i][0];
                if (fields[i] == "months") {
                    abbreviation += fields[i][1];
                }
                let text = `${invite[prefixes[j]+fields[i]]}${abbreviation} `;
                if (prefixes[j] ==  "user-") {
                    userExpiryTime += text;
                } else {
                    time += text;
                }
            }
        }
    }
    parsed.expiresIn = window.lang.var("strings", "inviteExpiresInTime", time.slice(0, -1));
    parsed.userExpiry = invite["user-expiry"] as boolean;
    parsed.userExpiryTime = userExpiryTime.slice(0, -1);
    parsed.remainingUses = invite["no-limit"] ? "∞" : String(invite["remaining-uses"])
    parsed.usedBy = invite["used-by"] as { [name: string]: number } || {} ;
    parsed.created = invite["created"] as number || 0;
    parsed.profile = invite["profile"] as string || "";
    parsed.notifyExpiry = invite["notify-expiry"] as boolean || false;
    parsed.notifyCreation = invite["notify-creation"] as boolean || false;
    return parsed;
}

export class createInvite {
    private _sendToEnabled = document.getElementById("create-send-to-enabled") as HTMLInputElement;
    private _sendTo = document.getElementById("create-send-to") as HTMLInputElement;
    private _discordSearch: HTMLSpanElement;
    private _userExpiryToggle = document.getElementById("create-user-expiry-enabled") as HTMLInputElement;
    private _uses = document.getElementById('create-uses') as HTMLInputElement;
    private _infUses = document.getElementById("create-inf-uses") as HTMLInputElement;
    private _infUsesWarning = document.getElementById('create-inf-uses-warning') as HTMLParagraphElement;
    private _createButton = document.getElementById("create-submit") as HTMLSpanElement;
    private _profile = document.getElementById("create-profile") as HTMLSelectElement;
    private _label = document.getElementById("create-label") as HTMLInputElement;
    private _userLabel = document.getElementById("create-user-label") as HTMLInputElement;

    private _months = document.getElementById("create-months") as HTMLSelectElement;
    private _days = document.getElementById("create-days") as HTMLSelectElement;
    private _hours = document.getElementById("create-hours") as HTMLSelectElement;
    private _minutes = document.getElementById("create-minutes") as HTMLSelectElement;
    private _userMonths = document.getElementById("user-months") as HTMLSelectElement;
    private _userDays = document.getElementById("user-days") as HTMLSelectElement;
    private _userHours = document.getElementById("user-hours") as HTMLSelectElement;
    private _userMinutes = document.getElementById("user-minutes") as HTMLSelectElement;

    private _invDurationButton = document.getElementById('radio-inv-duration') as HTMLInputElement;
    private _userExpiryButton = document.getElementById('radio-user-expiry') as HTMLInputElement;
    private _invDuration = document.getElementById('inv-duration');
    private _userExpiry = document.getElementById('user-expiry');

    private _sendToDiscord: (passData: string) => void;

    // Broadcast when new invite created
    private _newInviteEvent = new CustomEvent("newInviteEvent");
    private _firstLoad = true;

    private _count: number = 30;
    private _populateNumbers = () => {
        const fieldIDs = ["months", "days", "hours", "minutes"];
        const prefixes = ["create-", "user-"];
        for (let i = 0; i < fieldIDs.length; i++) {
            for (let j = 0; j < prefixes.length; j++) { 
                const field = document.getElementById(prefixes[j] + fieldIDs[i]);
                field.textContent = '';
                for (let n = 0; n <= this._count; n++) {
                   const opt = document.createElement("option") as HTMLOptionElement;
                   opt.textContent = ""+n;
                   opt.value = ""+n;
                   field.appendChild(opt);
                }
            }
        }
    }

    get label(): string { return this._label.value; }
    set label(label: string) { this._label.value = label; }

    get user_label(): string { return this._userLabel.value; }
    set user_label(label: string) { this._userLabel.value = label; }

    get sendToEnabled(): boolean {
        return this._sendToEnabled.checked;
    }
    set sendToEnabled(state: boolean) {
        this._sendToEnabled.checked = state;
        this._sendTo.disabled = !state;
        if (state) {
            this._sendToEnabled.parentElement.classList.remove("~neutral");
            this._sendToEnabled.parentElement.classList.add("~urge");
            if (window.discordEnabled) {
                this._discordSearch.classList.remove("~neutral");
                this._discordSearch.classList.add("~urge");
                this._discordSearch.onclick = () => this._sendToDiscord("");
            }
        } else {
            this._sendToEnabled.parentElement.classList.remove("~urge");
            this._sendToEnabled.parentElement.classList.add("~neutral");
            if (window.discordEnabled) {
                this._discordSearch.classList.remove("~urge");
                this._discordSearch.classList.add("~neutral");
                this._discordSearch.onclick = null;
            }
        }
    }

    get infiniteUses(): boolean {
        return this._infUses.checked;
    }
    set infiniteUses(state: boolean) {
        this._infUses.checked = state;
        this._uses.disabled = state;
        if (state) {
            this._infUses.parentElement.classList.remove("~neutral");
            this._infUses.parentElement.classList.add("~urge");
            this._infUsesWarning.classList.remove("unfocused");
        } else {
            this._infUses.parentElement.classList.remove("~urge");
            this._infUses.parentElement.classList.add("~neutral");
            this._infUsesWarning.classList.add("unfocused");
        }
    }
    
    get uses(): number { return this._uses.valueAsNumber; }
    set uses(n: number) { this._uses.valueAsNumber = n; }

    private _checkDurationValidity = () => {
        if (this.months + this.days + this.hours + this.minutes == 0) {
            this._createButton.setAttribute("disabled", "");
            this._createButton.onclick = null;
        } else {
            this._createButton.removeAttribute("disabled");
            this._createButton.onclick = this.create;
        }
    }

    get months(): number {
        return +this._months.value;
    }
    set months(n: number) {
        this._months.value = ""+n;
        this._checkDurationValidity();
    }
    get days(): number {
        return +this._days.value;
    }
    set days(n: number) {
        this._days.value = ""+n;
        this._checkDurationValidity();
    }
    get hours(): number {
        return +this._hours.value;
    }
    set hours(n: number) {
        this._hours.value = ""+n;
        this._checkDurationValidity();
    }
    get minutes(): number {
        return +this._minutes.value;
    }
    set minutes(n: number) {
        this._minutes.value = ""+n;
        this._checkDurationValidity();
    }
    get userExpiry(): boolean {
        return this._userExpiryToggle.checked;
    }
    set userExpiry(enabled: boolean) {
        this._userExpiryToggle.checked = enabled;
        const parent = this._userExpiryToggle.parentElement;
        if (enabled) {
            parent.classList.add("~urge");
            parent.classList.remove("~neutral");
        } else {
            parent.classList.add("~neutral");
            parent.classList.remove("~urge");
        }
        this._userMonths.disabled = !enabled;
        this._userDays.disabled = !enabled;
        this._userHours.disabled = !enabled;
        this._userMinutes.disabled = !enabled;
    }
    get userMonths(): number {
        return +this._userMonths.value;
    }
    set userMonths(n: number) {
        this._userMonths.value = ""+n;
    }
    get userDays(): number {
        return +this._userDays.value;
    }
    set userDays(n: number) {
        this._userDays.value = ""+n;
    }
    get userHours(): number {
        return +this._userHours.value;
    }
    set userHours(n: number) {
        this._userHours.value = ""+n;
    }
    get userMinutes(): number {
        return +this._userMinutes.value;
    }
    set userMinutes(n: number) {
        this._userMinutes.value = ""+n;
    }

    get sendTo(): string { return this._sendTo.value; }
    set sendTo(address: string) { this._sendTo.value = address; }

    get profile(): string { 
        const val = this._profile.value;
        if (val == "noProfile") {
            return "";
        }
        return val;
    }
    set profile(p: string) {
        if (p == "") { p = "noProfile"; }
        this._profile.value = p;
    }

    loadProfiles = () => {
        let innerHTML = `<option value="noProfile">${window.lang.strings("inviteNoProfile")}</option>`;
        for (let profile of window.availableProfiles) {
            innerHTML += `<option value="${profile}">${profile}</option>`;
        }
        let selected = this.profile;
        this._profile.innerHTML = innerHTML;
        if (this._firstLoad) {
            this.profile = window.availableProfiles[0] || "";
            this._firstLoad = false;
        } else {
            this.profile = selected;
        }
    }

    create = () => {
        toggleLoader(this._createButton);
        let userExpiry = this.userExpiry;
        if (this.userMonths == 0 && this.userDays == 0 && this.userHours == 0 && this.userMinutes == 0) {
            userExpiry = false;
        }
        let send = {
            "months": this.months,
            "days": this.days,
            "hours": this.hours,
            "minutes": this.minutes,
            "user-expiry": userExpiry,
            "user-months": this.userMonths,
            "user-days": this.userDays,
            "user-hours": this.userHours,
            "user-minutes": this.userMinutes,
            "multiple-uses": (this.uses > 1 || this.infiniteUses),
            "no-limit": this.infiniteUses,
            "remaining-uses": this.uses,
            "send-to": this.sendToEnabled ? this.sendTo : "",
            "profile": this.profile,
            "label": this.label,
            "user_label": this.user_label
        };
        _post("/invites", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200 || req.status == 204) {
                    document.dispatchEvent(this._newInviteEvent);
                }
                toggleLoader(this._createButton);
            }
        });
    }

    constructor() {
        this._populateNumbers();
        this.months = 0;
        this.days = 0;
        this.hours = 0;
        this.minutes = 30;
        this._infUses.onchange = () => { this.infiniteUses = this.infiniteUses; };
        this.infiniteUses = false;
        this._sendToEnabled.onchange = () => { this.sendToEnabled = this.sendToEnabled; };
        this.userExpiry = false;
        this._userExpiryToggle.onchange = () => { this.userExpiry = this._userExpiryToggle.checked; }
        this._userMonths.disabled = true;
        this._userDays.disabled = true;
        this._userHours.disabled = true;
        this._userMinutes.disabled = true;
        this._createButton.onclick = this.create;
        this.sendTo = "";
        this.uses = 1;
        this.label = "";

        const checkDuration = () => {
            const invSpan = this._invDurationButton.nextElementSibling as HTMLSpanElement;
            const userSpan = this._userExpiryButton.nextElementSibling as HTMLSpanElement;
            if (this._invDurationButton.checked) {
                this._invDuration.classList.remove("unfocused");
                this._userExpiry.classList.add("unfocused");
                invSpan.classList.add("@high");
                invSpan.classList.remove("@low");
                userSpan.classList.add("@low");
                userSpan.classList.remove("@high");
            } else if (this._userExpiryButton.checked) {
                this._userExpiry.classList.remove("unfocused");
                this._invDuration.classList.add("unfocused");
                invSpan.classList.add("@low");
                invSpan.classList.remove("@high");
                userSpan.classList.add("@high");
                userSpan.classList.remove("@low");
            }
        };

        this._userExpiryButton.checked = false;
        this._invDurationButton.checked = true;
        this._userExpiryButton.onchange = checkDuration;
        this._invDurationButton.onchange = checkDuration;

        this._days.onchange = this._checkDurationValidity;
        this._months.onchange = this._checkDurationValidity;
        this._hours.onchange = this._checkDurationValidity;
        this._minutes.onchange = this._checkDurationValidity;
        document.addEventListener("profileLoadEvent", () => { this.loadProfiles(); }, false);

        if (!window.emailEnabled && !window.discordEnabled) {
            document.getElementById("create-send-to-container").classList.add("unfocused");
        }

        if (window.discordEnabled) {
            this._discordSearch = document.getElementById("create-send-to-search") as HTMLSpanElement;
            this._sendToDiscord = newDiscordSearch(
                window.lang.strings("findDiscordUser"),
                window.lang.strings("searchDiscordUser"),
                window.lang.strings("select"),
                (user: DiscordUser) => {
                    this.sendTo = user.name;
                    window.modals.discord.close();
                }
            );
        }
        this.sendToEnabled = false;
    }
}
