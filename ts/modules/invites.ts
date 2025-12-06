import { _get, _post, _delete, _patch, toClipboard, toggleLoader, toDateString, SetupCopyButton, addLoader, removeLoader, DateCountdown } from "../modules/common.js";
import { DiscordSearch, DiscordUser, newDiscordSearch } from "../modules/discord.js";
import { reloadProfileNames }  from "../modules/profiles.js";
import { HiddenInputField } from "./ui.js";

declare var window: GlobalWindow;

const INF = "∞";

export const generateCodeLink = (code: string): string => {
    // let codeLink = window.pages.Base + window.pages.Form + "/" + code;
    let codeLink = window.pages.ExternalURI + window.pages.Form + "/" + code;
    return codeLink;
}

class DOMInvite implements Invite {
    updateNotify = (checkbox: HTMLInputElement) => {
        let state = {
            code: this.code,
            notify_expiry: this.notify_expiry,
            notify_creation: this.notify_creation
        };
        let revertChanges: () => void;
        if (checkbox.classList.contains("inv-notify-expiry")) {
            revertChanges = () => { this.notify_expiry = !this.notify_expiry };
        } else {
            revertChanges = () => { this.notify_creation = !this.notify_creation };
        }
        _patch("/invites/edit", state, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && !(req.status == 200 || req.status == 204)) {
                revertChanges();
            }
        });
    }

    delete = () => _delete("/invites", { "code": this.code }, (req: XMLHttpRequest) => {
        if (req.readyState == 4 && (req.status == 200 || req.status == 204)) {
            this.remove();
            const inviteDeletedEvent = new CustomEvent("inviteDeletedEvent", { detail: this.code });
            document.dispatchEvent(inviteDeletedEvent);
        }
    })

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

    private _label: string = "";
    get label(): string { return this._label; }
    set label(label: string) {
        this._label = label;
        if (label == "") {
            this.code = this.code;
        } else {
            this._labelEditor.value = label;
        }
    }

    private _code: string = "None";
    get code(): string { return this._code; }
    set code(code: string) {
        this._code = code;
        this._codeLink = generateCodeLink(code);
        if (this.label == "") {
            this._labelEditor.value = code.replace(/-/g, '-');
        }
        this._linkEl.href = this._codeLink;
    }
    private _codeLink: string;

    private _validTill: number;
    private _validTillUpdater: ReturnType<typeof setTimeout> = null;
    get valid_till(): number { return this._validTill; }
    set valid_till(v: number) {
        this._validTill = v;
        if (this._validTillUpdater) clearTimeout(this._validTillUpdater);
        this._validTillUpdater = DateCountdown(this._codeArea.querySelector("span.inv-duration"), v);
    }

    private _userExpiryEnabled: boolean;
    get user_expiry(): boolean { return this._userExpiryEnabled; }
    set user_expiry(v: boolean) { this._userExpiryEnabled = v; }
    private _userExpiry = { months: 0, days: 0, hours: 0, minutes: 0 };
    private _userExpiryString: string;
    get user_months(): number { return this._userExpiry.months; }
    get user_days(): number { return this._userExpiry.days; }
    get user_hours(): number { return this._userExpiry.hours; }
    get user_minutes(): number { return this._userExpiry.minutes; }
    set user_months(v: number) {
        this._userExpiry.months = v;
        this._updateUserExpiry();
    }
    set user_days(v: number) {
        this._userExpiry.days = v;
        this._updateUserExpiry();
    }
    set user_hours(v: number) {
        this._userExpiry.hours = v;
        this._updateUserExpiry();
    }
    set user_minutes(v: number) {
        this._userExpiry.minutes = v;
        this._updateUserExpiry();
    }
    set user_expiry_time(v: { months: number, days: number, hours: number, minutes: number }) {
        this._userExpiry = v;
        this._updateUserExpiry()
    }
    private _updateUserExpiry() {
        const expiry = this._middle.querySelector("span.user-expiry") as HTMLSpanElement;
        this._userExpiryString = "";
        if (!(this._userExpiry.months || this._userExpiry.days || this._userExpiry.hours || this._userExpiry.minutes)) {
            expiry.textContent = "";
            expiry.parentElement.classList.add("unfocused");
        } else {
            expiry.textContent = window.lang.strings("userExpiry");
            expiry.parentElement.classList.remove("unfocused");
            const fields = ["months", "days", "hours", "minutes"].map((v) => this._userExpiry[v]);
            const abbrevs = ["mo", "d", "h", "m"];
            for (let i = 0; i < fields.length; i++) {
                if (fields[i]) {
                    this._userExpiryString += ""+fields[i] + abbrevs[i] + " ";
                }
            }
            this._userExpiryString = this._userExpiryString.slice(0, -1);
        }
        this._middle.querySelector("strong.user-expiry-time").textContent = this._userExpiryString;
    }
    
    private _noLimit: boolean = false;
    get no_limit(): boolean { return this._noLimit; }
    set no_limit(v: boolean) {
        this._noLimit = v;
        const remaining = this._middle.querySelector("strong.inv-remaining") as HTMLElement;
        if (!this.no_limit) remaining.textContent = ""+this._remainingUses;
        else remaining.textContent = INF;
    }

    private _remainingUses: number = 1;
    get remaining_uses(): number { return this._remainingUses; }
    set remaining_uses(v: number) {
        this._remainingUses = v;
        const remaining = this._middle.querySelector("strong.inv-remaining") as HTMLElement;
        if (!this.no_limit) remaining.textContent = ""+this._remainingUses;
        else remaining.textContent = INF;
    }

    private _send_to: string = "";
    get send_to(): string { return this._send_to };
    set send_to(address: string | null) {
        this._send_to = address;
        const container = this._infoArea.querySelector(".tooltip") as HTMLDivElement;
        const icon = container.querySelector("i");
        const chip = container.querySelector("span.inv-email-chip");
        const tooltip = container.querySelector("span.content") as HTMLSpanElement;
        if (!address) {
            icon.classList.remove("ri-mail-line");
            icon.classList.remove("ri-mail-close-line");
            chip.classList.remove("~neutral");
            chip.classList.remove("~critical");
            chip.classList.remove("button");
            chip.parentElement.classList.remove("h-full");
        } else {
            chip.classList.add("button");
            chip.parentElement.classList.add("h-full");
            if (address.includes(window.lang.strings("failed"))) {
                icon.classList.remove("ri-mail-line");
                icon.classList.add("ri-mail-close-line");
                chip.classList.remove("~neutral");
                chip.classList.add("~critical");
            } else {
                icon.classList.remove("ri-mail-close-line");
                icon.classList.add("ri-mail-line");
                chip.classList.remove("~critical");
                chip.classList.add("~neutral");
            }
        }
        // innerHTML as the newer sent_to re-uses this with HTML.
        tooltip.innerHTML = address;
    }
    private _sendToDialog: SendToDialog; 
    private _sent_to: SentToList;
    get sent_to(): SentToList { return this._sent_to; }
    set sent_to(v: SentToList) {
        this._sent_to = v;
        if (!v || !(v.success || v.failed)) return;
        let text = "";
        if (v.success && v.success.length > 0) {
            text += window.lang.strings("sentTo") + ": " + v.success.join(", ") + " <br>"
        }
        if (v.failed && v.failed.length > 0) {
            text += window.lang.strings("failed") + ": " + v.failed.map((el: SendFailure) => {
                let err: string;
                switch (el.reason) {
                    case "CheckLogs":
                        err = window.lang.notif("errorCheckLogs");
                        break;
                    case "NoUser":
                        err = window.lang.notif("errorNoUser");
                        break;
                    case "MultiUser":
                        err = window.lang.notif("errorMultiUser");
                        break;
                    case "InvalidAddress":
                        err = window.lang.notif("errorInvalidAddress");
                        break;
                    default:
                        err = el.reason;
                        break;
                }
                return el.address + " (" + err + ")";
            }).join(", ");
        }
        if (text.length != 0) this.send_to = text;
    }

    private _usedBy: { [name: string]: number };
    get used_by(): { [name: string]: number } { return this._usedBy; }
    set used_by(uB: { [name: string]: number } | null) {
        this._usedBy = uB;
        if (!uB || Object.keys(uB).length == 0) {
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
    get notify_expiry(): boolean { return this._notifyExpiry }
    set notify_expiry(state: boolean) {
        this._notifyExpiry = state;
        (this._left.querySelector("input.inv-notify-expiry") as HTMLInputElement).checked = state;
    }

    private _notifyCreation: boolean = false;
    get notify_creation(): boolean { return this._notifyCreation }
    set notify_creation(state: boolean) {
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
        let state = {code: this.code};
        if (profile != "noProfile") { state["profile"] = profile };
        _patch("/invites/edit", state, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (!(req.status == 200 || req.status == 204)) {
                    select.value = previous || "noProfile";
                } else {
                    this._profile = profile;
                }
            }
        });
    }

    private _setLabel = () => {
        const newLabel = this._labelEditor.value.trim();
        const old = this.label;
        this.label = newLabel;
        let state = {
            code: this.code,
            label: newLabel
        };
        _patch("/invites/edit", state, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200 && req.status != 204) {
                this.label = old;
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

    private _linkContainer: HTMLElement;
    private _linkEl: HTMLAnchorElement;
    private _labelEditor: HiddenInputField;

    private _detailsToggle: HTMLInputElement;

    private _gap: number;
    get gap(): number { return this._gap; }
    set gap(v: number) {
        // Do it this way to ensure the class is included by tailwind
        let gapClass: string;
        switch (v) {
            case 1:
                gapClass = "gap-1";
                break;
            case 2:
                gapClass = "gap-2";
                break;
            case 3:
                gapClass = "gap-3";
                break;
            case 4:
                gapClass = "gap-4";
                break;
            default:
                gapClass = "gap-"+v;
                break;
        }
        this._container.classList.remove("gap-"+this._gap);
        this._container.classList.add(gapClass);
        this._gap = v;
    }

    // whether the details card is expanded.
    get expanded(): boolean {
        return this._detailsToggle.checked;
    }
    set expanded(state: boolean) {
        this._detailsToggle.checked = state;
        if (state) {
            this._detailsToggle.previousElementSibling.classList.add("rotated");
            this._detailsToggle.previousElementSibling.classList.remove("not-rotated");
            
            const mainTransitionStart = () => {
                this._details.removeEventListener("transitionend", mainTransitionStart);
                this._details.style.transitionDuration = "";
                this._details.addEventListener("transitionend", mainTransitionEnd);
                this._details.style.opacity = "100%";
                this._details.style.maxHeight = "calc(" + (1*this._details.scrollHeight)+"px" + " + " + (0.125 * 8 * this.gap) + "rem)"; // Compensate for the margin and padding (ugly)
                this._details.style.marginTop = "0";
                this._details.style.marginBottom = "0";
                this._details.style.paddingTop = "";
                this._details.style.paddingBottom = "";
            };
            const mainTransitionEnd = () => {
                this._details.removeEventListener("transitionend", mainTransitionEnd);
                this._details.style.maxHeight = "9999px";
            };
            this._details.classList.remove("unfocused");
            this._details.classList.add("focused");
            this._details.style.transitionDuration = "1ms";
            // Add negative y margin to cancel out "gap-x" when we unhide (and are initially height: 0)
            // perhaps not great assuming --spacing == 0.25rem
            this._details.style.marginTop = (-0.125 * this.gap)+"rem";
            this._details.style.marginBottom = (-0.125 * this.gap)+"rem";
            this._details.style.paddingTop = "0";
            this._details.style.paddingBottom = "0";
            mainTransitionStart();
        } else {
            this._detailsToggle.previousElementSibling.classList.remove("rotated");
            this._detailsToggle.previousElementSibling.classList.add("not-rotated");
            const mainTransitionEnd = () => {
                this._details.removeEventListener("transitionend", mainTransitionEnd);
                this._details.style.paddingTop = "";
                this._details.style.paddingBottom = "";
                this._details.style.marginTop = "0";
                this._details.style.marginBottom = "0";
                this._details.classList.add("unfocused");
                this._details.classList.remove("focused");
            };
            const mainTransitionStart = () => {
                this._details.removeEventListener("transitionend", mainTransitionStart);
                this._details.style.transitionDuration = "";
                this._details.addEventListener("transitionend", mainTransitionEnd);
                this._details.style.paddingTop = "0";
                this._details.style.paddingBottom = "0";
                this._details.style.maxHeight = "0";
                this._details.style.opacity = "0";
                // Add negative y margin to cancel out "gap-x" when we finish hiding (and end up height:0)
                // perhaps not great assuming --spacing == 0.25rem
                this._details.style.marginTop = (-0.125 * this.gap)+"rem";
                this._details.style.marginBottom = (-0.125 * this.gap)+"rem";
            };
            this._details.style.transitionDuration = "1ms";
            this._details.addEventListener("transitionend", mainTransitionStart);
            this._details.style.maxHeight = (1*this._details.scrollHeight)+"px";
        }
    }

    setExpandedWithoutAnimation(state: boolean) {
        this._detailsToggle.checked = state;
        if (state) {
            this._detailsToggle.previousElementSibling.classList.add("rotated");
            this._detailsToggle.previousElementSibling.classList.remove("not-rotated");
            
            this._details.classList.remove("unfocused");
            this._details.classList.add("focused");
            this._details.style.maxHeight = "9999px";
            this._details.style.opacity = "100%";
        } else {
            this._detailsToggle.previousElementSibling.classList.remove("rotated");
            this._detailsToggle.previousElementSibling.classList.add("not-rotated");
            this._details.classList.add("unfocused");
            this._details.classList.remove("focused");
            this._details.style.maxHeight = "0";
            this._details.style.opacity = "0";
        }
    }

    focus = () => this._container.scrollIntoView({ behavior: "smooth", block: "center" });

    constructor(invite: Invite) {
        // first create the invite structure, then use our setter methods to fill in the data.
        this._container = document.createElement('div') as HTMLDivElement;
        this._container.classList.add("inv", "overflow-visible", "flex", "flex-col");
       
        // Stores gap-x so we can cancel it out for transitions
        this.gap = 2;

        this._header = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._header);
        this._header.classList.add("card", "dark:~d_neutral", "@low", "inv-header", "flex", "flex-row", "justify-between", "overflow-visible", "gap-2", "z-[1]");

        this._codeArea = document.createElement('div') as HTMLDivElement;
        this._header.appendChild(this._codeArea);
        this._codeArea.classList.add("flex", "flex-row", "flex-wrap", "justify-between", "w-full", "items-center", "gap-2", "truncate");
        this._codeArea.innerHTML = `
        <div class="flex items-center gap-x-4 gap-y-2 truncate">
            <a class="invite-link-container text-black dark:text-white font-mono bg-inherit truncate"></a>
            <button class="invite-copy-button"></button>
        </div>
        <span>${window.lang.var("strings", "inviteExpiresInTime", "<span class=\"inv-duration\"></span>")}</span>
        `;

        this._linkContainer = this._codeArea.getElementsByClassName("invite-link-container")[0] as HTMLElement;
        this._labelEditor = new HiddenInputField({
            container: this._linkContainer,
            buttonOnLeft: false,
            customContainerHTML: `<a class="hidden-input-content invite-link text-black dark:text-white font-mono bg-inherit truncate"></a>`,
            clickAwayShouldSave: true,
            onSet: this._setLabel
        });
        this._linkEl = this._linkContainer.getElementsByClassName("invite-link")[0] as HTMLAnchorElement;

        const copyButton = this._codeArea.getElementsByClassName("invite-copy-button")[0] as HTMLButtonElement;
        SetupCopyButton(copyButton, this._codeLink); 

        this._infoArea = document.createElement('div') as HTMLDivElement;
        this._header.appendChild(this._infoArea);
        this._infoArea.classList.add("inv-infoarea", "flex", "flex-row", "items-center", "gap-2");
        this._infoArea.innerHTML = `
        <div class="tooltip below darker" tabindex="0">
            <span class="inv-email-chip h-full"><i></i></span>
            <span class="content sm p-1"></span>
        </div>
        <span class="button ~critical @low inv-delete h-full">${window.lang.strings("delete")}</span>
        <label>
            <i class="icon px-2.5 py-2 ri-arrow-down-s-line text-xl not-rotated"></i>
            <input class="inv-toggle-details unfocused" type="checkbox">
        </label>
        `;
        
        (this._infoArea.querySelector(".inv-delete") as HTMLSpanElement).onclick = this.delete;

        this._detailsToggle = (this._infoArea.querySelector("input.inv-toggle-details") as HTMLInputElement);
        this._detailsToggle.onclick = () => {
            this.expanded = this.expanded;
        };
        const toggleDetails = (event: Event) => { 
            if (event.target == this._header || event.target == this._codeArea || event.target == this._infoArea) {
                this.expanded = !this.expanded; 
            }
        };
        this._header.onclick = toggleDetails;


        this._details = document.createElement('div') as HTMLDivElement;
        this._container.appendChild(this._details);
        this._details.classList.add("card", "~neutral", "@low", "inv-details", "transition-all", "unfocused");
        this._details.style.maxHeight = "0";
        this._details.style.opacity = "0";
        const detailsInner = document.createElement('div') as HTMLDivElement;
        this._details.appendChild(detailsInner);
        detailsInner.classList.add("inv-row", "flex", "flex-row", "flex-wrap", "justify-between", "gap-4");

        this._left = document.createElement('div') as HTMLDivElement;
        this._left.classList.add("flex", "flex-row", "flex-wrap", "gap-4", "min-w-full", "sm:min-w-fit", "whitespace-nowrap");
        detailsInner.appendChild(this._left);
        const leftLeft = document.createElement("div") as HTMLDivElement;
        this._left.appendChild(leftLeft);
        leftLeft.classList.add("inv-profilearea", "min-w-full", "sm:min-w-fit", "flex", "flex-col", "gap-4");
        let innerHTML = `
        <label class="flex flex-col gap-2">
            <p class="label supra">${window.lang.strings("profile")}</p>
            <div class="select ~neutral @low inv-profileselect min-w-full inline-block">
                <select>
                    <option value="noProfile" selected>${window.lang.strings("inviteNoProfile")}</option>
                </select>
            </div>
        </label>
        `;
        if (window.notificationsEnabled) {
            innerHTML += `
            <div class="flex flex-col gap-2">
                <p class="label supra">${window.lang.strings("notifyEvent")}</p>
                <label class="switch block">
                    <input class="inv-notify-expiry" type="checkbox">
                    <span>${window.lang.strings("notifyInviteExpiry")}</span>
                </label>
                <label class="switch block">
                    <input class="inv-notify-creation" type="checkbox">
                    <span>${window.lang.strings("notifyUserCreation")}</span>
                </label>
            </div>
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
        this._middle.classList.add("flex", "flex-col", "grow", "gap-4");
        this._middle.innerHTML = `
        <p class="label flex items-center gap-2 supra">${window.lang.strings("inviteDateCreated")} <strong class="inv-created"></strong></p>
        <p class="label flex items-center gap-2 supra">${window.lang.strings("inviteRemainingUses")} <strong class="inv-remaining"></strong></p>
        <p class="label flex items-center gap-2 supra"><span class="user-expiry"></span> <strong class="user-expiry-time"></strong></p>
        <p class="flex items-center gap-2"><span class="user-label-label label supra"></span> <span class="user-label chip ~blue unfocused"></span></p>
        <div class="invite-send-to-dialog"></div>
        `;

        this._right = document.createElement('div') as HTMLDivElement;
        detailsInner.appendChild(this._right);
        this._right.classList.add("card", "~neutral", "@low", "inv-created-users", "min-w-full", "sm:min-w-fit", "whitespace-nowrap");
        this._right.innerHTML = `<span class="label supra table-header">${window.lang.strings("inviteUsersCreated")}</span>`;
        this._userTable = document.createElement('div') as HTMLDivElement;
        this._userTable.classList.add("text-sm", "mt-1", );
        this._right.appendChild(this._userTable);

        this.setExpandedWithoutAnimation(false);
        this.update(invite);

        document.addEventListener("profileLoadEvent", () => { this.loadProfiles(); }, false);
        document.addEventListener("timefmt-change", () => {
            this.created = this.created;
            this.used_by = this.used_by;
        });
    }

    update = (invite: Invite) => {
        this.code = invite.code;
        this.valid_till = invite.valid_till;
        if (invite.user_expiry) {
            this.user_expiry = invite.user_expiry;
            this.user_expiry_time = {
                months: invite.user_months,
                days: invite.user_days,
                hours: invite.user_hours,
                minutes: invite.user_minutes
            };
        }
        this.created = invite.created;
        this.profile = invite.profile;
        this.used_by = invite.used_by;
        this.no_limit = invite.no_limit ? invite.no_limit : false;
        this.remaining_uses = invite.remaining_uses;
        this.send_to = invite.send_to;
        this.sent_to = invite.sent_to;
        if (window.notificationsEnabled) {
            this.notify_creation = invite.notify_creation;
            this.notify_expiry = invite.notify_expiry;
        }
        if (invite.label) {
            this.label = invite.label;
        }
        if (invite.user_label) {
            this.user_label = invite.user_label;
        }
        this._sendToDialog = new SendToDialog(this._middle.getElementsByClassName("invite-send-to-dialog")[0] as HTMLElement, invite, () => {
            const needsUpdatingEvent = new CustomEvent("inviteNeedsUpdating", { detail: this.code });
            document.dispatchEvent(needsUpdatingEvent);
        });
    }

    asElement = (): HTMLDivElement => { return this._container; }

    remove = () => { this._container.remove(); }
}

export class DOMInviteList implements InviteList {
    private _list: HTMLDivElement;
    private _empty: boolean;
    // since invite reload sends profiles, this event it broadcast so the createInvite object can load them.

    invites: { [code: string]: DOMInvite };

    focusInvite = (inviteCode: string, errorMsg: string = window.lang.notif("errorInviteNoLongerExists")) => {
        for (let code of Object.keys(this.invites)) {
            this.invites[code].setExpandedWithoutAnimation(code == inviteCode);
        }
        if (inviteCode in this.invites) this.invites[inviteCode].focus();
        else window.notifications.customError("inviteDoesntExistError", errorMsg);
    };

    public static readonly _inviteURLEvent = "invite-url";
    registerURLListener = () => document.addEventListener(DOMInviteList._inviteURLEvent, (event: CustomEvent) => {
        this.focusInvite(event.detail);
    })

    isInviteURL = () => {
        const urlParams = new URLSearchParams(window.location.search);
        const inviteCode = urlParams.get("invite");
        return Boolean(inviteCode);
    }

    loadInviteURL = () => {
        const urlParams = new URLSearchParams(window.location.search);
        const inviteCode = urlParams.get("invite");
        this.focusInvite(inviteCode, window.lang.notif("errorInviteNotFound"));
    }

    constructor() {
        this._list = document.getElementById('invites') as HTMLDivElement;
        this.empty = true;
        this.invites = {};
        // FIXME: Do this better, take advantage of getting the code in e.detail
        document.addEventListener("inviteNeedsUpdating", () => { this.reload(); }, false);

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
                <div class="card dark:~d_neutral @low inv-header">
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

    reload = (callback?: () => void) => reloadProfileNames(() => _get("/invites", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            let data = req.response;
            if (data["invites"] === undefined || data["invites"] == null || data["invites"].length == 0) {
                this.empty = true;
                return;
            }
            // get a list of all current inv codes on dom
            // every time we find a match in resp, delete from list
            // at end delete all remaining in list from dom
            let invitesOnDOM: { [code: string]: boolean } = {};
            for (let code in this.invites) { invitesOnDOM[code] = true; }
            for (let invite of (data["invites"] as Array<Invite>)) {
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
    }));
}

export const inviteURLEvent = (id: string) => { return new CustomEvent(DOMInviteList._inviteURLEvent, {"detail": id}) };

export class createInvite {
    private _sendTo: SendToDialog;
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

    get sendTo(): string {
        if (!(this._sendTo)) return "";
        if (this._sendTo.addresses.length > 1) console.error("FIXME: SendToDialog has collected more than one address, make them usable or fix it!");
        if (this._sendTo.addresses.length > 0) return this._sendTo.addresses[0];
        else return "";
    }
    set sendTo(address: string) { if (!(this._sendTo)) return; this._sendTo.addresses = [address]; }

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
            "send-to": this.sendTo,
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

        const sendToContainer = document.getElementById("create-send-to-container");
        if (window.emailEnabled || window.discordEnabled) {
            this._sendTo = new SendToDialog(sendToContainer);
        } else { 
            sendToContainer.classList.add("unfocused");
        }
    }
}

class SendToDialog {
    private _container: HTMLElement;
    private _input: HTMLInputElement;
    private _submit?: HTMLButtonElement;
    // FIXME: Make an interface for multiple addresses
    // private _addresses: string[] = [];
    get addresses(): string[] {
        if (this._input.value) return [this._input.value];
        return [];
    }
    set addresses(v: string[]) {
        if (v.length > 0) this._input.value = v[0];
        // this._addresses = v;
    };

    private _search?: HTMLButtonElement;
    private _discordSearch?: DiscordSearch;


    constructor(container: HTMLElement, invite?: Invite, onSuccess?: () => void) {
        this._container = container;
        this._container.classList.add("flex", "flex-col", "gap-2");
        this._container.innerHTML = `
            <label class="label supra">${window.lang.strings("inviteSendToEmail")}</label>
            <div class="flex flex-row gap-2">
                <input class="input ~neutral @low send-to-dialog-input" type="email" placeholder="example@example.com">
                <button class="button ~urge @low send-to-dialog-search unfocused" title="${window.lang.strings("search")}">
                    <i class="icon ri-search-2-line"></i>
                </button>
                <button class="button ~urge @low send-to-dialog-submit unfocused" title="${window.lang.strings("submit")}">
                    <i class="icon ri-send-plane-2-line"></i>
                </button>
            </div>
        `;
        this._input = this._container.getElementsByClassName("send-to-dialog-input")[0] as HTMLInputElement;
        if (window.discordEnabled) {
            this._input.type = "text";
            this._input.placeholder = "example@example.com | user#1234";
            this._search = this._container.getElementsByClassName("send-to-dialog-search")[0] as HTMLButtonElement;
            this._search.classList.remove("unfocused");
            this._discordSearch = newDiscordSearch(
                window.lang.strings("findDiscordUser"),
                window.lang.strings("searchDiscordUser"),
                window.lang.strings("select"),
                (user: DiscordUser) => {
                    this.addresses = [user.name];
                    // this.addresses.push(user.name);

                    window.modals.discord.close();
                }
            );
            // FIXME: Check why we're passing an empty string rather than the input value
            this._search.onclick = () => this._discordSearch("");
        }

        if (invite) {
            if (this._search) {
                this._search.classList.add("~neutral");
                this._search.classList.remove("~urge");
            }
            this._submit = this._container.getElementsByClassName("send-to-dialog-submit")[0] as HTMLButtonElement;
            this._submit.classList.remove("unfocused");
            this._submit.onclick = () => {
                const icon = this._submit.children[0] as HTMLElement;
                addLoader(icon, true);
                if (this.addresses.length == 0) return;
                _post("/invites/send", {"invite": invite.code, "send-to": this.addresses[0]}, (req: XMLHttpRequest) => {
                    if (req.readyState != 4) return;
                    removeLoader(icon, true)
                    if (req.status != 200 && req.status != 204) {
                        window.notifications.customError("errorSendInvite", window.lang.notif("errorFailureCheckLogs"));
                        return;
                    }
                    window.notifications.customSuccess("sendInvite", window.lang.strings("sent"));
                    if (onSuccess) onSuccess();
                    this.addresses = [];
                });
            };
            this._input.addEventListener("keypress", (e: KeyboardEvent) => {
                if (e.key === "Enter") {
                    e.preventDefault();
                    this._submit.click();
                }
            })
        }
    }
}
