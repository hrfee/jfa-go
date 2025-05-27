import { _get, _post, _delete, toggleLoader, addLoader, removeLoader, toDateString, insertText, toClipboard } from "../modules/common"
import { templateEmail } from "../modules/settings"
import { Marked } from "@ts-stack/markdown";
import { stripMarkdown } from "../modules/stripmd"
import { DiscordUser, newDiscordSearch } from "../modules/discord"
import { SearchConfiguration, QueryType, SearchableItem, SearchableItemDataAttribute } from "../modules/search"
import { HiddenInputField } from "./ui"
import { PaginatedList } from "./list"

// FIXME: Find and define a threshold after which searches are no longer performed on input (or just in general by the browser).

declare var window: GlobalWindow;

const USER_DEFAULT_SORT_FIELD      = "name";
const USER_DEFAULT_SORT_ASCENDING  = true;

const dateParser = require("any-date-parser");

interface User {
    id: string;
    name: string;
    email: string | undefined;
    notify_email: boolean;
    last_active: number;
    admin: boolean;
    disabled: boolean;
    expiry: number;
    telegram: string;
    notify_telegram: boolean;
    discord: string;
    notify_discord: boolean;
    discord_id: string;
    matrix: string;
    notify_matrix: boolean;
    label: string;
    accounts_admin: boolean;
    referrals_enabled: boolean;
}

interface getPinResponse {
    token: string;
    username: string;
}

interface announcementTemplate {
    name: string;
    subject: string;
    message: string;
}

var addDiscord: (passData: string) => void;
    
const queries = (): { [field: string]: QueryType } => { return {
    "id": {
        // We don't use a translation here to circumvent the name substitution feature.
        name: "Jellyfin/Emby ID",
        getter: "id",
        bool: false,
        string: true,
        date: false
    },
    "label": {
        name: window.lang.strings("label"),
        getter: "label",
        bool: true,
        string: true,
        date: false
    },
    "username": {
        name: window.lang.strings("username"),
        getter: "name",
        bool: false,
        string: true,
        date: false
    },
    "name": {
        name: window.lang.strings("username"),
        getter: "name",
        bool: false,
        string: true,
        date: false,
        show: false
    },
    "admin": {
        name: window.lang.strings("admin"),
        getter: "admin",
        bool: true,
        string: false,
        date: false
    },
    "disabled": {
        name: window.lang.strings("disabled"),
        getter: "disabled",
        bool: true,
        string: false,
        date: false
    },
    "access-jfa": {
        name: window.lang.strings("accessJFA"),
        getter: "accounts_admin",
        bool: true,
        string: false,
        date: false,
        dependsOnElement: ".accounts-header-access-jfa"
    },
    "email": {
        name: window.lang.strings("emailAddress"),
        getter: "email",
        bool: true,
        string: true,
        date: false,
        dependsOnElement: ".accounts-header-email"
    },
    "telegram": {
        name: "Telegram",
        getter: "telegram",
        bool: true,
        string: true,
        date: false,
        dependsOnElement: ".accounts-header-telegram"
    },
    "matrix": {
        name: "Matrix",
        getter: "matrix",
        bool: true,
        string: true,
        date: false,
        dependsOnElement: ".accounts-header-matrix"
    },
    "discord": {
        name: "Discord",
        getter: "discord",
        bool: true,
        string: true,
        date: false,
        dependsOnElement: ".accounts-header-discord"
    },
    "expiry": {
        name: window.lang.strings("expiry"),
        getter: "expiry",
        bool: true,
        string: false,
        date: true,
        dependsOnElement: ".accounts-header-expiry"
    },
    "last-active": {
        name: window.lang.strings("lastActiveTime"),
        getter: "last_active",
        bool: true,
        string: false,
        date: true
    },
    "referrals-enabled": {
        name: window.lang.strings("referrals"),
        getter: "referrals_enabled",
        bool: true,
        string: false,
        date: false,
        dependsOnElement: ".accounts-header-referrals"
    }
}};


class user implements User, SearchableItem {
    private _id = "";
    private _row: HTMLTableRowElement;
    private _check: HTMLInputElement;
    private _username: HTMLSpanElement;
    private _admin: HTMLSpanElement;
    private _disabled: HTMLSpanElement;
    private _email: HTMLInputElement;
    private _emailEditor: HiddenInputField;
    private _notifyEmail: boolean;
    private _emailAddress: string;
    private _telegram: HTMLTableDataCellElement;
    private _telegramUsername: string;
    private _notifyTelegram: boolean;
    private _discord: HTMLTableDataCellElement;
    private _discordUsername: string;
    private _discordID: string;
    private _notifyDiscord: boolean;
    private _matrix: HTMLTableDataCellElement;
    private _matrixID: string;
    private _notifyMatrix: boolean;
    private _expiry: HTMLTableDataCellElement;
    private _expiryUnix: number;
    private _lastActive: HTMLTableDataCellElement;
    private _lastActiveUnix: number;
    private _notifyDropdown: HTMLDivElement;
    private _label: HTMLInputElement;
    private _labelEditor: HiddenInputField;
    private _userLabel: string;
    private _accounts_admin: HTMLInputElement
    private _selected: boolean;
    private _referralsEnabled: boolean;
    private _referralsEnabledCheck: HTMLElement;

    focus = () => this._row.scrollIntoView({ behavior: "smooth", block: "center" });

    lastNotifyMethod = (): string => {
        // Telegram, Matrix, Discord
        const telegram = window.telegramEnabled && this._telegramUsername && this._telegramUsername != "";
        const discord = window.discordEnabled && this._discordUsername && this._discordUsername != "";
        const matrix = window.matrixEnabled && this._matrixID && this._matrixID != "";
        const email = window.emailEnabled && this.email != "";
        if (discord) return "discord";
        if (matrix) return "matrix";
        if (telegram) return "telegram";
        if (email) return "email";
    }

    private _checkUnlinkArea = () => {
        const unlinkHeader = this._notifyDropdown.querySelector(".accounts-unlink-header") as HTMLSpanElement;
        if (this.lastNotifyMethod() == "email" || !this.lastNotifyMethod()) {
            unlinkHeader.classList.add("unfocused");
        } else {
            unlinkHeader.classList.remove("unfocused");
        }
    }

    get selected(): boolean { return this._selected; }
    set selected(state: boolean) {
        this._selected = state;
        this._check.checked = state;
        state ? document.dispatchEvent(this._checkEvent()) : document.dispatchEvent(this._uncheckEvent());
    }

    get name(): string { return this._username.textContent; }
    set name(value: string) { this._username.textContent = value; }

    get admin(): boolean { return !(this._admin.classList.contains("hidden")); }
    set admin(state: boolean) {
        if (state) {
            this._admin.classList.remove("hidden")
            this._admin.textContent = window.lang.strings("admin");
        } else {
            this._admin.classList.add("hidden")
            this._admin.textContent = "";
        }
    }

    get accounts_admin(): boolean { return this._accounts_admin.checked; }
    set accounts_admin(a: boolean) {
        if (!window.jellyfinLogin) return;
        this._accounts_admin.checked = a;
        this._accounts_admin.disabled = (window.jfAllowAll || (a && this.admin && window.jfAdminOnly));
        if (this._accounts_admin.disabled) {
            this._accounts_admin.title = window.lang.strings("accessJFASettings");
        } else {
            this._accounts_admin.title = "";
        }
    }

    get disabled(): boolean { return !(this._disabled.classList.contains("hidden")); }
    set disabled(state: boolean) {
        if (state) {
            this._disabled.classList.remove("hidden")
            this._disabled.textContent = window.lang.strings("disabled");
        } else {
            this._disabled.classList.add("hidden")
            this._disabled.textContent = "";
        }
    }

    get email(): string { return this._emailAddress; }
    set email(value: string) {
        this._emailAddress = value;
        this._emailEditor.value = value;
        const lastNotifyMethod = this.lastNotifyMethod() == "email";
        if (!value) {
            this._notifyDropdown.querySelector(".accounts-area-email").classList.add("unfocused");
        } else {
            this._notifyDropdown.querySelector(".accounts-area-email").classList.remove("unfocused");
            if (lastNotifyMethod) {
                (this._email.parentElement as HTMLDivElement).appendChild(this._notifyDropdown);
            }
        }
    }

    get notify_email(): boolean { return this._notifyEmail; }
    set notify_email(s: boolean) {
        if (this._notifyDropdown) {
            (this._notifyDropdown.querySelector(".accounts-contact-email") as HTMLInputElement).checked = s;
        }
    }

    get referrals_enabled(): boolean { return this._referralsEnabled; }
    set referrals_enabled(v: boolean) {
        this._referralsEnabled = v;
        if (!window.referralsEnabled) return;
        if (!v) {
            this._referralsEnabledCheck.textContent = ``;
        } else {
            this._referralsEnabledCheck.innerHTML = `<i class="ri-check-line" aria-label="${window.lang.strings("enabled")}"></i>`;
        }
    }

    private _constructDropdown = (): HTMLDivElement => {
        const el = document.createElement("div") as HTMLDivElement;
        const telegram = this._telegramUsername != "";
        const discord = this._discordUsername != "";
        const matrix = this._matrixID != "";
        const email = this._emailAddress != "";
        if (!telegram && !discord && !matrix && !email) return;
        let innerHTML = `
        <i class="icon ri-settings-2-line ml-2 dropdown-button"></i>
        <div class="dropdown manual over-top">
            <div class="dropdown-display lg">
                <div class="card ~neutral @low">
                    <div class="supra sm mb-2">${window.lang.strings("contactThrough")}</div>
                    <div class="accounts-area-email">
                        <label class="row switch pb-2">
                            <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-email mr-2">
                            </span>Email</span>
                        </label>
                    </div>
                    <div class="accounts-area-telegram">
                        <label class="row switch pb-2">
                            <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-telegram mr-2">
                            <span>Telegram</span>
                        </label>
                    </div>
                    <div class="accounts-area-discord">
                        <label class="row switch pb-2">
                            <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-discord mr-2">
                            <span>Discord</span>
                        </label>
                    </div>
                    <div class="accounts-area-matrix">
                        <label class="row switch pb-2">
                            <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-matrix mr-2">
                            <span>Matrix</span>
                        </label>
                    </div>
                    <div class="supra sm mb-2 accounts-unlink-header">${window.lang.strings("unlink")}:</div>
                    <div class="accounts-unlink-telegram"> 
                        <button class="button ~critical mb-2 w-full">Telegram</button>
                    </div>
                    <div class="accounts-unlink-discord"> 
                        <button class="button ~critical mb-2 w-full">Discord</button>
                    </div>
                    <div class="accounts-unlink-matrix"> 
                        <button class="button ~critical mb-2 w-full">Matrix</button>
                    </div>
                </div>
            </div>
        </div>
        `;
        el.innerHTML = innerHTML;
        const button = el.querySelector("i");
        const dropdown = el.querySelector("div.dropdown") as HTMLDivElement;
        const checks = el.querySelectorAll("input") as NodeListOf<HTMLInputElement>;
        for (let i = 0; i < checks.length; i++) {
            checks[i].onclick = () => this._setNotifyMethod();
        }
        
        for (let service of ["telegram", "discord", "matrix"]) {
            el.querySelector(".accounts-unlink-"+service).addEventListener("click", () => _delete(`/users/${service}`, {"id": this.id}, () => document.dispatchEvent(new CustomEvent("accounts-reload"))));
        }

        button.onclick = () => {
            dropdown.classList.add("selected");
            document.addEventListener("click", outerClickListener);
        };
        const outerClickListener = (event: Event) => {
            if (!(event.target instanceof HTMLElement && (el.contains(event.target) || button.contains(event.target)))) {
                dropdown.classList.remove("selected");
                document.removeEventListener("click", outerClickListener);
            }
        };
        return el;
    }

    get matrix(): string { return this._matrixID; }
    set matrix(u: string) {
        if (!window.matrixEnabled) {
            this._notifyDropdown.querySelector(".accounts-area-matrix").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-matrix").classList.add("unfocused");
            return;
        }
        const lastNotifyMethod = this.lastNotifyMethod() == "matrix";
        this._matrixID = u;
        if (!u) {
            this._notifyDropdown.querySelector(".accounts-area-matrix").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-matrix").classList.add("unfocused");
            this._matrix.innerHTML = `
            <div class="table-inline justify-center">
                <span class="chip btn @low"><i class="ri-link" alt="${window.lang.strings("add")}"></i></span>
                <input type="text" class="input ~neutral @low stealth-input unfocused" placeholder="@user:riot.im">
            </div>
        `;
            (this._matrix.querySelector("span") as HTMLSpanElement).onclick = this._addMatrix;
        } else {
            this._notifyDropdown.querySelector(".accounts-area-matrix").classList.remove("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-matrix").classList.remove("unfocused");
            this._matrix.innerHTML = `
            <div class="table-inline">
                ${u}
            </div>
            `;
            if (lastNotifyMethod) {
                (this._matrix.querySelector(".table-inline") as HTMLDivElement).appendChild(this._notifyDropdown);
            }
        }
        this._checkUnlinkArea();
    }
   
    private _addMatrix = () => {
        const addButton = this._matrix.querySelector(".btn") as HTMLSpanElement;
        const input = this._matrix.querySelector("input.stealth-input") as HTMLInputElement;
        const addIcon = addButton.querySelector("i");
        if (addButton.classList.contains("chip")) {
            input.classList.remove("unfocused");
            addIcon.classList.add("ri-check-line");
            addIcon.classList.remove("ri-link");
            addButton.classList.remove("chip")
            const outerClickListener = (event: Event) => {
                if (!(event.target instanceof HTMLElement && (this._matrix.contains(event.target) || addButton.contains(event.target)))) {
                    document.dispatchEvent(new CustomEvent("accounts-reload"));
                    document.removeEventListener("click", outerClickListener);
                }
            };
            document.addEventListener("click", outerClickListener);
        } else {
            if (input.value.charAt(0) != "@" || !input.value.includes(":")) return;
            const send = {
                jf_id: this.id,
                user_id: input.value
            }
            _post("/users/matrix", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    document.dispatchEvent(new CustomEvent("accounts-reload"));
                    if (req.status != 200) {
                        window.notifications.customError("errorConnectMatrix", window.lang.notif("errorFailureCheckLogs"));
                        return;
                    }
                    window.notifications.customSuccess("connectMatrix", window.lang.notif("accountConnected"));
                }
            });
        }
    }

    get notify_matrix(): boolean { return this._notifyMatrix; }
    set notify_matrix(s: boolean) {
        if (this._notifyDropdown) {
            (this._notifyDropdown.querySelector(".accounts-contact-matrix") as HTMLInputElement).checked = s;
        }
    }
    
    get telegram(): string { return this._telegramUsername; }
    set telegram(u: string) {
        if (!window.telegramEnabled) {
            this._notifyDropdown.querySelector(".accounts-area-telegram").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-telegram").classList.add("unfocused");
            return;
        }
        const lastNotifyMethod = this.lastNotifyMethod() == "telegram";
        this._telegramUsername = u;
        if (!u) {
            this._notifyDropdown.querySelector(".accounts-area-telegram").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-telegram").classList.add("unfocused");
            this._telegram.innerHTML = `<div class="table-inline justify-center"><span class="chip btn @low"><i class="ri-link" alt="${window.lang.strings("add")}"></i></span></div>`;
            (this._telegram.querySelector("span") as HTMLSpanElement).onclick = this._addTelegram;
        } else {
            this._notifyDropdown.querySelector(".accounts-area-telegram").classList.remove("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-telegram").classList.remove("unfocused");
            this._telegram.innerHTML = `
            <div class="table-inline">
                <a href="https://t.me/${u}" target="_blank">@${u}</a>
            </div>
            `;
            if (lastNotifyMethod) {
                (this._telegram.querySelector(".table-inline") as HTMLDivElement).appendChild(this._notifyDropdown);
            }
        }
        this._checkUnlinkArea();
    }
    
    get notify_telegram(): boolean { return this._notifyTelegram; }
    set notify_telegram(s: boolean) {
        if (this._notifyDropdown) {
            (this._notifyDropdown.querySelector(".accounts-contact-telegram") as HTMLInputElement).checked = s;
        }
    }

    private _setNotifyMethod = () => {
        const email = this._notifyDropdown.getElementsByClassName("accounts-contact-email")[0] as HTMLInputElement;
        let send = {
            id: this.id,
            email: email.checked
        }
        if (window.telegramEnabled && this._telegramUsername) {
            const telegram = this._notifyDropdown.getElementsByClassName("accounts-contact-telegram")[0] as HTMLInputElement;
            send["telegram"] = telegram.checked;
        }
        if (window.discordEnabled && this._discordUsername) {
            const discord = this._notifyDropdown.getElementsByClassName("accounts-contact-discord")[0] as HTMLInputElement;
            send["discord"] = discord.checked;
        }
        _post("/users/contact", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    window.notifications.customError("errorSetNotify", window.lang.notif("errorSaveSettings"));
                    document.dispatchEvent(new CustomEvent("accounts-reload"));
                    return;
                }
            }
        }, false, (req: XMLHttpRequest) => {
            if (req.status == 0) {
                window.notifications.connectionError();
                document.dispatchEvent(new CustomEvent("accounts-reload"));
                return;
            } else if (req.status == 401) {
                window.notifications.customError("401Error", window.lang.notif("error401Unauthorized"));
                document.dispatchEvent(new CustomEvent("accounts-reload"));
            }
        });
    }
    
    get discord(): string { return this._discordUsername; }
    set discord(u: string) {
        if (!window.discordEnabled) {
            this._notifyDropdown.querySelector(".accounts-area-discord").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-discord").classList.add("unfocused");
            return;
        }
        const lastNotifyMethod = this.lastNotifyMethod() == "discord";
        this._discordUsername = u;
        if (!u) {
            this._discord.innerHTML = `<div class="table-inline justify-center"><span class="chip btn @low"><i class="ri-link" alt="${window.lang.strings("add")}"></i></span></div>`;
            (this._discord.querySelector("span") as HTMLSpanElement).onclick = () => addDiscord(this.id);
            this._notifyDropdown.querySelector(".accounts-area-discord").classList.add("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-discord").classList.add("unfocused");
        } else {
            this._notifyDropdown.querySelector(".accounts-area-discord").classList.remove("unfocused");
            this._notifyDropdown.querySelector(".accounts-unlink-discord").classList.remove("unfocused");
            this._discord.innerHTML = `
            <div class="table-inline">
                <a href="https://discord.com/users/${this._discordID}" class="discord-link" target="_blank">${u}</a>
            </div>
            `;
            if (lastNotifyMethod) {
                (this._discord.querySelector(".table-inline") as HTMLDivElement).appendChild(this._notifyDropdown);
            }
        }
        this._checkUnlinkArea();
    }

    get discord_id(): string { return this._discordID; }
    set discord_id(id: string) {
        if (!window.discordEnabled || this._discordUsername == "") return; 
        this._discordID = id;
        const link = this._discord.getElementsByClassName("discord-link")[0] as HTMLAnchorElement;
        link.href = `https://discord.com/users/${id}`;
    }
    
    get notify_discord(): boolean { return this._notifyDiscord; }
    set notify_discord(s: boolean) {
        if (this._notifyDropdown) {
            (this._notifyDropdown.querySelector(".accounts-contact-discord") as HTMLInputElement).checked = s;
        }
    }

    get expiry(): number { return this._expiryUnix; }
    set expiry(unix: number) {
        this._expiryUnix = unix;
        if (unix == 0) {
            this._expiry.textContent = "";
        } else {
            this._expiry.textContent = toDateString(new Date(unix*1000));
        }
    }

    get last_active(): number { return this._lastActiveUnix; }
    set last_active(unix: number) {
        this._lastActiveUnix = unix;
        if (unix == 0) {
            this._lastActive.textContent == "n/a";
        } else {
            this._lastActive.textContent = toDateString(new Date(unix*1000));
        }
    }

    get label(): string { return this._userLabel; }
    set label(l: string) {
        this._userLabel = l ? l : "";
        this._labelEditor.value = l ? l : "";
    }

    matchesSearch = (query: string): boolean => {
        return (
            this.id.includes(query) ||
            this.name.toLowerCase().includes(query) ||
            this.label.toLowerCase().includes(query) ||
            this.email.toLowerCase().includes(query) ||
            this.discord.toLowerCase().includes(query) ||
            this.matrix.toLowerCase().includes(query) ||
            this.telegram.toLowerCase().includes(query)
        );
    }

    private _checkEvent = () => new CustomEvent("accountCheckEvent", {detail: this.id});
    private _uncheckEvent = () => new CustomEvent("accountUncheckEvent", {detail: this.id});

    constructor(user: User) {
        this._row = document.createElement("tr") as HTMLTableRowElement;
        this._row.classList.add("border-b", "border-dashed", "dark:border-dotted", "dark:border-stone-700");
        let innerHTML = `
            <td><input type="checkbox" class="accounts-select-user" value=""></td>
            <td><div class="flex flex-row gap-2 items-center">
                <span class="accounts-username"></span>
                <div class="flex flex-row gap-2 items-baseline">
                    <span class="accounts-label-container" title="${window.lang.strings("label")}"></span>
                    <span class="accounts-admin chip ~info hidden"></span>
                    <span class="accounts-disabled chip ~warning hidden"></span></span>
                </div>
            </div></td>
        `;
        if (window.jellyfinLogin) {
            innerHTML += `
            <td><div class="table-inline justify-center"><input type="checkbox" class="accounts-access-jfa" value=""></div></td>
            `;
        }
        innerHTML += `
            <td><div class="flex flex-row gap-2 items-baseline">
                <span class="accounts-email-container" title="${window.lang.strings("emailAddress")}"></span>
            </div></td>
        `;
        if (window.telegramEnabled) {
            innerHTML += `
            <td class="accounts-telegram"></td>
            `;
        }
        if (window.matrixEnabled) {
            innerHTML += `
            <td class="accounts-matrix"></td>
            `;
        }
        if (window.discordEnabled) {
            innerHTML += `
            <td class="accounts-discord"></td>
            `;
        }
        if (window.referralsEnabled) {
            innerHTML += `
            <td class="accounts-referrals text-center-i grid gap-4 place-items-stretch"></td>
            `;
        }
        innerHTML += `
        <td class="accounts-expiry"></td>
        <td class="accounts-last-active whitespace-nowrap"></td>
        `;
        this._row.innerHTML = innerHTML;
        this._check = this._row.querySelector("input[type=checkbox].accounts-select-user") as HTMLInputElement;
        this._accounts_admin = this._row.querySelector("input[type=checkbox].accounts-access-jfa") as HTMLInputElement;
        this._username = this._row.querySelector(".accounts-username") as HTMLSpanElement;
        this._admin = this._row.querySelector(".accounts-admin") as HTMLSpanElement;
        this._disabled = this._row.querySelector(".accounts-disabled") as HTMLSpanElement;
        this._email = this._row.querySelector(".accounts-email-container") as HTMLInputElement;
        this._emailEditor = new HiddenInputField({
            container: this._email,
            onSet: this._updateEmail,
            customContainerHTML: `<span class="hidden-input-content"></span>`,
            buttonOnLeft: true,
            clickAwayShouldSave: false,
        });
        this._telegram = this._row.querySelector(".accounts-telegram") as HTMLTableDataCellElement;
        this._discord = this._row.querySelector(".accounts-discord") as HTMLTableDataCellElement;
        this._matrix = this._row.querySelector(".accounts-matrix") as HTMLTableDataCellElement;
        this._expiry = this._row.querySelector(".accounts-expiry") as HTMLTableDataCellElement;
        this._lastActive = this._row.querySelector(".accounts-last-active") as HTMLTableDataCellElement;
        this._label = this._row.querySelector(".accounts-label-container") as HTMLInputElement;
        this._labelEditor = new HiddenInputField({
            container: this._label,
            onSet: this._updateLabel,
            customContainerHTML: `<span class="chip ~gray hidden-input-content"></span>`,
            buttonOnLeft: true,
            clickAwayShouldSave: false,
        });

        this._check.onchange = () => { this.selected = this._check.checked; }
        
        if (window.jellyfinLogin) {
            this._accounts_admin.onchange = () => {
                this.accounts_admin = this._accounts_admin.checked;
                let send = {};
                send[this.id] = this.accounts_admin;
                _post("/users/accounts-admin", send, (req: XMLHttpRequest) => {
                    if (req.readyState == 4) {
                        if (req.status != 204) {
                            this.accounts_admin = !this.accounts_admin;
                            window.notifications.customError("accountsAdminChanged", window.lang.notif("errorUnknown"));
                        }
                    }
                });
            };
        }
        
        if (window.referralsEnabled) {
            this._referralsEnabledCheck = this._row.querySelector(".accounts-referrals");
        }

        this._notifyDropdown = this._constructDropdown();

        this.update(user);
        
        document.addEventListener("timefmt-change", () => {
            this.expiry = this.expiry;
            this.last_active = this.last_active;
        });
    }
    
    private _updateLabel = () => {
        let send = {};
        send[this.id] = this._labelEditor.value;
        _post("/users/labels", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 204) {
                    this.label = this._labelEditor.previous;
                    window.notifications.customError("labelChanged", window.lang.notif("errorUnknown"));
                }
            }
        });
    };

    private _updateEmail = () => {
        let send = {};
        send[this.id] = this._emailEditor.value;
        _post("/users/emails", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200) {
                    window.notifications.customSuccess("emailChanged", window.lang.var("notifications", "changedEmailAddress", `"${this.name}"`));
                } else {
                    this.email = this._emailEditor.previous;
                    window.notifications.customError("emailChanged", window.lang.var("notifications", "errorChangedEmailAddress", `"${this.name}"`)); 
                }
            }
        });
    }
    
    private _addTelegram = () => _get("/telegram/pin", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4 && req.status == 200) {
            const pin = document.getElementById("telegram-pin");
            const link = document.getElementById("telegram-link") as HTMLAnchorElement;
            const username = document.getElementById("telegram-username") as HTMLSpanElement;
            const waiting = document.getElementById("telegram-waiting") as HTMLSpanElement;
            let resp = req.response as getPinResponse;
            pin.textContent = resp.token;
            link.href = "https://t.me/" + resp.username;
            username.textContent = resp.username;
            addLoader(waiting);
            let modalClosed = false;
            window.modals.telegram.onclose = () => { 
                modalClosed = true;
                removeLoader(waiting);
            }
            let send = {
                token: resp.token,
                id: this.id
            };
            const checkVerified = () => _post("/users/telegram", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status == 200 && req.response["success"] as boolean) {
                        removeLoader(waiting);
                        waiting.classList.add("~positive");
                        waiting.classList.remove("~info");
                        window.notifications.customSuccess("telegramVerified", window.lang.notif("telegramVerified"));
                        setTimeout(() => {
                            window.modals.telegram.close();
                            waiting.classList.add("~info");
                            waiting.classList.remove("~positive");
                        }, 2000);
                        document.dispatchEvent(new CustomEvent("accounts-reload"));
                    } else if (!modalClosed) {
                        setTimeout(checkVerified, 1500);
                    }
                }
            }, true);
            window.modals.telegram.show();
            checkVerified();
        }
    });

    get id() { return this._id; }
    set id(v: string) {
        this._id = v;
        this._row.setAttribute(SearchableItemDataAttribute, v);
    }


    update = (user: User) => {
        this.id = user.id;
        this.name = user.name;
        this.email = user.email || "";
        // Little hack to get settings cogs to appear on first load
        this._discordUsername = user.discord;
        this._telegramUsername = user.telegram;
        this._matrixID = user.matrix;
        this.discord = user.discord;
        this.telegram = user.telegram;
        this.matrix = user.matrix;
        this.last_active = user.last_active;
        this.admin = user.admin;
        this.disabled = user.disabled;
        this.expiry = user.expiry;
        this.notify_discord = user.notify_discord;
        this.notify_telegram = user.notify_telegram;
        this.notify_matrix = user.notify_matrix;
        this.notify_email = user.notify_email;
        this.discord_id = user.discord_id;
        this.label = user.label;
        this.accounts_admin = user.accounts_admin;
        this.referrals_enabled = user.referrals_enabled;
    }

    asElement = (): HTMLTableRowElement => { return this._row; }
    remove = () => {
        if (this.selected) {
            document.dispatchEvent(this._uncheckEvent());
        }
        this._row.remove(); 
    }
}  

interface UsersDTO extends paginatedDTO {
    users: User[];
}

export class accountsList extends PaginatedList {
    protected _container = document.getElementById("accounts-list") as HTMLTableSectionElement;
    
    private _addUserButton = document.getElementById("accounts-add-user") as HTMLSpanElement;
    private _announceButton = document.getElementById("accounts-announce") as HTMLSpanElement;
    private _announceSaveButton = document.getElementById("save-announce") as HTMLSpanElement;
    private _announceNameLabel = document.getElementById("announce-name") as HTMLLabelElement;
    private _announcePreview: HTMLElement;
    private _previewLoaded = false;
    private _announceTextarea = document.getElementById("textarea-announce") as HTMLTextAreaElement;
    private _deleteUser = document.getElementById("accounts-delete-user") as HTMLSpanElement;
    private _disableEnable = document.getElementById("accounts-disable-enable") as HTMLSpanElement;
    private _enableExpiry = document.getElementById("accounts-enable-expiry") as HTMLSpanElement;
    private _deleteNotify = document.getElementById("delete-user-notify") as HTMLInputElement;
    private _deleteReason = document.getElementById("textarea-delete-user") as HTMLTextAreaElement;
    private _expiryDropdown = document.getElementById("accounts-expiry-dropdown") as HTMLElement;
    private _extendExpiry = document.getElementById("accounts-extend-expiry") as HTMLSpanElement;
    private  _extendExpiryForm = document.getElementById("form-extend-expiry") as HTMLFormElement;
    private _extendExpiryTextInput = document.getElementById("extend-expiry-text") as HTMLInputElement;
    private _extendExpiryFieldInputs = document.getElementById("extend-expiry-field-inputs") as HTMLElement;
    private _usingExtendExpiryTextInput = true;

    private _extendExpiryDate = document.getElementById("extend-expiry-date") as HTMLElement;
    private _removeExpiry = document.getElementById("accounts-remove-expiry") as HTMLSpanElement;
    private _enableExpiryNotify = document.getElementById("expiry-extend-enable") as HTMLInputElement;
    private _enableExpiryReason = document.getElementById("textarea-extend-enable") as HTMLTextAreaElement;
    private _modifySettings = document.getElementById("accounts-modify-user") as HTMLSpanElement;
    private _modifySettingsProfile = document.getElementById("radio-use-profile") as HTMLInputElement;
    private _modifySettingsUser = document.getElementById("radio-use-user") as HTMLInputElement;
    private _enableReferrals = document.getElementById("accounts-enable-referrals") as HTMLSpanElement;
    private _enableReferralsProfile = document.getElementById("radio-referrals-use-profile") as HTMLInputElement;
    private _enableReferralsInvite = document.getElementById("radio-referrals-use-invite") as HTMLInputElement;
    private _sendPWR = document.getElementById("accounts-send-pwr") as HTMLSpanElement;
    private _profileSelect = document.getElementById("modify-user-profiles") as HTMLSelectElement;
    private _userSelect = document.getElementById("modify-user-users") as HTMLSelectElement;
    private _referralsProfileSelect = document.getElementById("enable-referrals-user-profiles") as HTMLSelectElement;
    private _referralsInviteSelect = document.getElementById("enable-referrals-user-invites") as HTMLSelectElement;
    private _referralsExpiry = document.getElementById("enable-referrals-user-expiry") as HTMLInputElement;

    private _applyHomescreen = document.getElementById("modify-user-homescreen") as HTMLInputElement;
    private _applyConfiguration = document.getElementById("modify-user-configuration") as HTMLInputElement;
    private _applyOmbi = document.getElementById("modify-user-ombi") as HTMLInputElement;
    private _applyJellyseerr = document.getElementById("modify-user-jellyseerr") as HTMLInputElement;

    private _selectAll = document.getElementById("accounts-select-all") as HTMLInputElement;
    // private _users: { [id: string]: user };
    // private _ordering: string[] = [];
    get users(): { [id: string]: user } { return this._search.items as { [id: string]: user }; }
    // set users(v: { [id: string]: user }) { this._search.items = v as SearchableItems; }

    // Whether the enable/disable button should enable or not.
    private _shouldEnable = false;

    private _addUserForm = document.getElementById("form-add-user") as HTMLFormElement;
    private _addUserName = this._addUserForm.querySelector("input[type=text]") as HTMLInputElement;
    private _addUserEmail = this._addUserForm.querySelector("input[type=email]") as HTMLInputElement;
    private _addUserPassword = this._addUserForm.querySelector("input[type=password]") as HTMLInputElement;
    private _addUserProfile = this._addUserForm.querySelector("select") as HTMLSelectElement;
    
    // Columns for sorting.
    private _columns: { [className: string]: Column } = {};

    // Whether the "Extend expiry" is extending or setting an expiry.
    private _settingExpiry = false;

    private _sortingByButton = document.getElementById("accounts-sort-by-field") as HTMLButtonElement;

    private _maxDayHourMinuteOptions = 30;
    private _populateNumbers = () => {
        const fieldIDs = ["months", "days", "hours", "minutes"];
        const prefixes = ["extend-expiry-"];
        for (let i = 0; i < fieldIDs.length; i++) {
            for (let j = 0; j < prefixes.length; j++) { 
                const field = document.getElementById(prefixes[j] + fieldIDs[i]);
                field.textContent = '';
                for (let n = 0; n <= this._maxDayHourMinuteOptions; n++) {
                   const opt = document.createElement("option") as HTMLOptionElement;
                   opt.textContent = ""+n;
                   opt.value = ""+n;
                   field.appendChild(opt);
                }
            }
        }
    }
    
    constructor() {
        super({
            loader: document.getElementById("accounts-loader"),
            loadMoreButton: document.getElementById("accounts-load-more") as HTMLButtonElement,
            loadAllButton: document.getElementById("accounts-load-all") as HTMLButtonElement,
            refreshButton: document.getElementById("accounts-refresh") as HTMLButtonElement,
            filterArea: document.getElementById("accounts-filter-area"),
            searchOptionsHeader: document.getElementById("accounts-search-options-header"),
            searchBox: document.getElementById("accounts-search") as HTMLInputElement,
            recordCounter: document.getElementById("accounts-record-counter"),
            totalEndpoint: "/users/count",
            getPageEndpoint: "/users",
            itemsPerPage: 40,
            maxItemsLoadedForSearch: 200,
            appendNewItems: (resp: paginatedDTO) => {
                for (let u of ((resp as UsersDTO).users || [])) {
                    if (u.id in this.users) {
                        this.users[u.id].update(u);
                    } else {
                        this.add(u);
                    }
                }

                this._search.setOrdering(
                    this._columns[this._search.sortField].sort(this.users),
                    this._search.sortField,
                    this._search.ascending
                );
            },
            replaceWithNewItems: (resp: paginatedDTO) => {
                let accountsOnDOM: { [id: string]: boolean } = {};

                for (let id of Object.keys(this.users)) { accountsOnDOM[id] = true; }
                for (let u of ((resp as UsersDTO).users || [])) {
                    if (u.id in accountsOnDOM) {
                        this.users[u.id].update(u);
                        delete accountsOnDOM[u.id];
                    } else {
                        this.add(u);
                    }
                }

                // Delete accounts w/ remaining IDs (those not in resp.users)
                // console.log("Removing", Object.keys(accountsOnDOM).length, "from DOM");
                for (let id in accountsOnDOM) {
                    this.users[id].remove()
                    delete this.users[id];
                }

                this._search.setOrdering(
                    this._columns[this._search.sortField].sort(this.users),
                    this._search.sortField,
                    this._search.ascending
                );
            },
            defaultSortField: USER_DEFAULT_SORT_FIELD,
            defaultSortAscending: USER_DEFAULT_SORT_ASCENDING,
            pageLoadCallback: (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                // FIXME: Error message
                if (req.status != 200) return;
            }
        });
        this._populateNumbers();
        
        // FIXME: Remove!
        (window as any).acc = this;

        let searchConfig: SearchConfiguration = {
            filterArea: this._c.filterArea,
            sortingByButton: this._sortingByButton,
            searchOptionsHeader: this._c.searchOptionsHeader,
            notFoundPanel: document.getElementById("accounts-not-found"),
            notFoundLocallyText: document.getElementById("accounts-no-local-results"),
            filterList: document.getElementById("accounts-filter-list"),
            search: this._c.searchBox,
            queries: queries(),
            setVisibility: null,
            clearSearchButtonSelector: ".accounts-search-clear",
            serverSearchButtonSelector: ".accounts-search-server",
            onSearchCallback: (_0: boolean, _1: boolean) => {
                this._checkCheckCount();
            },
            searchServer: null,
            clearServerSearch: null,
        };
        
        this.initSearch(searchConfig);
        
        this._selectAll.checked = false;
        this._selectAll.onchange = () => {
            this.selectAll = this._selectAll.checked;
        };
        document.addEventListener("accounts-reload", () => this.reload());
        document.addEventListener("accountCheckEvent", () => { this._counter.selected++; this._checkCheckCount(); });
        document.addEventListener("accountUncheckEvent", () => { this._counter.selected--; this._checkCheckCount(); });
        this._addUserButton.onclick = () => {
            this._populateAddUserProfiles();
            window.modals.addUser.toggle();
        };
        this._addUserForm.addEventListener("submit", this._addUser);

        this._deleteNotify.onchange = () => {
            if (this._deleteNotify.checked) {
                this._deleteReason.classList.remove("unfocused");
            } else {
                this._deleteReason.classList.add("unfocused");
            }
        };
        this._modifySettings.onclick = this.modifyUsers;
        this._modifySettings.classList.add("unfocused");

        if (window.ombiEnabled)
            this._applyOmbi.parentElement.classList.remove("unfocused");
        else
            this._applyOmbi.parentElement.classList.add("unfocused");
        if (window.jellyseerrEnabled)
            this._applyJellyseerr.parentElement.classList.remove("unfocused");
        else
            this._applyJellyseerr.parentElement.classList.add("unfocused");

        const checkSource = () => {
            const profileSpan = this._modifySettingsProfile.nextElementSibling as HTMLSpanElement;
            const userSpan = this._modifySettingsUser.nextElementSibling as HTMLSpanElement;
            if (this._modifySettingsProfile.checked) {
                this._userSelect.parentElement.classList.add("unfocused");
                this._profileSelect.parentElement.classList.remove("unfocused")
                profileSpan.classList.add("@high");
                profileSpan.classList.remove("@low");
                userSpan.classList.remove("@high");
                userSpan.classList.add("@low");
                this._applyOmbi.parentElement.classList.remove("unfocused");
                this._applyJellyseerr.parentElement.classList.remove("unfocused");
            } else {
                this._userSelect.parentElement.classList.remove("unfocused");
                this._profileSelect.parentElement.classList.add("unfocused");
                userSpan.classList.add("@high");
                userSpan.classList.remove("@low");
                profileSpan.classList.remove("@high");
                profileSpan.classList.add("@low");
                this._applyOmbi.parentElement.classList.add("unfocused");
                this._applyJellyseerr.parentElement.classList.add("unfocused");
            }
        };
        this._modifySettingsProfile.onchange = checkSource;
        this._modifySettingsUser.onchange = checkSource;

        if (window.referralsEnabled) {
            const profileSpan = this._enableReferralsProfile.nextElementSibling as HTMLSpanElement;
            const inviteSpan = this._enableReferralsInvite.nextElementSibling as HTMLSpanElement;
            const checkReferralSource = () => {
                console.debug("States:", this._enableReferralsProfile.checked, this._enableReferralsInvite.checked);
                if (this._enableReferralsProfile.checked) {
                    this._referralsInviteSelect.parentElement.classList.add("unfocused");
                    this._referralsProfileSelect.parentElement.classList.remove("unfocused")
                    profileSpan.classList.add("@high");
                    profileSpan.classList.remove("@low");
                    inviteSpan.classList.remove("@high");
                    inviteSpan.classList.add("@low");
                } else {
                    this._referralsInviteSelect.parentElement.classList.remove("unfocused");
                    this._referralsProfileSelect.parentElement.classList.add("unfocused");
                    inviteSpan.classList.add("@high");
                    inviteSpan.classList.remove("@low");
                    profileSpan.classList.remove("@high");
                    profileSpan.classList.add("@low");
                }
            };
            profileSpan.onclick = () => {
                this._enableReferralsProfile.checked = true;
                this._enableReferralsInvite.checked = false;
                checkReferralSource();
            };
            inviteSpan.onclick = () => {;
                this._enableReferralsInvite.checked = true;
                this._enableReferralsProfile.checked = false;
                checkReferralSource();
            };
            this._enableReferrals.onclick = () => {
                this.enableReferrals();
                profileSpan.onclick(null);
            };
        }

        this._deleteUser.onclick = this.deleteUsers;
        this._deleteUser.classList.add("unfocused");

        this._announceButton.onclick = this.announce;
        this._announceButton.parentElement.classList.add("unfocused");

        this._extendExpiry.onclick = () => { this.extendExpiry(); };
        this._removeExpiry.onclick = () => { this.removeExpiry(); };
        this._expiryDropdown.classList.add("unfocused");
        this._extendExpiryDate.classList.add("unfocused");

        this._extendExpiryTextInput.onkeyup = () => {
            this._extendExpiryTextInput.parentElement.parentElement.classList.remove("opacity-60");
            this._extendExpiryFieldInputs.classList.add("opacity-60");
                this._usingExtendExpiryTextInput = true;
            this._displayExpiryDate();
        }

        this._extendExpiryTextInput.onclick = () => {
            this._extendExpiryTextInput.parentElement.parentElement.classList.remove("opacity-60");
            this._extendExpiryFieldInputs.classList.add("opacity-60");
            this._usingExtendExpiryTextInput = true;
            this._displayExpiryDate();
        };

        this._extendExpiryFieldInputs.onclick = () => {
            this._extendExpiryFieldInputs.classList.remove("opacity-60");
            this._extendExpiryTextInput.parentElement.parentElement.classList.add("opacity-60");
            this._usingExtendExpiryTextInput = false;
            this._displayExpiryDate();
        };
        
        for (let field of ["months", "days", "hours", "minutes"]) {
            (document.getElementById("extend-expiry-"+field) as HTMLSelectElement).onchange = () => {
                this._extendExpiryFieldInputs.classList.remove("opacity-60");
                this._extendExpiryTextInput.parentElement.parentElement.classList.add("opacity-60");
                this._usingExtendExpiryTextInput = false;
                this._displayExpiryDate();
            };
        }

        this._disableEnable.onclick = this.enableDisableUsers;
        this._disableEnable.parentElement.classList.add("unfocused");

        this._enableExpiry.onclick = () => { this.extendExpiry(true); };
        this._enableExpiryNotify.onchange = () => {
            if (this._enableExpiryNotify.checked) {
                this._enableExpiryReason.classList.remove("unfocused");
            } else {
                this._enableExpiryReason.classList.add("unfocused");
            }
        };

        if (!window.usernameEnabled) {
            this._addUserName.classList.add("unfocused");
            this._addUserName = this._addUserEmail;
        }

        if (!window.linkResetEnabled) {
            this._sendPWR.classList.add("unfocused");
        } else {
            this._sendPWR.onclick = this.sendPWR;
        }
        /*if (!window.emailEnabled) {
            this._deleteNotify.parentElement.classList.add("unfocused");
            this._deleteNotify.checked = false;
        }*/

        this._announceTextarea.onkeyup = this.loadPreview;
        addDiscord = newDiscordSearch(window.lang.strings("linkDiscord"), window.lang.strings("searchDiscordUser"), window.lang.strings("add"), (user: DiscordUser, id: string) => { 
            _post("/users/discord", {jf_id: id, discord_id: user.id}, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    document.dispatchEvent(new CustomEvent("accounts-reload"));
                    if (req.status != 200) {
                        window.notifications.customError("errorConnectDiscord", window.lang.notif("errorFailureCheckLogs"));
                        return
                    }
                    window.notifications.customSuccess("discordConnected", window.lang.notif("accountConnected"));
                    window.modals.discord.close()
                }
            });
        });

        this._announceSaveButton.onclick = this.saveAnnouncement;
        const announceVarUsername = document.getElementById("announce-variables-username") as HTMLSpanElement;
        announceVarUsername.onclick = () => {
            insertText(this._announceTextarea, announceVarUsername.children[0].textContent);
            this.loadPreview();
        };

        const headerNames: string[] = ["username", "access-jfa", "email", "telegram", "matrix", "discord", "expiry", "last-active", "referrals"];
        const headerGetters: string[] = ["name", "accounts_admin", "email", "telegram", "matrix", "discord", "expiry", "last_active", "referrals_enabled"];
        for (let i = 0; i < headerNames.length; i++) {
            const header: HTMLTableCellElement = document.querySelector(".accounts-header-" + headerNames[i]) as HTMLTableCellElement;
            if (header !== null) {
                this._columns[headerGetters[i]] = new Column(header, headerGetters[i], Object.getOwnPropertyDescriptor(user.prototype, headerGetters[i]).get);
            }
        }

        // Start off sorting by username (this._c.defaultSortField)
        const defaultSort = () => {
            document.dispatchEvent(new CustomEvent("header-click", { detail: this._c.defaultSortField }));
            this._columns[this._c.defaultSortField].ascending = this._c.defaultSortAscending;
            this._columns[this._c.defaultSortField].hideIcon();
            this._sortingByButton.classList.add("hidden");
            this._search.showHideSearchOptionsHeader();
        };

        this._sortingByButton.addEventListener("click", defaultSort);

        document.addEventListener("header-click", (event: CustomEvent) => {
            this._search.setOrdering(
                this._columns[event.detail].sort(this.users),
                event.detail,
                this._columns[event.detail].ascending
            );
            this._sortingByButton.replaceChildren(this._columns[event.detail].asElement());
            this._sortingByButton.classList.remove("hidden");
            // console.log("ordering by", event.detail, ": ", this._ordering);
            if (this._search.inSearch) {
                this._search.onSearchBoxChange();
            } else {
                this.setVisibility(this._search.ordering, true);
                this._search.setNotFoundPanelVisibility(false);
            }
            this._search.inServerSearch = false;
            this.autoSetServerSearchButtonsDisabled();
            this._search.showHideSearchOptionsHeader();
        });

        defaultSort();

        this._search.showHideSearchOptionsHeader();

        this.registerURLListener();
    }

    reload = (callback?: (resp: paginatedDTO) => void) => {
        this._reload(callback);
        this.loadTemplates();
    }

    loadMore = (loadAll: boolean = false, callback?: () => void) => {
        this._loadMore(
            loadAll,
            callback
        );
    };

    get selectAll(): boolean { return this._selectAll.checked; }
    set selectAll(state: boolean) {
        let count = 0;
        for (let id in this.users) {
            if (this._container.contains(this.users[id].asElement())) { // Only select visible elements
                this.users[id].selected = state;
                count++;
            }
        }
        this._selectAll.checked = state;
        this._selectAll.indeterminate = false;
        state ? this._counter.selected = count : 0;
    }
    
    selectAllBetweenIDs = (startID: string, endID: string) => {
        let inRange = false;
        for (let id of this._search.ordering) {
            if (!(inRange || id == startID)) continue;
            inRange = true;
            if (!(this._container.contains(this.users[id].asElement()))) continue;
            this.users[id].selected = true;
            if (id == endID) return;
        }
    }

    add = (u: User) => {
        let domAccount = new user(u);
        this.users[u.id] = domAccount;
        // console.log("after appending lengths:", Object.keys(this.users).length, Object.keys(this._search.items).length);
    }

    private _checkCheckCount = () => {
        const list = this._collectUsers();
        this._counter.selected = list.length;
        if (this._counter.selected == 0) {
            this._selectAll.indeterminate = false;
            this._selectAll.checked = false;
            this._modifySettings.classList.add("unfocused");
            if (window.referralsEnabled) {
                this._enableReferrals.classList.add("unfocused");
            }
            this._deleteUser.classList.add("unfocused");
            if (window.emailEnabled || window.telegramEnabled) {
                this._announceButton.parentElement.classList.add("unfocused");
            }
            this._expiryDropdown.classList.add("unfocused");
            this._disableEnable.parentElement.classList.add("unfocused");
            this._sendPWR.classList.add("unfocused");
        } else {
            let visibleCount = 0;
            for (let id in this.users) {
                if (this._container.contains(this.users[id].asElement())) {
                    visibleCount++;
                }
            }
            if (this._counter.selected == visibleCount) {
                this._selectAll.checked = true;
                this._selectAll.indeterminate = false;
            } else {
                this._selectAll.checked = false;
                this._selectAll.indeterminate = true;
            }
            this._modifySettings.classList.remove("unfocused");
            if (window.referralsEnabled) {
                this._enableReferrals.classList.remove("unfocused");
            }
            this._deleteUser.classList.remove("unfocused");
            this._deleteUser.textContent = window.lang.quantity("deleteUser", list.length);
            if (window.emailEnabled || window.telegramEnabled) {
                this._announceButton.parentElement.classList.remove("unfocused");
            }
            let anyNonExpiries = list.length == 0 ? true : false;
            let allNonExpiries = true;
            let noContactCount = 0;
            let referralState = Number(this.users[list[0]].referrals_enabled); // -1 = hide, 0 = show "enable", 1 = show "disable"
            // Only show enable/disable button if all selected have the same state.
            this._shouldEnable = this.users[list[0]].disabled
            let showDisableEnable = true;
            for (let id of list) {
                if (!anyNonExpiries && !this.users[id].expiry) {
                    anyNonExpiries = true;
                    this._expiryDropdown.classList.add("unfocused");
                }
                if (this.users[id].expiry) {
                    allNonExpiries = false;
                }
                if (showDisableEnable && this.users[id].disabled != this._shouldEnable) {
                    showDisableEnable = false;
                    this._disableEnable.parentElement.classList.add("unfocused");
                }
                if (!showDisableEnable && anyNonExpiries) { break; }
                if (!this.users[id].lastNotifyMethod()) {
                    noContactCount++;
                }
                if (window.referralsEnabled && referralState != -1 && Number(this.users[id].referrals_enabled) != referralState) {
                    referralState = -1;
                }
            }
            this._settingExpiry = false;
            if (!anyNonExpiries && !allNonExpiries) {
                this._expiryDropdown.classList.remove("unfocused");
                this._extendExpiry.textContent = window.lang.strings("extendExpiry");
                this._removeExpiry.classList.remove("unfocused");
            }
            if (allNonExpiries) {
                this._expiryDropdown.classList.remove("unfocused");
                this._extendExpiry.textContent = window.lang.strings("setExpiry");
                this._settingExpiry = true;
                this._removeExpiry.classList.add("unfocused");
            }
            // Only show "Send PWR" if a maximum of 1 user selected doesn't have a contact method
            if (noContactCount > 1) {
                this._sendPWR.classList.add("unfocused");
            } else if (window.linkResetEnabled) {
                this._sendPWR.classList.remove("unfocused");
            }
            if (showDisableEnable) {
                let message: string;
                if (this._shouldEnable) {
                    this._disableEnable.parentElement.classList.remove("manual");
                    message = window.lang.strings("reEnable");
                    this._disableEnable.classList.add("~positive");
                    this._disableEnable.classList.remove("~warning");
                } else {
                    this._disableEnable.parentElement.classList.add("manual");
                    message = window.lang.strings("disable");
                    this._disableEnable.classList.add("~warning");
                    this._disableEnable.classList.remove("~positive");
                }
                this._disableEnable.parentElement.classList.remove("unfocused");
                this._disableEnable.textContent = message;
            }
            if (window.referralsEnabled) {
                if (referralState == -1) {
                    this._enableReferrals.classList.add("unfocused");
                }  else {
                    this._enableReferrals.classList.remove("unfocused");
                }
                if (referralState == 0) {
                    this._enableReferrals.classList.add("~urge");
                    this._enableReferrals.classList.remove("~warning");
                    this._enableReferrals.textContent = window.lang.strings("enableReferrals");
                } else if (referralState == 1) {
                    this._enableReferrals.classList.add("~warning");
                    this._enableReferrals.classList.remove("~urge");
                    this._enableReferrals.textContent = window.lang.strings("disableReferrals");
                }
            }
        }
    }
    
    private _collectUsers = (): string[] => {
        let list: string[] = [];
        for (let id in this.users) {
            if (this._container.contains(this.users[id].asElement()) && this.users[id].selected) { list.push(id); }
        }
        return list;
    }

    private _addUser = (event: Event) => {
        event.preventDefault();
        const button = this._addUserForm.querySelector("span.submit") as HTMLSpanElement;
        const send = {
            "username": this._addUserName.value,
            "email": this._addUserEmail.value,
            "password": this._addUserPassword.value,
            "profile": this._addUserProfile.value,
        };
        for (let field in send) {
            if (!send[field]) {
                window.notifications.customError("addUserBlankField", window.lang.notif("errorBlankFields"));
                return;
            }
        }
        toggleLoader(button);
        _post("/user", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                toggleLoader(button);
                if (req.status == 200 || (req.response["user"] as boolean)) {
                    window.notifications.customSuccess("addUser", window.lang.var("notifications", "userCreated", `"${send['username']}"`));
                    if (!req.response["email"]) {
                        window.notifications.customError("sendWelcome", window.lang.notif("errorSendWelcomeEmail"));
                        console.error("User created, but welcome email failed");
                    }
                } else {
                    let msg = window.lang.var("notifications", "errorUserCreated", `"${send['username']}"`);
                    if ("error" in req.response) {
                        let realError = window.lang.notif(req.response["error"]);
                        if (realError) msg = realError;
                    }
                    window.notifications.customError("addUser", msg);
                }
                if (req.response["error"] as String) {
                    console.error(req.response["error"]);
                }

                this.reload();
                window.modals.addUser.close();
            }
        }, true);
    }
    loadPreview = () => {
        let content = this._announceTextarea.value;
        if (!this._previewLoaded) {
            content = stripMarkdown(content);
            this._announcePreview.textContent = content;
        } else {
            content = Marked.parse(content);
            this._announcePreview.innerHTML = content;
        }
    }
    saveAnnouncement = (event: Event) => {
        event.preventDefault();
        const form = document.getElementById("form-announce") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        if (this._announceNameLabel.classList.contains("unfocused")) {
            this._announceNameLabel.classList.remove("unfocused");
            form.onsubmit = this.saveAnnouncement;
            button.textContent = window.lang.get("strings", "saveAsTemplate");
            this._announceSaveButton.classList.add("unfocused");
            const details = document.getElementById("announce-details");
            details.classList.add("unfocused");
            return;
        }
        const name = (this._announceNameLabel.querySelector("input") as HTMLInputElement).value;
        if (!name) { return; }
        const subject = document.getElementById("announce-subject") as HTMLInputElement;
        let send: announcementTemplate = {
            name: name,
            subject: subject.value,
            message: this._announceTextarea.value
        }
        _post("/users/announce/template", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                this.reload();
                toggleLoader(button);
                window.modals.announce.close();
                if (req.status != 200 && req.status != 204) {
                    window.notifications.customError("announcementError", window.lang.notif("errorFailureCheckLogs"));
                } else {
                    window.notifications.customSuccess("announcementSuccess", window.lang.notif("savedAnnouncement"));
                }
            }
        });
    }
    announce = (event?: Event, template?: announcementTemplate) => {
        const modalHeader = document.getElementById("header-announce");
        modalHeader.textContent = window.lang.quantity("announceTo", this._collectUsers().length);
        const form = document.getElementById("form-announce") as HTMLFormElement;
        let list = this._collectUsers();
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        removeLoader(button);
        button.textContent = window.lang.get("strings", "send");
        const details = document.getElementById("announce-details");
        details.classList.remove("unfocused");
        this._announceSaveButton.classList.remove("unfocused");
        const subject = document.getElementById("announce-subject") as HTMLInputElement;
        this._announceNameLabel.classList.add("unfocused");
        if (template) {
            subject.value = template.subject;
            this._announceTextarea.value = template.message;
        } else {
            subject.value = "";
            this._announceTextarea.value = "";
        }
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            let send = {
                "users": list,
                "subject": subject.value,
                "message": this._announceTextarea.value
            }
            _post("/users/announce", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    window.modals.announce.close();
                    if (req.status != 200 && req.status != 204) {
                        window.notifications.customError("announcementError", window.lang.notif("errorFailureCheckLogs"));
                    } else {
                        window.notifications.customSuccess("announcementSuccess", window.lang.notif("sentAnnouncement"));
                    }
                }
            });
        };
        _get("/config/emails/Announcement", null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                const preview = document.getElementById("announce-preview") as HTMLDivElement;
                if (req.status != 200) {
                    preview.innerHTML = `<pre class="preview-content" class="font-mono bg-inherit"></pre>`;
                    window.modals.announce.show();
                    this._previewLoaded = false;
                    return;
                }
                    
                let templ = req.response as templateEmail;
                if (!templ.html) {
                    preview.innerHTML = `<pre class="preview-content" class="font-mono bg-inherit"></pre>`;
                    this._previewLoaded = false;
                } else {
                    preview.innerHTML = templ.html;
                    this._previewLoaded = true;
                }
                this._announcePreview = preview.getElementsByClassName("preview-content")[0] as HTMLElement;
                this.loadPreview();
                window.modals.announce.show();
            }
        });
    }
    loadTemplates = () => _get("/users/announce", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                this._announceButton.nextElementSibling.children[0].classList.add("unfocused");
                return;
            }
            this._announceButton.nextElementSibling.children[0].classList.remove("unfocused");
            const list = req.response["announcements"] as string[];
            if (list.length == 0) {
                this._announceButton.nextElementSibling.children[0].classList.add("unfocused");
                return;
            }
            if (list.length > 0) {
                this._announceButton.innerHTML = `${window.lang.strings("announce")} <i class="ml-2 ri-arrow-drop-down-line"></i>`;
            }
            const dList = document.getElementById("accounts-announce-templates") as HTMLDivElement;
            dList.textContent = '';
            for (let name of list) {
                const el = document.createElement("div") as HTMLDivElement;
                el.classList.add("flex", "flex-row", "justify-between", "truncate", "mt-2");
                el.innerHTML = `
                <span class="button ~neutral sm full-width accounts-announce-template-button">${name}</span><span class="button ~critical fr ml-4 accounts-announce-template-delete">&times;</span>
                `;
                let urlSafeName = encodeURIComponent(encodeURIComponent(name));
                (el.querySelector("span.accounts-announce-template-button") as HTMLSpanElement).onclick = () => {
                    _get("/users/announce/" + urlSafeName, null, (req: XMLHttpRequest) => {
                        if (req.readyState == 4) {
                            let template: announcementTemplate;
                            if (req.status != 200) {
                                window.notifications.customError("getTemplateError", window.lang.notif("errorFailureCheckLogs"));
                            } else {
                                template = req.response;
                            }
                            this.announce(null, template);
                        }
                    });
                };
                (el.querySelector("span.accounts-announce-template-delete") as HTMLSpanElement).onclick = () => {
                    _delete("/users/announce/" + urlSafeName, null, (req: XMLHttpRequest) => {
                        if (req.readyState == 4) {
                            if (req.status != 200) {
                                window.notifications.customError("deleteTemplateError", window.lang.notif("errorFailureCheckLogs"));
                            }
                            this.reload();
                        }
                    });
                };
                dList.appendChild(el);
            }
        }
    });

    private _enableDisableUsers = (users: string[], enable: boolean, notify: boolean, reason: string|null, post: (req: XMLHttpRequest) => void) => {
        let send = {
            "users": users,
            "enabled": enable,
            "notify": notify
        };
        if (reason) send["reason"] = reason;
        _post("/users/enable", send, post, true);
    };

    enableDisableUsers = () => {
        // We can share the delete modal for this
        const modalHeader = document.getElementById("header-delete-user");
        const form = document.getElementById("form-delete-user") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        let list = this._collectUsers();
        if (this._shouldEnable) {
            modalHeader.textContent = window.lang.quantity("reEnableUsers", list.length);
            button.textContent = window.lang.strings("reEnable");
            button.classList.add("~urge");
            button.classList.remove("~critical");
        } else {
            modalHeader.textContent = window.lang.quantity("disableUsers", list.length);
            button.textContent = window.lang.strings("disable");
            button.classList.add("~critical");
            button.classList.remove("~urge");
        }
        this._deleteNotify.checked = false;
        this._deleteReason.value = "";
        this._deleteReason.classList.add("unfocused");
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            this._enableDisableUsers(list, this._shouldEnable, this._deleteNotify.checked, this._deleteNotify ? this._deleteReason.value : null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    window.modals.deleteUser.close();
                    if (req.status != 200 && req.status != 204) {
                        let errorMsg = window.lang.notif("errorFailureCheckLogs");
                        if (!("error" in req.response)) {
                            errorMsg = window.lang.notif("errorPartialFailureCheckLogs");
                        }
                        window.notifications.customError("deleteUserError", errorMsg);
                    } else if (this._shouldEnable) {
                        window.notifications.customSuccess("enableUserSuccess", window.lang.quantity("enabledUser", list.length));
                    } else {
                        window.notifications.customSuccess("disableUserSuccess", window.lang.quantity("disabledUser", list.length));
                    }
                    this.reload();
                }
            });
        }
        window.modals.deleteUser.show();
    }

    deleteUsers = () => {
        const modalHeader = document.getElementById("header-delete-user");
        let list = this._collectUsers();
        modalHeader.textContent = window.lang.quantity("deleteNUsers", list.length);
        const form = document.getElementById("form-delete-user") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        button.textContent = window.lang.strings("delete");
        button.classList.add("~critical");
        button.classList.remove("~urge");
        this._deleteNotify.checked = false;
        this._deleteReason.value = "";
        this._deleteReason.classList.add("unfocused");
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            let send = {
                "users": list,
                "notify": this._deleteNotify.checked,
                "reason": this._deleteNotify ? this._deleteReason.value : ""
            };
            _delete("/users", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    window.modals.deleteUser.close();
                    if (req.status != 200 && req.status != 204) {
                        let errorMsg = window.lang.notif("errorFailureCheckLogs");
                        if (!("error" in req.response)) {
                            errorMsg = window.lang.notif("errorPartialFailureCheckLogs");
                        }
                        window.notifications.customError("deleteUserError", errorMsg);
                    } else {
                        window.notifications.customSuccess("deleteUserSuccess", window.lang.quantity("deletedUser", list.length));
                    }
                    this.reload();
                }
            });
        };
        window.modals.deleteUser.show();
    }
    
    sendPWR = () => {
        addLoader(this._sendPWR);
        let list = this._collectUsers();
        let manualUser: user;
        for (let id of list) {
            let user = this.users[id];
            if (!user.lastNotifyMethod() && !user.email) {
                manualUser  = user;
                break;
            }
        }
        const messageBox = document.getElementById("send-pwr-note") as HTMLParagraphElement;
        let message: string;
        let send = {
            users: list
        };
        _post("/users/password-reset", send, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._sendPWR);
            let link: string;
            if (req.status == 200) {
                link = req.response["link"];
                if (req.response["manual"] as boolean) {
                    message = window.lang.var("strings", "sendPWRManual", manualUser.name);
                } else {
                    message = window.lang.strings("sendPWRSuccess") + " " + window.lang.strings("sendPWRSuccessManual");
                }
            } else if (req.status == 204) {
                    message = window.lang.strings("sendPWRSuccess");
            } else {
                window.notifications.customError("errorSendPWR", window.lang.strings("errorFailureCheckLogs"));
                return;
            }
            message += " " + window.lang.strings("sendPWRValidFor");
            messageBox.textContent = message;
            let linkButton = document.getElementById("send-pwr-link") as HTMLSpanElement;
            if (link) {
                linkButton.classList.remove("unfocused");
                linkButton.onclick = () => {
                    toClipboard(link);
                    linkButton.textContent = window.lang.strings("copied");
                    linkButton.classList.add("~positive");
                    linkButton.classList.remove("~urge");
                    setTimeout(() => {
                        linkButton.textContent = window.lang.strings("copy");
                        linkButton.classList.add("~urge");
                        linkButton.classList.remove("~positive");
                    }, 800);
                };
            } else {
                linkButton.classList.add("unfocused");
            }
            window.modals.sendPWR.show();
        }, true);
    }

    modifyUsers = () => {
        const modalHeader = document.getElementById("header-modify-user");
        modalHeader.textContent = window.lang.quantity("modifySettingsFor", this._collectUsers().length)
        let list = this._collectUsers();

        (() => {
            let innerHTML = "";
            for (const profile of window.availableProfiles) {
                innerHTML += `<option value="${profile}">${profile}</option>`;
            }
            this._profileSelect.innerHTML = innerHTML;
        })();

        (() => {
            let innerHTML = "";
            for (let id in this.users) {
                innerHTML += `<option value="${id}">${this.users[id].name}</option>`;
            }
            this._userSelect.innerHTML = innerHTML;
        })();

        const form = document.getElementById("form-modify-user") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;

        this._modifySettingsProfile.checked = true;
        this._modifySettingsUser.checked = false;
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            let send = {
                "apply_to": list,
                "homescreen": this._applyHomescreen.checked,
                "configuration": this._applyConfiguration.checked,
                "ombi": this._applyOmbi.checked,
                "jellyseerr": this._applyJellyseerr.checked
            };
            if (this._modifySettingsProfile.checked && !this._modifySettingsUser.checked) { 
                send["from"] = "profile";
                send["profile"] = this._profileSelect.value;
            } else if (this._modifySettingsUser.checked && !this._modifySettingsProfile.checked) {
                send["from"] = "user";
                send["id"] = this._userSelect.value;
            }
            _post("/users/settings", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    if (req.status == 500) {
                        let response = JSON.parse(req.response);
                        let errorMsg = "";
                        if ("homescreen" in response && "policy" in response) {
                            const homescreen = Object.keys(response["homescreen"]).length;
                            const policy = Object.keys(response["policy"]).length;
                            if (homescreen != 0 && policy == 0) {
                                errorMsg = window.lang.notif("errorSettingsAppliedNoHomescreenLayout");
                            } else if (policy != 0 && homescreen == 0) {
                                errorMsg = window.lang.notif("errorHomescreenAppliedNoSettings");
                            } else if (policy != 0 && homescreen != 0) {
                                errorMsg = window.lang.notif("errorSettingsFailed");
                            }
                        } else if ("error" in response) {
                            errorMsg = response["error"];
                        }
                        window.notifications.customError("modifySettingsError", errorMsg);
                    } else if (req.status == 200 || req.status == 204) {
                        window.notifications.customSuccess("modifySettingsSuccess", window.lang.quantity("appliedSettings", this._collectUsers().length));
                    }
                    this.reload();
                    window.modals.modifyUser.close();
                }
            });
        };
        window.modals.modifyUser.show();
    }
    
    enableReferrals = () => {
        const modalHeader = document.getElementById("header-enable-referrals-user");
        modalHeader.textContent = window.lang.quantity("enableReferralsFor", this._collectUsers().length)
        let list = this._collectUsers();

        // Check if we're disabling or enabling
        if (this.users[list[0]].referrals_enabled) {
            _delete("/users/referral", {"users": list}, (req: XMLHttpRequest) => {
                if (req.readyState != 4 || req.status != 200) return;
                window.notifications.customSuccess("disabledReferralsSuccess", window.lang.quantity("appliedSettings", list.length));
                this.reload();
            });
            return;
        }
            
        (() => {
            _get("/invites", null, (req: XMLHttpRequest) => {
                if (req.readyState != 4 || req.status != 200) return;

                // 1. Invites

                let innerHTML = "";
                let invites = req.response["invites"] as Array<Invite>;
                if (invites) {
                    for (let inv of invites) {
                        let name = inv.code;
                        if (inv.label) {
                            name = `${inv.label} (${inv.code})`;
                        }
                        innerHTML += `<option value="${inv.code}">${name}</option>`;
                    }
                    this._enableReferralsInvite.checked = true;
                } else {
                    this._enableReferralsInvite.checked = false;
                    innerHTML += `<option>${window.lang.strings("inviteNoInvites")}</option>`;
                }
                this._enableReferralsProfile.checked = !(this._enableReferralsInvite.checked);
                this._referralsInviteSelect.innerHTML = innerHTML;
            
                // 2. Profiles

                innerHTML = "";
                for (const profile of window.availableProfiles) {
                    innerHTML += `<option value="${profile}">${profile}</option>`;
                }
                this._referralsProfileSelect.innerHTML = innerHTML;
            });
        })();

        const form = document.getElementById("form-enable-referrals-user") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            let send = {
                "users": list
            };
            // console.log("profile:", this._enableReferralsProfile.checked, this._enableReferralsInvite.checked); 
            if (this._enableReferralsProfile.checked && !this._enableReferralsInvite.checked) { 
                send["from"] = "profile";
                send["profile"] = this._referralsProfileSelect.value;
            } else if (this._enableReferralsInvite.checked && !this._enableReferralsProfile.checked) {
                send["from"] = "invite";
                send["id"] = this._referralsInviteSelect.value;
            }
            _post("/users/referral/" + send["from"] + "/" + (send["id"] ? send["id"] : send["profile"]) + "/" + (this._referralsExpiry.checked ? "with-expiry" : "none"), send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    if (req.status == 400) {
                        window.notifications.customError("noReferralTemplateError", window.lang.notif("errorNoReferralTemplate"));
                    } else if (req.status == 200 || req.status == 204) {
                        window.notifications.customSuccess("enableReferralsSuccess", window.lang.quantity("appliedSettings", list.length));
                    }
                    this.reload();
                    window.modals.enableReferralsUser.close();
                }
            });
        };
        this._enableReferralsProfile.checked = true;
        this._enableReferralsInvite.checked = false;
        this._referralsExpiry.checked = false;
        window.modals.enableReferralsUser.show();
    }

    removeExpiry = () => {
        const list = this._collectUsers();

        let success = true;
        for (let id of list) {
            _delete("/users/" + id + "/expiry", null, (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                if (req.status != 200) {
                    success = false;
                    return;
                }
            });
            if (!success) break;
        }

        if (success) {
            window.notifications.customSuccess("modifySettingsSuccess", window.lang.quantity("appliedSettings", list.length));
        } else {
            window.notifications.customError("modifySettingsError", window.lang.notif("errorSettingsFailed"));
        }
        this.reload();
    }

    _displayExpiryDate = () => {
        let date: Date;
        let invalid = false;
        let users = this._collectUsers();
        if (this._usingExtendExpiryTextInput) {
            date = (Date as any).fromString(this._extendExpiryTextInput.value) as Date;
            invalid = "invalid" in (date as any);
        } else {
            let fields: Array<HTMLSelectElement> = [
                document.getElementById("extend-expiry-months") as HTMLSelectElement,
                document.getElementById("extend-expiry-days") as HTMLSelectElement,
                document.getElementById("extend-expiry-hours") as HTMLSelectElement,
                document.getElementById("extend-expiry-minutes") as HTMLSelectElement
            ];
            invalid = fields[0].value == "0" && fields[1].value == "0" && fields[2].value == "0" && fields[3].value == "0";
            let id = users.length > 0 ? users[0] : "";
            if (!id) invalid = true;
            else {
                date = new Date(this.users[id].expiry*1000);
                if (this.users[id].expiry == 0) date = new Date();
                date.setMonth(date.getMonth() + (+fields[0].value))
                date.setDate(date.getDate() + (+fields[1].value));
                date.setHours(date.getHours() + (+fields[2].value));
                date.setMinutes(date.getMinutes() + (+fields[3].value));
            }
        }
        const submit = this._extendExpiryForm.querySelector(`input[type="submit"]`) as HTMLInputElement;
        const submitSpan = submit.nextElementSibling;
        if (invalid) {
            submit.disabled = true;
            submitSpan.classList.add("opacity-60");
            this._extendExpiryDate.classList.add("unfocused");
        } else {
            submit.disabled = false;
            submitSpan.classList.remove("opacity-60");
            this._extendExpiryDate.innerHTML = `
            <div class="flex flex-col">
                <span>${window.lang.strings("accountWillExpire").replace("{date}", toDateString(date))}</span>
                ${users.length > 1 ? "<span>"+window.lang.strings("expirationBasedOn")+"</span>" : ""}
            </div>
            `;
            this._extendExpiryDate.classList.remove("unfocused");
        }
    }

    extendExpiry = (enableUser?: boolean) => {
        const list = this._collectUsers();
        let applyList: string[] = [];
        for (let id of list) {
            applyList.push(id);
        }
        this._enableExpiryReason.classList.add("unfocused");
        this._enableExpiryNotify.parentElement.classList.remove("unfocused");
        this._enableExpiryNotify.checked = false;
        this._enableExpiryReason.value = "";
        let header: string;
        if (enableUser) {
            header = window.lang.quantity("reEnableUsers", list.length);
        } else if (this._settingExpiry) {
            header = window.lang.quantity("setExpiry", list.length);
            // this._enableExpiryNotify.parentElement.classList.add("unfocused");
        } else {
            header = window.lang.quantity("extendExpiry", applyList.length);
            // this._enableExpiryNotify.parentElement.classList.add("unfocused");
        }
        document.getElementById("header-extend-expiry").textContent = header;
        const extend = () => {
            let send = { "users": applyList, "timestamp": 0, "notify": this._enableExpiryNotify.checked }
            if (this._enableExpiryNotify.checked) {
                send["reason"] = this._enableExpiryReason.value;
            }
            if (this._usingExtendExpiryTextInput) {
                let date = (Date as any).fromString(this._extendExpiryTextInput.value) as Date;
                send["timestamp"] = Math.floor(date.getTime() / 1000);
                if ("invalid" in (date as any)) {
                    window.notifications.customError("extendExpiryError", window.lang.notif("errorInvalidDate"));
                    return;
                }
            } else {
                for (let field of ["months", "days", "hours", "minutes"]) {
                    send[field] = +(document.getElementById("extend-expiry-"+field) as HTMLSelectElement).value;
                }
            }

            _post("/users/extend", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status != 200 && req.status != 204) {
                        window.notifications.customError("extendExpiryError", window.lang.notif("errorFailureCheckLogs"));
                    } else {
                        window.notifications.customSuccess("extendExpiry", window.lang.quantity("extendedExpiry", applyList.length));
                    }
                    window.modals.extendExpiry.close()
                    this.reload();
                }
            });
        };
        this._extendExpiryForm.onsubmit = (event: Event) => {
            event.preventDefault();
            if (enableUser) {
                this._enableDisableUsers(applyList, true, this._enableExpiryNotify.checked, this._enableExpiryNotify.checked ? this._enableExpiryReason.value : null, (req: XMLHttpRequest) => {
                    if (req.readyState == 4) {
                        if (req.status != 200 && req.status != 204) {
                            window.modals.extendExpiry.close();
                            let errorMsg = window.lang.notif("errorFailureCheckLogs");
                            if (!("error" in req.response)) {
                                errorMsg = window.lang.notif("errorPartialFailureCheckLogs");
                            }
                            window.notifications.customError("deleteUserError", errorMsg);
                            return;
                        }
                        extend();
                    }
                });
            } else {
                extend();
            }
        }
        this._extendExpiryTextInput.value = "";
        this._usingExtendExpiryTextInput = false;
        this._extendExpiryDate.classList.add("unfocused");
        this._displayExpiryDate();
        window.modals.extendExpiry.show();
    }
   
    private _populateAddUserProfiles = () => {
        this._addUserProfile.textContent = "";
        let innerHTML = `<option value="none">${window.lang.strings("inviteNoProfile")}</option>`;
        for (let i = 0; i < window.availableProfiles.length; i++) {
            innerHTML += `<option value="${window.availableProfiles[i]}" ${i == 0 ? "selected" : ""}>${window.availableProfiles[i]}</option>`;
        }
        this._addUserProfile.innerHTML = innerHTML;
    }

    focusAccount = (userID: string) => {
        console.debug("focusing user", userID);
        this._c.searchBox.value = `id:"${userID}"`;
        this._search.onSearchBoxChange();
        if (userID in this.users) this.users[userID].focus();
    }

    public static readonly _accountURLEvent = "account-url";
    registerURLListener = () => document.addEventListener(accountsList._accountURLEvent, (event: CustomEvent) => {
        this.focusAccount(event.detail);
    });

    isAccountURL = () => {
        const urlParams = new URLSearchParams(window.location.search);
        const userID = urlParams.get("user");
        return Boolean(userID);
    }

    loadAccountURL = () => {
        const urlParams = new URLSearchParams(window.location.search);
        const userID = urlParams.get("user");
        this.focusAccount(userID);
    }

}

export const accountURLEvent = (id: string) => { return new CustomEvent(accountsList._accountURLEvent, {"detail": id}) };

type GetterReturnType = Boolean | boolean | String | Number | number;
type Getter = () => GetterReturnType;

// When a column is clicked, it broadcasts it's name and ordering to be picked up and stored by accountsList
// When list is refreshed, accountList calls method of the specific Column and re-orders accordingly.
// Listen for broadcast event from others, check its not us by comparing the header className in the message, then hide the arrow icon
class Column {
    private _header: HTMLTableCellElement;
    private _card: HTMLElement;
    private _cardSortingByIcon: HTMLElement;
    private _name: string;
    private _headerContent: string;
    private _getter: Getter;
    private _ascending: boolean;
    private _active: boolean;

    constructor(header: HTMLTableCellElement, name: string, getter: Getter) {
        this._header = header;
        this._name = name;
        this._headerContent = this._header.textContent;
        this._getter = getter;
        this._ascending = true;
        this._active = false;

        this._header.addEventListener("click", () => {
            // If we are the active sort column, a click means to switch between ascending/descending.
            if (this._active) {
                this.ascending = !this.ascending;
                return;
            }
            this._active = true;
            this._header.setAttribute("aria-sort", this._headerContent);
            this.updateHeader();
            document.dispatchEvent(new CustomEvent("header-click", { detail: this._name }));
        });
        document.addEventListener("header-click", (event: CustomEvent) => {
            if (event.detail != this._name) {
                this._active = false;
                this._header.removeAttribute("aria-sort");
                this.hideIcon();
            }
        });

        this._card = document.createElement("button");
        this._card.classList.add("button", "~neutral", "@low", "center", "flex", "flex-row", "gap-1");
        this._card.innerHTML = `
        <i class="sorting-by-direction ri-arrow-up-s-line"></i>
        <span class="font-bold">${window.lang.strings("sortingBy")}: </span><span>${this._headerContent}</span>
        <i class="ri-close-line text-2xl"></i>
        `;
        this._cardSortingByIcon = this._card.querySelector(".sorting-by-direction");
    }

    hideIcon = () => {
        this._header.textContent = this._headerContent;
    }

    updateHeader = () => {
        this._header.innerHTML = `
        <span class="">${this._headerContent}</span>
        <i class="ri-arrow-${this._ascending? "up" : "down"}-s-line" aria-hidden="true"></i>
        `;
    }

    asElement = () => { return this._card };

    get ascending() { return this._ascending; }
    set ascending(v: boolean) {
        this._ascending = v;
        if (v) {
            this._cardSortingByIcon.classList.add("ri-arrow-up-s-line");
            this._cardSortingByIcon.classList.remove("ri-arrow-down-s-line");
        } else {
            this._cardSortingByIcon.classList.add("ri-arrow-down-s-line");
            this._cardSortingByIcon.classList.remove("ri-arrow-up-s-line");
        }
        if (!this._active) return;
        this.updateHeader();
        this._header.setAttribute("aria-sort", this._headerContent);
        document.dispatchEvent(new CustomEvent("header-click", { detail: this._name }));
    }

    // Sorts the user list. previouslyActive is whether this column was previously sorted by, indicating that the direction should change.
    sort = (users: { [id: string]: user }): string[] => {
        let userIDs = Object.keys(users);
        userIDs.sort((a: string, b: string): number => {
            let av: GetterReturnType = this._getter.call(users[a]);
            if (typeof av === "string") av = av.toLowerCase();
            let bv: GetterReturnType = this._getter.call(users[b]);
            if (typeof bv === "string") bv = bv.toLowerCase();
            if (av < bv) return this._ascending ? -1 : 1;
            if (av > bv) return this._ascending ? 1 : -1;
            return 0;
        });

        return userIDs;
    }
}
