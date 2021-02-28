import { _get, _post, _delete, toggleLoader } from "../modules/common.js";

interface User {
    id: string;
    name: string;
    email: string | undefined;
    last_active: string;
    admin: boolean;
    disabled: boolean;
    expiry: string;
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
    private _expiry: HTMLTableDataCellElement;
    private _lastActive: HTMLTableDataCellElement;
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
    
    get expiry(): string { return this._expiry.textContent; }
    set expiry(value: string) { this._expiry.textContent = value; }

    get last_active(): string { return this._lastActive.textContent; }
    set last_active(value: string) { this._lastActive.textContent = value; }

    private _checkEvent = new CustomEvent("accountCheckEvent");
    private _uncheckEvent = new CustomEvent("accountUncheckEvent");

    constructor(user: User) {
        this._row = document.createElement("tr") as HTMLTableRowElement;
        this._row.innerHTML = `
            <td><input type="checkbox" value=""></td>
            <td><span class="accounts-username"></span> <span class="accounts-admin"></span> <span class="accounts-disabled"></span></td>
            <td><i class="icon ri-edit-line accounts-email-edit"></i><span class="accounts-email-container ml-half"></span></td>
            <td class="accounts-expiry"></td>
            <td class="accounts-last-active"></td>
        `;
        const emailEditor = `<input type="email" class="input ~neutral !normal stealth-input">`;
        this._check = this._row.querySelector("input[type=checkbox]") as HTMLInputElement;
        this._username = this._row.querySelector(".accounts-username") as HTMLSpanElement;
        this._admin = this._row.querySelector(".accounts-admin") as HTMLSpanElement;
        this._disabled = this._row.querySelector(".accounts-disabled") as HTMLSpanElement;
        this._email = this._row.querySelector(".accounts-email-container") as HTMLInputElement;
        this._emailEditButton = this._row.querySelector(".accounts-email-edit") as HTMLElement;
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

    update = (user: User) => {
        this.id = user.id;
        this.name = user.name;
        this.email = user.email || "";
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
    private _deleteUser = document.getElementById("accounts-delete-user") as HTMLSpanElement;
    private _deleteNotify = document.getElementById("delete-user-notify") as HTMLInputElement;
    private _deleteReason = document.getElementById("textarea-delete-user") as HTMLTextAreaElement;
    private _extendExpiry = document.getElementById("accounts-extend-expiry") as HTMLSpanElement;
    private _modifySettings = document.getElementById("accounts-modify-user") as HTMLSpanElement;
    private _modifySettingsProfile = document.getElementById("radio-use-profile") as HTMLInputElement;
    private _modifySettingsUser = document.getElementById("radio-use-user") as HTMLInputElement;
    private _profileSelect = document.getElementById("modify-user-profiles") as HTMLSelectElement;
    private _userSelect = document.getElementById("modify-user-users") as HTMLSelectElement;

    private _selectAll = document.getElementById("accounts-select-all") as HTMLInputElement;
    private _users: { [id: string]: user };
    private _checkCount: number = 0;

    private _addUserForm = document.getElementById("form-add-user") as HTMLFormElement;
    private _addUserName = this._addUserForm.querySelector("input[type=text]") as HTMLInputElement;
    private _addUserEmail = this._addUserForm.querySelector("input[type=email]") as HTMLInputElement;
    private _addUserPassword = this._addUserForm.querySelector("input[type=password]") as HTMLInputElement;
    
    private _count = 30;
    private _populateNumbers = () => {
        const fieldIDs = ["days", "hours", "minutes"];
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

    get selectAll(): boolean { return this._selectAll.checked; }
    set selectAll(state: boolean) { 
        for (let id in this._users) {
            this._users[id].selected = state;
        }
        this._selectAll.checked = state;
        this._selectAll.indeterminate = false;
        state ? this._checkCount = Object.keys(this._users).length : 0;

    }
    
    add = (u: User) => {
        let domAccount = new user(u);
        this._users[u.id] = domAccount;
        this._table.appendChild(domAccount.asElement());
    }

    private _checkCheckCount = () => {
        if (this._checkCount == 0) {
            this._selectAll.indeterminate = false;
            this._selectAll.checked = false;
            this._modifySettings.classList.add("unfocused");
            this._deleteUser.classList.add("unfocused");
            if (window.emailEnabled) {
                this._announceButton.classList.add("unfocused");
            }
            this._extendExpiry.classList.add("unfocused");
        } else {
            if (this._checkCount == Object.keys(this._users).length) {
                this._selectAll.checked = true;
                this._selectAll.indeterminate = false;
            } else {
                this._selectAll.checked = false;
                this._selectAll.indeterminate = true;
            }
            this._modifySettings.classList.remove("unfocused");
            this._deleteUser.classList.remove("unfocused");
            this._deleteUser.textContent = window.lang.quantity("deleteUser", this._checkCount);
            if (window.emailEnabled) {
                this._announceButton.classList.remove("unfocused");
            }
            const list = this._collectUsers();
            let anyNonExpiries = false;
            for (let id of list) {
                if (!this._users[id].expiry) {
                    anyNonExpiries = true;
                    this._extendExpiry.classList.add("unfocused");
                    break;
                }
            }
            if (!anyNonExpiries) {
                this._extendExpiry.classList.remove("unfocused");
            }
        }
    }
    
    private _collectUsers = (): string[] => {
        let list: string[] = [];
        for (let id in this._users) {
            if (this._users[id].selected) { list.push(id); }
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
    
    announce = () => {
        const modalHeader = document.getElementById("header-announce");
        modalHeader.textContent = window.lang.quantity("announceTo", this._checkCount);
        const form = document.getElementById("form-announce") as HTMLFormElement;
        let list = this._collectUsers();
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        const subject = document.getElementById("announce-subject") as HTMLInputElement;
        const message = document.getElementById("textarea-announce") as HTMLTextAreaElement;
        subject.value = "";
        message.value = "";
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);
            let send = {
                "users": list,
                "subject": subject.value,
                "message": message.value
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
        window.modals.announce.show();
    }

    deleteUsers = () => {
        const modalHeader = document.getElementById("header-delete-user");
        modalHeader.textContent = window.lang.quantity("deleteNUsers", this._checkCount);
        let list = this._collectUsers();
        const form = document.getElementById("form-delete-user") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
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
                        window.notifications.customSuccess("deleteUserSuccess", window.lang.quantity("deletedUser", this._checkCount));
                    }
                    this.reload();
                }
            });
        };
        window.modals.deleteUser.show();
    }

    modifyUsers = () => {
        const modalHeader = document.getElementById("header-modify-user");
        modalHeader.textContent = window.lang.quantity("modifySettingsFor", this._checkCount)
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
                        window.notifications.customSuccess("modifySettingsSuccess", window.lang.quantity("appliedSettings", this._checkCount));
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
            for (let field of ["days", "hours", "minutes"]) {
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
        this._selectAll.onchange = () => { this.selectAll = this._selectAll.checked };
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

        if (!window.usernameEnabled) {
            this._addUserName.classList.add("unfocused");
            this._addUserName = this._addUserEmail;
        }
        /*if (!window.emailEnabled) {
            this._deleteNotify.parentElement.classList.add("unfocused");
            this._deleteNotify.checked = false;
        }*/
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
            this._checkCheckCount;
        }
    })
}
