import { _get, _post, _delete, toggleLoader, addLoader, removeLoader, toDateString } from "../modules/common.js";
import { templateEmail } from "../modules/settings.js";
import { Marked } from "@ts-stack/markdown";
import { stripMarkdown } from "../modules/stripmd.js";

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
}

interface getPinResponse {
    token: string;
    username: string;
}

interface DiscordUser {
    name: string;
    avatar_url: string;
    id: string;
}

class user implements User {
    private _row: HTMLTableRowElement;
    private _check: HTMLInputElement;
    private _username: HTMLSpanElement;
    private _admin: HTMLSpanElement;
    private _disabled: HTMLSpanElement;
    private _email: HTMLInputElement;
    private _notifyEmail: boolean;
    private _emailAddress: string;
    private _emailEditButton: HTMLElement;
    private _telegram: HTMLTableDataCellElement;
    private _telegramUsername: string;
    private _notifyTelegram: boolean;
    private _discord: HTMLTableDataCellElement;
    private _discordUsername: string;
    private _discordID: string;
    private _notifyDiscord: boolean;
    private _expiry: HTMLTableDataCellElement;
    private _expiryUnix: number;
    private _lastActive: HTMLTableDataCellElement;
    private _lastActiveUnix: number;
    id: string;
    private _selected: boolean;

    get selected(): boolean { return this._selected; }
    set selected(state: boolean) {
        this._selected = state;
        this._check.checked = state;
        state ? document.dispatchEvent(this._checkEvent) : document.dispatchEvent(this._uncheckEvent);
    }

    get name(): string { return this._username.textContent; }
    set name(value: string) { this._username.textContent = value; }

    get admin(): boolean { return this._admin.classList.contains("chip"); }
    set admin(state: boolean) {
        if (state) {
            this._admin.classList.add("chip", "~info", "ml-1");
            this._admin.textContent = window.lang.strings("admin");
        } else {
            this._admin.classList.remove("chip", "~info", "ml-1");
            this._admin.textContent = "";
        }
    }

    get disabled(): boolean { return this._disabled.classList.contains("chip"); }
    set disabled(state: boolean) {
        if (state) {
            this._disabled.classList.add("chip", "~warning", "ml-1");
            this._disabled.textContent = window.lang.strings("disabled");
        } else {
            this._disabled.classList.remove("chip", "~warning", "ml-1");
            this._disabled.textContent = "";
        }
    }

    get email(): string { return this._emailAddress; }
    set email(value: string) {
        this._emailAddress = value;
        const input = this._email.querySelector("input");
        if (input) {
            input.value = value;
        } else {
            this._email.textContent = value;
        }
    }

    get notify_email(): boolean { return this._notifyEmail; }
    set notify_email(s: boolean) {
        this._notifyEmail = s;
        if (window.telegramEnabled && this._telegramUsername != "") {
            const email = this._telegram.getElementsByClassName("accounts-contact-email")[0] as HTMLInputElement;
            if (email) {
                email.checked = s;
            }
        }
        if (window.discordEnabled && this._discordUsername != "") {
            const email = this._discord.getElementsByClassName("accounts-contact-email")[0] as HTMLInputElement;
            email.checked = s;
        }
    }
    
    get telegram(): string { return this._telegramUsername; }
    set telegram(u: string) {
        if (!window.telegramEnabled) return;
        this._telegramUsername = u;
        if (u == "") {
            this._telegram.innerHTML = `<span class="chip btn !low">Add</span>`;
            (this._telegram.querySelector("span") as HTMLSpanElement).onclick = this._addTelegram;
        } else {
            let innerHTML = `
            <a href="https://t.me/${u}" target="_blank">@${u}</a>
            `;
            if (!window.discordEnabled || this._discordUsername == "") {
                innerHTML += `
                <div class="table-inline">
                    <i class="icon ri-settings-2-line ml-half dropdown-button"></i>
                    <div class="dropdown manual">
                        <div class="dropdown-display lg">
                            <div class="card ~neutral !low">
                                <span class="supra sm">${window.lang.strings("contactThrough")}</span>
                                <label class="row switch pb-1 mt-half">
                                    <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-email">
                                    <span>Email</span>
                                </label>
                                <label class="row switch pb-1">
                                    <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-telegram">
                                    <span>Telegram</span>
                                </label>
                            </div>
                        </div>
                    </div>
                </div>
                `;
            }
            this._telegram.innerHTML = innerHTML;
            if (!window.discordEnabled || this._discordUsername == "") {
                // Javascript is necessary as including the button inside the dropdown would make it too wide to display next to the username.
                const button = this._telegram.querySelector("i");
                const dropdown = this._telegram.querySelector("div.dropdown") as HTMLDivElement;
                const checks = this._telegram.querySelectorAll("input") as NodeListOf<HTMLInputElement>;
                for (let i = 0; i < checks.length; i++) {
                    checks[i].onclick = () => this._setNotifyMethod("telegram");
                }

                button.onclick = () => {
                    dropdown.classList.add("selected");
                    document.addEventListener("click", outerClickListener);
                };
                const outerClickListener = (event: Event) => {
                    if (!(event.target instanceof HTMLElement && (this._telegram.contains(event.target) || button.contains(event.target)))) {
                        dropdown.classList.remove("selected");
                        document.removeEventListener("click", outerClickListener);
                    }
                };
            }
        }
    }
    
    get notify_telegram(): boolean { return this._notifyTelegram; }
    set notify_telegram(s: boolean) {
        if (!window.telegramEnabled || !this._telegramUsername) return;
        this._notifyTelegram = s;
        const telegram = this._telegram.getElementsByClassName("accounts-contact-telegram")[0] as HTMLInputElement;
        if (telegram) {
            telegram.checked = s;
        }
        if (window.discordEnabled && this._discordUsername != "") {
            const telegram = this._discord.getElementsByClassName("accounts-contact-telegram")[0] as HTMLInputElement;
            telegram.checked = s;
        }
    }

    private _setNotifyMethod = (mode: string = "telegram") => {
        let el: HTMLElement;
        if (mode == "telegram") { el = this._telegram }
        else if (mode == "discord") { el = this._discord }
        const email = el.getElementsByClassName("accounts-contact-email")[0] as HTMLInputElement;
        let send = {
            id: this.id,
            email: email.checked
        }
        if (window.telegramEnabled && this._telegramUsername != "") {
            const telegram = el.getElementsByClassName("accounts-contact-telegram")[0] as HTMLInputElement;
            send["telegram"] = telegram.checked;
        }
        if (window.discordEnabled && this._discordUsername != "") {
            const discord = el.getElementsByClassName("accounts-contact-discord")[0] as HTMLInputElement;
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
        if (!window.discordEnabled) return;
        this._discordUsername = u;
        if (u == "") {
            this._discord.innerHTML = `<span class="chip btn !low">Add</span>`;
            (this._discord.querySelector("span") as HTMLSpanElement).onclick = this._addDiscord;
        } else {
            let innerHTML = `
            <div class="table-inline">
                <a href="https://discord.com/users/${this._discordID}" class="discord-link" target="_blank">${u}</a>
                <i class="icon ri-settings-2-line ml-half dropdown-button"></i>
                <div class="dropdown manual">
                    <div class="dropdown-display lg">
                        <div class="card ~neutral !low">
                            <span class="supra sm">${window.lang.strings("contactThrough")}</span>
                            <label class="row switch pb-1 mt-half">
                                <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-email">
                                <span>Email</span>
                            </label>
                            <label class="row switch pb-1">
                                <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-discord">
                                <span>Discord</span>
                            </label>
            `;
            if (window.telegramEnabled && this._telegramUsername != "") {
                innerHTML += `
                            <label class="row switch pb-1">
                                <input type="checkbox" name="accounts-contact-${this.id}" class="accounts-contact-telegram">
                                <span>Telegram</span>
                            </label>
                `;
            }
            innerHTML += `
                        </div>
                    </div>
                </div>
            </div>
            `;
            this._discord.innerHTML = innerHTML;
            // Javascript is necessary as including the button inside the dropdown would make it too wide to display next to the username.
            const button = this._discord.querySelector("i");
            const dropdown = this._discord.querySelector("div.dropdown") as HTMLDivElement;
            const checks = this._discord.querySelectorAll("input") as NodeListOf<HTMLInputElement>;
            for (let i = 0; i < checks.length; i++) {
                checks[i].onclick = () => this._setNotifyMethod("discord");
            }

            button.onclick = () => {
                dropdown.classList.add("selected");
                document.addEventListener("click", outerClickListener);
            };
            const outerClickListener = (event: Event) => {
                if (!(event.target instanceof HTMLElement && (this._discord.contains(event.target) || button.contains(event.target)))) {
                    dropdown.classList.remove("selected");
                    document.removeEventListener("click", outerClickListener);
                }
            };
        }
    }

    get discord_id(): string { return this._discordID; }
    set discord_id(id: string) {
        this._discordID = id;
        const link = this._discord.getElementsByClassName("discord-link")[0] as HTMLAnchorElement;
        link.href = `https://discord.com/users/${id}`;
    }
    
    get notify_discord(): boolean { return this._notifyDiscord; }
    set notify_discord(s: boolean) {
        if (!window.discordEnabled || !this._discordUsername) return;
        this._notifyDiscord = s;
        const discord = this._discord.getElementsByClassName("accounts-contact-discord")[0] as HTMLInputElement;
        discord.checked = s;
        if (window.telegramEnabled && this._telegramUsername != "") {
            const discord = this._discord.getElementsByClassName("accounts-contact-discord")[0] as HTMLInputElement;
            discord.checked = s;
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

    private _checkEvent = new CustomEvent("accountCheckEvent");
    private _uncheckEvent = new CustomEvent("accountUncheckEvent");

    constructor(user: User) {
        this._row = document.createElement("tr") as HTMLTableRowElement;
        let innerHTML = `
            <td><input type="checkbox" value=""></td>
            <td><div class="table-inline"><span class="accounts-username"></span> <span class="accounts-admin"></span> <span class="accounts-disabled"></span></span></td>
            <td><div class="table-inline"><i class="icon ri-edit-line accounts-email-edit"></i><span class="accounts-email-container ml-half"></span></div></td>
        `;
        if (window.telegramEnabled) {
            innerHTML += `
            <td class="accounts-telegram"></td>
            `;
        }
        if (window.discordEnabled) {
            innerHTML += `
            <td class="accounts-discord"></td>
            `;
        }
        innerHTML += `
        <td class="accounts-expiry"></td>
        <td class="accounts-last-active"></td>
        `;
        this._row.innerHTML = innerHTML;
        const emailEditor = `<input type="email" class="input ~neutral !normal stealth-input">`;
        this._check = this._row.querySelector("input[type=checkbox]") as HTMLInputElement;
        this._username = this._row.querySelector(".accounts-username") as HTMLSpanElement;
        this._admin = this._row.querySelector(".accounts-admin") as HTMLSpanElement;
        this._disabled = this._row.querySelector(".accounts-disabled") as HTMLSpanElement;
        this._email = this._row.querySelector(".accounts-email-container") as HTMLInputElement;
        this._emailEditButton = this._row.querySelector(".accounts-email-edit") as HTMLElement;
        this._telegram = this._row.querySelector(".accounts-telegram") as HTMLTableDataCellElement;
        this._discord = this._row.querySelector(".accounts-discord") as HTMLTableDataCellElement;
        this._expiry = this._row.querySelector(".accounts-expiry") as HTMLTableDataCellElement;
        this._lastActive = this._row.querySelector(".accounts-last-active") as HTMLTableDataCellElement;
        this._check.onchange = () => { this.selected = this._check.checked; }

        const toggleStealthInput = () => {
            if (this._emailEditButton.classList.contains("ri-edit-line")) {
                this._email.innerHTML = emailEditor;
                this._email.querySelector("input").value = this._emailAddress;
                this._email.classList.remove("ml-half");
            } else {
                this._email.textContent = this._emailAddress;
                this._email.classList.add("ml-half");
            }
            this._emailEditButton.classList.toggle("ri-check-line");
            this._emailEditButton.classList.toggle("ri-edit-line");
        };
        const outerClickListener = (event: Event) => {
            if (!(event.target instanceof HTMLElement && (this._email.contains(event.target) || this._emailEditButton.contains(event.target)))) {
                toggleStealthInput();
                this.email = this.email;
                document.removeEventListener("click", outerClickListener);
            }
        };
        this._emailEditButton.onclick = () => {
            if (this._emailEditButton.classList.contains("ri-edit-line")) {
                document.addEventListener('click', outerClickListener);
            } else {
                this._updateEmail();
                document.removeEventListener('click', outerClickListener);
            }
            toggleStealthInput();
        };

        this.update(user);
        
        document.addEventListener("timefmt-change", () => {
            this.expiry = this.expiry;
            this.last_active = this.last_active;
        });
    }

    private _updateEmail = () => {
        let oldEmail = this.email;
        this.email = this._email.querySelector("input").value;
        let send = {};
        send[this.id] = this.email;
        _post("/users/emails", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200) {
                    window.notifications.customSuccess("emailChanged", window.lang.var("notifications", "changedEmailAddress", `"${this.name}"`));
                } else {
                    this.email = oldEmail;
                    window.notifications.customError("emailChanged", window.lang.var("notifications", "errorChangedEmailAddress", `"${this.name}"`)); 
                }
            }
        });
    }
    
    private _timer: NodeJS.Timer;

    private _discordKbListener = () => {
        clearTimeout(this._timer);
        const list = document.getElementById("discord-list") as HTMLTableElement;
        const input = document.getElementById("discord-search") as HTMLInputElement;
        if (input.value.length < 2) {
            return;
        }
        list.innerHTML = ``;
        addLoader(list);
        list.parentElement.classList.add("mb-1", "mt-1");
        this._timer = setTimeout(() => {
            _get("/users/discord/" + input.value, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status != 200) {
                        removeLoader(list);
                        list.parentElement.classList.remove("mb-1", "mt-1");
                        return;
                    }
                    const users = req.response["users"] as Array<DiscordUser>;
                    let innerHTML = ``;
                    for (let i = 0; i < users.length; i++) {
                        innerHTML += `
                        <tr>
                            <td class="img-circle sm">
                                <img class="img-circle" src="${users[i].avatar_url}" width="32" height="32">
                            </td>
                            <td class="w-100 sm">
                                <p class="content">${users[i].name}</p>
                            </td>
                            <td class="sm">
                                <span id="discord-user-${users[i].id}" class="button ~info !high">${window.lang.strings("add")}</span>
                            </td>
                        </tr>
                        `;
                    }
                    list.innerHTML = innerHTML;
                    removeLoader(list);
                    list.parentElement.classList.remove("mb-1", "mt-1");
                    for (let i = 0; i < users.length; i++) {
                        const button = document.getElementById(`discord-user-${users[i].id}`) as HTMLInputElement;
                        button.onclick = () => _post("/users/discord", {jf_id: this.id, discord_id: users[i].id}, (req: XMLHttpRequest) => {
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
                    }
                }
            });
        }, 750);
    }

    private _addDiscord = () => {
        if (!window.discordEnabled) { return; }
        const input = document.getElementById("discord-search") as HTMLInputElement;
        const list = document.getElementById("discord-list") as HTMLDivElement;
        list.innerHTML = ``;
        input.value = "";
        input.removeEventListener("keyup", this._discordKbListener);
        input.addEventListener("keyup", this._discordKbListener);
        window.modals.discord.show();
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


    update = (user: User) => {
        this.id = user.id;
        this.name = user.name;
        this.email = user.email || "";
        this.telegram = user.telegram;
        this.discord = user.discord;
        this.last_active = user.last_active;
        this.admin = user.admin;
        this.disabled = user.disabled;
        this.expiry = user.expiry;
        this.notify_telegram = user.notify_telegram;
        this.notify_discord = user.notify_discord;
        this.notify_email = user.notify_email;
    }

    asElement = (): HTMLTableRowElement => { return this._row; }
    remove = () => {
        if (this.selected) {
            document.dispatchEvent(this._uncheckEvent);
        }
        this._row.remove(); 
    }
}    

export class accountsList {
    private _table = document.getElementById("accounts-list") as HTMLTableSectionElement;
    
    private _addUserButton = document.getElementById("accounts-add-user") as HTMLSpanElement;
    private _announceButton = document.getElementById("accounts-announce") as HTMLSpanElement;
    private _announcePreview: HTMLElement;
    private _previewLoaded = false;
    private _announceTextarea = document.getElementById("textarea-announce") as HTMLTextAreaElement;
    private _deleteUser = document.getElementById("accounts-delete-user") as HTMLSpanElement;
    private _disableEnable = document.getElementById("accounts-disable-enable") as HTMLSpanElement;
    private _deleteNotify = document.getElementById("delete-user-notify") as HTMLInputElement;
    private _deleteReason = document.getElementById("textarea-delete-user") as HTMLTextAreaElement;
    private _extendExpiry = document.getElementById("accounts-extend-expiry") as HTMLSpanElement;
    private _modifySettings = document.getElementById("accounts-modify-user") as HTMLSpanElement;
    private _modifySettingsProfile = document.getElementById("radio-use-profile") as HTMLInputElement;
    private _modifySettingsUser = document.getElementById("radio-use-user") as HTMLInputElement;
    private _profileSelect = document.getElementById("modify-user-profiles") as HTMLSelectElement;
    private _userSelect = document.getElementById("modify-user-users") as HTMLSelectElement;
    private _search = document.getElementById("accounts-search") as HTMLInputElement;

    private _selectAll = document.getElementById("accounts-select-all") as HTMLInputElement;
    private _users: { [id: string]: user };
    private _sortedByName: string[] = [];
    private _checkCount: number = 0;
    private _inSearch = false;
    // Whether the enable/disable button should enable or not.
    private _shouldEnable = false;

    private _addUserForm = document.getElementById("form-add-user") as HTMLFormElement;
    private _addUserName = this._addUserForm.querySelector("input[type=text]") as HTMLInputElement;
    private _addUserEmail = this._addUserForm.querySelector("input[type=email]") as HTMLInputElement;
    private _addUserPassword = this._addUserForm.querySelector("input[type=password]") as HTMLInputElement;
    
    private _count = 30;
    private _populateNumbers = () => {
        const fieldIDs = ["months", "days", "hours", "minutes"];
        const prefixes = ["extend-expiry-"];
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
    
    search = (query: string): string[] => {
        query = query.toLowerCase()
        let result: string[] = [];
        if (query.includes(":")) {  // Support admin:<true/false> and disabled:<true/false>
            const words = query.split(" ");
            query = "";
            for (let word of words) {
                if (word.includes(":")) {
                    const querySplit = word.split(":")
                    let state = false;
                    if (querySplit[1] == "true") {
                        state = true;
                    }
                    for (let id in this._users) {
                        const user = this._users[id];
                        let attrib: boolean;
                        if (querySplit[0] == "admin") { attrib = user.admin; }
                        else if (querySplit[0] == "disabled") { attrib = user.disabled; }
                        if (attrib == state) { result.push(id); }
                    }
                } else { query += word + " "; }
            }
        }
        if (query == "") { return result; }
        for (let id in this._users) {
            const user = this._users[id];
            if (user.name.toLowerCase().includes(query)) {
                result.push(id);
            } else if (user.email.toLowerCase().includes(query)) {
                result.push(id);
            }
        }
        return result;
    }

    get selectAll(): boolean { return this._selectAll.checked; }
    set selectAll(state: boolean) {
        let count = 0;
        for (let id in this._users) {
            if (this._table.contains(this._users[id].asElement())) { // Only select visible elements
                this._users[id].selected = state;
                count++;
            }
        }
        this._selectAll.checked = state;
        this._selectAll.indeterminate = false;
        state ? this._checkCount = count : 0;
    }
    
    add = (u: User) => {
        let domAccount = new user(u);
        this._users[u.id] = domAccount;
        this.unhide(u.id);
    }

    unhide = (id: string) => {
        const keys = Object.keys(this._users);
        if (keys.length == 0) {
            this._table.appendChild(this._users[id].asElement());
            return;
        }
        this._sortedByName = keys.sort((a, b) => this._users[a].name.localeCompare(this._users[b].name));
        let index = this._sortedByName.indexOf(id)+1;
        if (index == this._sortedByName.length-1) {
            this._table.appendChild(this._users[id].asElement());
            return;
        }
        while (index < this._sortedByName.length) {
            if (this._table.contains(this._users[this._sortedByName[index]].asElement())) {
                this._table.insertBefore(this._users[id].asElement(), this._users[this._sortedByName[index]].asElement());
                return;
            }
            index++;
        }
        this._table.appendChild(this._users[id].asElement());
    }

    hide = (id: string) => {
        const el = this._users[id].asElement();
        if (this._table.contains(el)) {
            this._table.removeChild(this._users[id].asElement());
        }
    }

    private _checkCheckCount = () => {
        const list = this._collectUsers();
        this._checkCount = list.length;
        if (this._checkCount == 0) {
            this._selectAll.indeterminate = false;
            this._selectAll.checked = false;
            this._modifySettings.classList.add("unfocused");
            this._deleteUser.classList.add("unfocused");
            if (window.emailEnabled || window.telegramEnabled) {
                this._announceButton.classList.add("unfocused");
            }
            this._extendExpiry.classList.add("unfocused");
            this._disableEnable.classList.add("unfocused");
        } else {
            let visibleCount = 0;
            for (let id in this._users) {
                if (this._table.contains(this._users[id].asElement())) {
                    visibleCount++;
                }
            }
            if (this._checkCount == visibleCount) {
                this._selectAll.checked = true;
                this._selectAll.indeterminate = false;
            } else {
                this._selectAll.checked = false;
                this._selectAll.indeterminate = true;
            }
            this._modifySettings.classList.remove("unfocused");
            this._deleteUser.classList.remove("unfocused");
            this._deleteUser.textContent = window.lang.quantity("deleteUser", list.length);
            if (window.emailEnabled || window.telegramEnabled) {
                this._announceButton.classList.remove("unfocused");
            }
            let anyNonExpiries = list.length == 0 ? true : false;
            // Only show enable/disable button if all selected have the same state.
            this._shouldEnable = this._users[list[0]].disabled
            let showDisableEnable = true;
            for (let id of list) {
                if (!anyNonExpiries && !this._users[id].expiry) {
                    anyNonExpiries = true;
                    this._extendExpiry.classList.add("unfocused");
                }
                if (showDisableEnable && this._users[id].disabled != this._shouldEnable) {
                    showDisableEnable = false;
                    this._disableEnable.classList.add("unfocused");
                }
                if (!showDisableEnable && anyNonExpiries) { break; }
            }
            if (!anyNonExpiries) {
                this._extendExpiry.classList.remove("unfocused");
            }
            if (showDisableEnable) {
                let message: string;
                if (this._shouldEnable) {
                    message = window.lang.strings("reEnable");
                    this._disableEnable.classList.add("~positive");
                    this._disableEnable.classList.remove("~warning");
                } else {
                    message = window.lang.strings("disable");
                    this._disableEnable.classList.add("~warning");
                    this._disableEnable.classList.remove("~positive");
                }
                this._disableEnable.classList.remove("unfocused");
                this._disableEnable.textContent = message;
            }
        }
    }
    
    private _collectUsers = (): string[] => {
        let list: string[] = [];
        for (let id in this._users) {
            if (this._table.contains(this._users[id].asElement()) && this._users[id].selected) { list.push(id); }
        }
        return list;
    }

    private _addUser = (event: Event) => {
        event.preventDefault();
        const button = this._addUserForm.querySelector("span.submit") as HTMLSpanElement;
        const send = {
            "username": this._addUserName.value,
            "email": this._addUserEmail.value,
            "password": this._addUserPassword.value
        };
        for (let field in send) {
            if (!send[field]) {
                window.notifications.customError("addUserBlankField", window.lang.notif("errorBlankFields"));
                return;
            }
        }
        toggleLoader(button);
        _post("/users", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                toggleLoader(button);
                if (req.status == 200 || (req.response["user"] as boolean)) {
                    window.notifications.customSuccess("addUser", window.lang.var("notifications", "userCreated", `"${send['username']}"`));
                    if (!req.response["email"]) {
                        window.notifications.customError("sendWelcome", window.lang.notif("errorSendWelcomeEmail"));
                        console.log("User created, but welcome email failed");
                    }
                } else {
                    window.notifications.customError("addUser", window.lang.var("notifications", "errorUserCreated", `"${send['username']}"`));
                }
                if (req.response["error"] as String) {
                    console.log(req.response["error"]);
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
    announce = () => {
        const modalHeader = document.getElementById("header-announce");
        modalHeader.textContent = window.lang.quantity("announceTo", this._collectUsers().length);
        const form = document.getElementById("form-announce") as HTMLFormElement;
        let list = this._collectUsers();
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        const subject = document.getElementById("announce-subject") as HTMLInputElement;

        subject.value = "";
        this._announceTextarea.value = "";
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
                    preview.innerHTML = `<pre class="preview-content" class="monospace"></pre>`;
                    window.modals.announce.show();
                    this._previewLoaded = false;
                    return;
                }
                    
                let templ = req.response as templateEmail;
                if (!templ.html) {
                    preview.innerHTML = `<pre class="preview-content" class="monospace"></pre>`;
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
            let send = {
                "users": list,
                "enabled": this._shouldEnable,
                "notify": this._deleteNotify.checked,
                "reason": this._deleteNotify ? this._deleteReason.value : ""
            };
            _post("/users/enable", send, (req: XMLHttpRequest) => {
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
            }, true);
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
            for (let id in this._users) {
                innerHTML += `<option value="${id}">${this._users[id].name}</option>`;
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
                "homescreen": (document.getElementById("modify-user-homescreen") as HTMLInputElement).checked
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

    extendExpiry = () => {
        const list = this._collectUsers();
        let applyList: string[] = [];
        for (let id of list) {
            if (this._users[id].expiry) {
                applyList.push(id);
            }
        }
        document.getElementById("header-extend-expiry").textContent = window.lang.quantity("extendExpiry", applyList.length);
        const form = document.getElementById("form-extend-expiry") as HTMLFormElement;
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            let send = { "users": applyList }
            for (let field of ["months", "days", "hours", "minutes"]) {
                send[field] = +(document.getElementById("extend-expiry-"+field) as HTMLSelectElement).value;
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
        }
        window.modals.extendExpiry.show();
    }

    constructor() {
        this._populateNumbers();
        this._users = {};
        this._selectAll.checked = false;
        this._selectAll.onchange = () => {
            this.selectAll = this._selectAll.checked;
        };
        document.addEventListener("accounts-reload", this.reload);
        document.addEventListener("accountCheckEvent", () => { this._checkCount++; this._checkCheckCount(); });
        document.addEventListener("accountUncheckEvent", () => { this._checkCount--; this._checkCheckCount(); });
        this._addUserButton.onclick = window.modals.addUser.toggle;
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
        const checkSource = () => {
            const profileSpan = this._modifySettingsProfile.nextElementSibling as HTMLSpanElement;
            const userSpan = this._modifySettingsUser.nextElementSibling as HTMLSpanElement;
            if (this._modifySettingsProfile.checked) {
                this._userSelect.parentElement.classList.add("unfocused");
                this._profileSelect.parentElement.classList.remove("unfocused")
                profileSpan.classList.add("!high");
                profileSpan.classList.remove("!normal");
                userSpan.classList.remove("!high");
                userSpan.classList.add("!normal");
            } else {
                this._userSelect.parentElement.classList.remove("unfocused");
                this._profileSelect.parentElement.classList.add("unfocused");
                userSpan.classList.add("!high");
                userSpan.classList.remove("!normal");
                profileSpan.classList.remove("!high");
                profileSpan.classList.add("!normal");
            }
        };
        this._modifySettingsProfile.onchange = checkSource;
        this._modifySettingsUser.onchange = checkSource;

        this._deleteUser.onclick = this.deleteUsers;
        this._deleteUser.classList.add("unfocused");

        this._announceButton.onclick = this.announce;
        this._announceButton.classList.add("unfocused");

        this._extendExpiry.onclick = this.extendExpiry;
        this._extendExpiry.classList.add("unfocused");

        this._disableEnable.onclick = this.enableDisableUsers;
        this._disableEnable.classList.add("unfocused");

        if (!window.usernameEnabled) {
            this._addUserName.classList.add("unfocused");
            this._addUserName = this._addUserEmail;
        }
        /*if (!window.emailEnabled) {
            this._deleteNotify.parentElement.classList.add("unfocused");
            this._deleteNotify.checked = false;
        }*/

        const setVisibility = (users: string[], visible: boolean) => {
            for (let id in this._users) {
                if (users.indexOf(id) != -1) {
                    if (visible) {
                        this.unhide(id);
                    } else {
                        this.hide(id);
                    }
                } else {
                    if (visible) {
                        this.hide(id);
                    } else {
                        this.unhide(id);
                    }
                }
            }
        }

        this._search.oninput = () => {
            const query = this._search.value;
            if (!query) {
                setVisibility(Object.keys(this._users), true);
                this._inSearch = false;
            } else {
                this._inSearch = true;
                setVisibility(this.search(query), true);
            }
            this._checkCheckCount();
        };

        this._announceTextarea.onkeyup = this.loadPreview;
    }

    reload = () => _get("/users", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4 && req.status == 200) {
            // same method as inviteList.reload()
            let accountsOnDOM: { [id: string]: boolean } = {};
            for (let id in this._users) { accountsOnDOM[id] = true; }
            for (let u of (req.response["users"] as User[])) {
                if (u.id in this._users) {
                    this._users[u.id].update(u);
                    delete accountsOnDOM[u.id];
                } else {
                    this.add(u);
                }
            }
            for (let id in accountsOnDOM) {
                this._users[id].remove();
                delete this._users[id];
            }
            this._checkCheckCount();
        }
    })
}
