import { _get, _post, _delete, toClipboard, toggleLoader } from "../modules/common.js";

export class DOMInvite implements Invite {
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
    
    private _code: string = "None";
    get code(): string { return this._code; }
    set code(code: string) {
        this._code = code;
        this._codeLink = window.location.href.split("#")[0] + "invite/" + code;
        const linkEl = this._codeArea.querySelector("a") as HTMLAnchorElement;
        linkEl.textContent = code.replace(/-/g, '-');
        linkEl.href = this._codeLink;
    }
    private _codeLink: string;

    private _expiresIn: string;
    get expiresIn(): string { return this._expiresIn }
    set expiresIn(expiry: string) {
        this._expiresIn = expiry;
        this._infoArea.querySelector("span.inv-expiry").textContent = expiry;
    }

    private _remainingUses: string = "1";
    get remainingUses(): string { return this._remainingUses; }
    set remainingUses(remaining: string) {
        this._remainingUses = remaining;
        this._middle.querySelector("strong.inv-remaining").textContent = remaining;
    }

    private _email: string = "";
    get email(): string { return this._email };
    set email(address: string) {
        this._email = address;
        const container = this._infoArea.querySelector(".tooltip") as HTMLDivElement;
        const icon = container.querySelector("i");
        const chip = container.querySelector("span.inv-email-chip");
        const tooltip = container.querySelector("span.content") as HTMLSpanElement;
        if (address == "") {
            container.classList.remove("mr-1");
            icon.classList.remove("ri-mail-line");
            icon.classList.remove("ri-mail-close-line");
            chip.classList.remove("~neutral");
            chip.classList.remove("~critical");
            chip.classList.remove("chip");
        } else {
            container.classList.add("mr-1");
            chip.classList.add("chip");
            if (address.includes("Failed to send to")) {
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

    private _usedBy: string[][];
    get usedBy(): string[][] { return this._usedBy; }
    set usedBy(uB: string[][]) {
        // ub[i][0]: username, ub[i][1]: date
        this._usedBy = uB;
        if (uB.length == 0) {
            this._right.classList.add("empty");
            this._userTable.innerHTML = `<p class="content">${window.lang.strings("inviteNoUsersCreated")}</p>`;
            return;
        }
        this._right.classList.remove("empty");
        let innerHTML = `
        <table class="table inv-table">
            <thead>
                <tr>
                    <th>${window.lang.strings("name")}</th>
                    <th>${window.lang.strings("date")}</th>
                </tr>
            </thead>
            <tbody>
        `;
        for (let user of uB) {
            innerHTML += `
                <tr>
                    <td>${user[0]}</td>
                    <td>${user[1]}</td>
                </tr>
            `;
        }
        innerHTML += `
            </tbody>
        </table>
        `;
        this._userTable.innerHTML = innerHTML;
    }

    private _created: string;
    get created(): string { return this._created; }
    set created(created: string) {
        this._created = created;
        this._middle.querySelector("strong.inv-created").textContent = created;
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

    constructor(invite: Invite) {
        // first create the invite structure, then use our setter methods to fill in the data.
        this._container = document.createElement('div') as HTMLDivElement;
        this._container.classList.add("inv");

        this._header = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._header);
        this._header.classList.add("card", "~neutral", "!normal", "inv-header", "elem-pad", "no-pad", "flex-expand", "row", "mt-half", "overflow-y");

        this._codeArea = document.createElement('div') as HTMLDivElement;
        this._header.appendChild(this._codeArea);
        this._codeArea.classList.add("inv-codearea");
        this._codeArea.innerHTML = `
        <a class="invite-link code monospace mr-1" href=""></a>
        <span class="button ~info !normal" title="${window.lang.strings("copy")}"><i class="ri-file-copy-line"></i></span>
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
        this._infoArea.classList.add("inv-infoarea");
        this._infoArea.innerHTML = `
        <div class="tooltip left">
            <span class="inv-email-chip"><i></i></span>
            <span class="content sm"></span>
        </div>
        <span class="inv-expiry mr-1"></span>
        <span class="button ~critical !normal inv-delete">${window.lang.strings("delete")}</span>
        <label>
            <i class="icon clickable ri-arrow-down-s-line not-rotated"></i>
            <input class="inv-toggle-details unfocused" type="checkbox">
        </label>
        `;
        
        (this._infoArea.querySelector(".inv-delete") as HTMLSpanElement).onclick = this.delete;

        const toggle = (this._infoArea.querySelector("input.inv-toggle-details") as HTMLInputElement);
        toggle.onchange = () => { this.expanded = !this.expanded; };

        this._details = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._details);
        this._details.classList.add("card", "~neutral", "!normal", "mt-half", "no-pad", "inv-details");
        const detailsInner = document.createElement('div') as HTMLDivElement;
        this._details.appendChild(detailsInner);
        detailsInner.classList.add("inv-row", "flex-expand", "row", "elem-pad", "align-top");

        this._left = document.createElement('div') as HTMLDivElement;
        detailsInner.appendChild(this._left);
        this._left.classList.add("inv-profilearea");
        let innerHTML = `
        <p class="supra mb-1 top">${window.lang.strings("profile")}</p>
        <div class="select ~neutral !normal inv-profileselect inline-block">
            <select>
                <option value="noProfile" selected>${window.lang.strings("inviteNoProfile")}</option>
            </select>
        </div>
        `;
        if (window.notificationsEnabled) {
            innerHTML += `
            <p class="label supra">${window.lang.strings("notifyEvent")}</p>
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
        this._left.innerHTML = innerHTML;
        (this._left.querySelector("select") as HTMLSelectElement).onchange = this.updateProfile;
        
        if (window.notificationsEnabled) {
            const notifyExpiry = this._left.querySelector("input.inv-notify-expiry") as HTMLInputElement;
            notifyExpiry.onchange = () => { this._notifyExpiry = notifyExpiry.checked; this.updateNotify(notifyExpiry); };

            const notifyCreation = this._left.querySelector("input.inv-notify-creation") as HTMLInputElement;
            notifyCreation.onchange = () => { this._notifyCreation = notifyCreation.checked; this.updateNotify(notifyCreation); };
        }

        this._middle = document.createElement('div') as HTMLDivElement;
        detailsInner.appendChild(this._middle);
        this._middle.classList.add("block");
        this._middle.innerHTML = `
        <p class="supra mb-1 top">${window.lang.strings("inviteDateCreated")} <strong class="inv-created"></strong></p>
        <p class="supra mb-1">${window.lang.strings("inviteRemainingUses")} <strong class="inv-remaining"></strong></p>
        `;

        this._right = document.createElement('div') as HTMLDivElement;
        detailsInner.appendChild(this._right);
        this._right.classList.add("card", "~neutral", "!low", "inv-created-users");
        this._right.innerHTML = `<strong class="supra table-header">${window.lang.strings("inviteUsersCreated")}</strong>`;
        this._userTable = document.createElement('div') as HTMLDivElement;
        this._right.appendChild(this._userTable);


        this.expanded = false;
        this.update(invite);

        document.addEventListener("profileLoadEvent", () => { this.loadProfiles(); }, false);
    }

    update = (invite: Invite) => {
        this.code = invite.code;
        this.created = invite.created;
        this.email = invite.email;
        this.expiresIn = invite.expiresIn;
        if (window.notificationsEnabled) {
            this.notifyCreation = invite.notifyCreation;
            this.notifyExpiry = invite.notifyExpiry;
        }
        this.profile = invite.profile;
        this.remainingUses = invite.remainingUses;
        this.usedBy = invite.usedBy;
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
    }

    get empty(): boolean { return this._empty; }
    set empty(state: boolean) {
        this._empty = state;
        if (state) {
            this.invites = {};
            this._list.classList.add("empty");
            this._list.innerHTML = `
            <div class="inv inv-empty">
                <div class="card ~neutral !normal inv-header flex-expand mt-half">
                    <div class="inv-codearea">
                        <span class="code monospace">${window.lang.strings("inviteNoInvites")}</span>
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

    reload = () => _get("/invites", null, (req: XMLHttpRequest) => {
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
        }
    })
}
    

function parseInvite(invite: { [f: string]: string | number | string[][] | boolean }): Invite {
    let parsed: Invite = {};
    parsed.code = invite["code"] as string;
    parsed.email = invite["email"] as string || "";
    let time = "";
    const fields = ["days", "hours", "minutes"];
    for (let i = 0; i < fields.length; i++) {
        if (invite[fields[i]] != 0) {
            time += `${invite[fields[i]]}${fields[i][0]} `;
        }
    }
    parsed.expiresIn = window.lang.var("strings", "inviteExpiresInTime", time.slice(0, -1));
    parsed.remainingUses = invite["no-limit"] ? "∞" : String(invite["remaining-uses"])
    parsed.usedBy = invite["used-by"] as string[][] || [];
    parsed.created = invite["created"] as string || window.lang.strings("unknown");
    parsed.profile = invite["profile"] as string || "";
    parsed.notifyExpiry = invite["notify-expiry"] as boolean || false;
    parsed.notifyCreation = invite["notify-creation"] as boolean || false;
    return parsed;
}

export class createInvite {
    private _sendToEnabled = document.getElementById("create-send-to-enabled") as HTMLInputElement;
    private _sendTo = document.getElementById("create-send-to") as HTMLInputElement;
    private _uses = document.getElementById('create-uses') as HTMLInputElement;
    private _infUses = document.getElementById("create-inf-uses") as HTMLInputElement;
    private _infUsesWarning = document.getElementById('create-inf-uses-warning') as HTMLParagraphElement;
    private _createButton = document.getElementById("create-submit") as HTMLSpanElement;
    private _profile = document.getElementById("create-profile") as HTMLSelectElement;

    private _days = document.getElementById("create-days") as HTMLSelectElement;
    private _hours = document.getElementById("create-hours") as HTMLSelectElement;
    private _minutes = document.getElementById("create-minutes") as HTMLSelectElement;

    // Broadcast when new invite created
    private _newInviteEvent = new CustomEvent("newInviteEvent");
    private _firstLoad = true;

    private _count: Number = 30;
    private _populateNumbers = () => {
        const fieldIDs = ["create-days", "create-hours", "create-minutes"];
        for (let i = 0; i < fieldIDs.length; i++) {
            const field = document.getElementById(fieldIDs[i]);
            field.textContent = '';
            for (let n = 0; n <= this._count; n++) {
               const opt = document.createElement("option") as HTMLOptionElement;
               opt.textContent = ""+n;
               opt.value = ""+n;
               field.appendChild(opt);
            }
        }
    }

    get sendToEnabled(): boolean {
        return this._sendToEnabled.checked;
    }
    set sendToEnabled(state: boolean) {
        this._sendToEnabled.checked = state;
        this._sendTo.disabled = !state;
        if (state) {
            this._sendToEnabled.parentElement.classList.remove("~neutral");
            this._sendToEnabled.parentElement.classList.add("~urge");
        } else {
            this._sendToEnabled.parentElement.classList.remove("~urge");
            this._sendToEnabled.parentElement.classList.add("~neutral");
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
        if (this.days + this.hours + this.minutes == 0) {
            this._createButton.setAttribute("disabled", "");
            this._createButton.onclick = null;
        } else {
            this._createButton.removeAttribute("disabled");
            this._createButton.onclick = this.create;
        }
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
        let send = {
            "days": this.days,
            "hours": this.hours,
            "minutes": this.minutes,
            "multiple-uses": (this.uses > 1 || this.infiniteUses),
            "no-limit": this.infiniteUses,
            "remaining-uses": this.uses,
            "email": this.sendToEnabled ? this.sendTo : "",
            "profile": this.profile
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
        this.days = 0;
        this.hours = 0;
        this.minutes = 30;
        this._infUses.onchange = () => { this.infiniteUses = this.infiniteUses; };
        this.infiniteUses = false;
        this._sendToEnabled.onchange = () => { this.sendToEnabled = this.sendToEnabled; };
        this.sendToEnabled = false;
        this._createButton.onclick = this.create;
        this.sendTo = "";
        this.uses = 1;

        this._days.onchange = this._checkDurationValidity;
        this._hours.onchange = this._checkDurationValidity;
        this._minutes.onchange = this._checkDurationValidity;
        document.addEventListener("profileLoadEvent", () => { this.loadProfiles(); }, false);

        if (!window.emailEnabled) {
            document.getElementById("create-send-to-container").classList.add("unfocused");
        }
    }
}



