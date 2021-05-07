import { _get, _post, _delete, toggleLoader, addLoader, removeLoader, toDateString } from "../modules/common.js";
import { templateEmail } from "../modules/settings.js";
import { Marked } from "@ts-stack/markdown";
import { stripMarkdown } from "../modules/stripmd.js";

interface User {
    id: string;
    name: string;
    email: string | undefined;
    last_active: number;
    admin: boolean;
    disabled: boolean;
    expiry: number;
    telegram: string;
}

interface getPinResponse {
    token: string;
    username: string;
}

class user implements User {
    private _row: HTMLTableRowElement;
    private _check: HTMLInputElement;
    private _username: HTMLSpanElement;
    private _admin: HTMLSpanElement;
    private _disabled: HTMLSpanElement;
    private _email: HTMLInputElement;
    private _emailAddress: string;
    private _emailEditButton: HTMLElement;
    private _telegram: HTMLTableDataCellElement;
    private _telegramUsername: string;
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
    
    get telegram(): string { return this._telegramUsername; }
    set telegram(u: string) {
        if (!window.telegramEnabled) return;
        this._telegramUsername = u;
        if (u == "") {
            this._telegram.innerHTML = `<span class="chip btn !low">Add</span>`;
            (this._telegram.querySelector("span") as HTMLSpanElement).onclick = this._addTelegram;
        } else {
            this._telegram.innerHTML = `<a href="https://t.me/${u}" target="_blank">@${u}</a>`;
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
            <td><span class="accounts-username"></span> <span class="accounts-admin"></span> <span class="accounts-disabled"></span></td>
            <td><i class="icon ri-edit-line accounts-email-edit"></i><span class="accounts-email-container ml-half"></span></td>
        `;
        if (window.telegramEnabled) {
            innerHTML += `
            <td class="accounts-telegram"></td>
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
        this.last_active = user.last_active;
        this.admin = user.admin;
        this.disabled = user.disabled;
        this.expiry = user.expiry;
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
