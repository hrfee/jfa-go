import { _get, _post, _delete, toggleLoader } from "../modules/common.js";

interface Profile {
    admin: boolean;
    libraries: string;
    fromUser: string;
    ombi: boolean;
    referrals_enabled: boolean;
}

class profile implements Profile {
    private _row: HTMLTableRowElement;
    private _name: HTMLElement;
    private _adminChip: HTMLSpanElement;
    private _libraries: HTMLTableDataCellElement;
    private _ombiButton: HTMLSpanElement;
    private _fromUser: HTMLTableDataCellElement;
    private _defaultRadio: HTMLInputElement;
    private _ombi: boolean;
    private _referralsButton: HTMLSpanElement;
    private _referralsEnabled: boolean;

    get name(): string { return this._name.textContent; }
    set name(v: string) { this._name.textContent = v; }

    get admin(): boolean { return this._adminChip.classList.contains("chip"); }
    set admin(state: boolean) {
        if (state) {
            this._adminChip.classList.add("chip", "~info", "ml-2");
            this._adminChip.textContent = "Admin";
        } else {
            this._adminChip.classList.remove("chip", "~info", "ml-2");
            this._adminChip.textContent = "";
        }
    }

    get libraries(): string { return this._libraries.textContent; }
    set libraries(v: string) { this._libraries.textContent = v; }

    get ombi(): boolean { return this._ombi; }
    set ombi(v: boolean) {
        if (!window.ombiEnabled) return;
        this._ombi = v;
        if (v) {
            this._ombiButton.textContent = window.lang.strings("delete");
            this._ombiButton.classList.add("~critical");
            this._ombiButton.classList.remove("~neutral");
        } else {
            this._ombiButton.textContent = window.lang.strings("add");
            this._ombiButton.classList.add("~neutral");
            this._ombiButton.classList.remove("~critical");
        }
    }

    get fromUser(): string { return this._fromUser.textContent; }
    set fromUser(v: string) { this._fromUser.textContent = v; }
   
    get referrals_enabled(): boolean { return this._referralsEnabled; }
    set referrals_enabled(v: boolean) {
        if (!window.referralsEnabled) return;
        this._referralsEnabled = v;
        if (v) {
            this._referralsButton.textContent = window.lang.strings("delete");
            this._referralsButton.classList.add("~critical");
            this._referralsButton.classList.remove("~neutral");
        } else {
            this._referralsButton.textContent = window.lang.strings("add");
            this._referralsButton.classList.add("~neutral");
            this._referralsButton.classList.remove("~critical");
        }
    }

    get default(): boolean { return this._defaultRadio.checked; }
    set default(v: boolean) { this._defaultRadio.checked = v; }

    constructor(name: string, p: Profile) {
        this._row = document.createElement("tr") as HTMLTableRowElement;
        let innerHTML = `
            <td><b class="profile-name"></b> <span class="profile-admin"></span></td>
            <td><input type="radio" name="profile-default"></td>
        `;
        if (window.ombiEnabled) innerHTML += `
            <td><span class="button @low profile-ombi"></span></td>
        `;
        if (window.referralsEnabled) innerHTML += `
            <td><span class="button @low profile-referrals"></span></td>
        `;
        innerHTML += `
            <td class="profile-from truncate"></td>
            <td class="profile-libraries"></td>
            <td><span class="button ~critical @low">${window.lang.strings("delete")}</span></td>
        `;
        this._row.innerHTML = innerHTML;
        this._name = this._row.querySelector("b.profile-name");
        this._adminChip = this._row.querySelector("span.profile-admin") as HTMLSpanElement;
        this._libraries = this._row.querySelector("td.profile-libraries") as HTMLTableDataCellElement;
        if (window.ombiEnabled)
            this._ombiButton = this._row.querySelector("span.profile-ombi") as HTMLSpanElement;
        if (window.referralsEnabled)
            this._referralsButton = this._row.querySelector("span.profile-referrals") as HTMLSpanElement;
        this._fromUser = this._row.querySelector("td.profile-from") as HTMLTableDataCellElement;
        this._defaultRadio = this._row.querySelector("input[type=radio]") as HTMLInputElement;
        this._defaultRadio.onclick = () => document.dispatchEvent(new CustomEvent("profiles-default", { detail: this.name }));
        (this._row.querySelector("span.\\~critical") as HTMLSpanElement).onclick = this.delete;

        this.update(name, p);
    }
    
    update = (name: string, p: Profile) => {
        this.name = name;
        this.admin = p.admin;
        this.fromUser = p.fromUser;
        this.libraries = p.libraries;
        this.ombi = p.ombi;
        this.referrals_enabled = p.referrals_enabled;
    }

    setOmbiFunc = (ombiFunc: (ombi: boolean) => void) => { this._ombiButton.onclick = () => ombiFunc(this._ombi); }
    setReferralFunc = (referralFunc: (enabled: boolean) => void) => { this._referralsButton.onclick = () => referralFunc(this._referralsEnabled); }

    remove = () => { document.dispatchEvent(new CustomEvent("profiles-delete", { detail: this._name })); this._row.remove(); }

    delete = () => _delete("/profiles", { "name": this.name }, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200 || req.status == 204) {
                this.remove();
            } else {
                window.notifications.customError("profileDelete", window.lang.var("notifications", "errorDeleteProfile", `"${this.name}"`));
            }
        }
    })

    asElement = (): HTMLTableRowElement => { return this._row; }
}

interface profileResp {
    default_profile: string;
    profiles: { [name: string]: Profile };
}

export class ProfileEditor {
    private _table = document.getElementById("table-profiles") as HTMLTableElement;
    private _createButton = document.getElementById("button-profile-create") as HTMLSpanElement;
    private _profiles: { [name: string]: profile } = {};
    private _default: string;
    private _ombiProfiles: ombiProfiles;

    private _createForm = document.getElementById("form-add-profile") as HTMLFormElement;
    private _profileName = document.getElementById("add-profile-name") as HTMLInputElement;
    private _userSelect = document.getElementById("add-profile-user") as HTMLSelectElement;
    private _storeHomescreen = document.getElementById("add-profile-homescreen") as HTMLInputElement;

    get empty(): boolean { return (Object.keys(this._table.children).length == 0) }
    set empty(state: boolean) {
        if (state) {
            this._table.innerHTML = `<tr><td class="empty">${window.lang.strings("inviteNoInvites")}</td></tr>`
        } else if (this._table.querySelector("td.empty")) {
            this._table.textContent = ``;
        }
    }

    get default(): string { return this._default; }
    set default(v: string) {
        this._default = v;
        if (v != "") { this._profiles[v].default = true; }
        for (let name in this._profiles) {
            if (name != v) { this._profiles[name].default = false; }
        }
    }

    load = () => _get("/profiles", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200) {
                let resp = req.response as profileResp;
                if (Object.keys(resp.profiles).length == 0) {
                    this.empty = true;
                } else {
                    this.empty = false;
                    for (let name in resp.profiles) {
                        if (name in this._profiles) {
                            this._profiles[name].update(name, resp.profiles[name]);
                        } else {
                            this._profiles[name] = new profile(name, resp.profiles[name]);
                            if (window.ombiEnabled)
                                this._profiles[name].setOmbiFunc((ombi: boolean) => {
                                    if (ombi) {
                                        this._ombiProfiles.delete(name, (req: XMLHttpRequest) => {
                                            if (req.readyState == 4) {
                                                if (req.status != 204) {
                                                    window.notifications.customError("errorDeleteOmbi", window.lang.notif("errorUnknown"));
                                                    return;
                                                }
                                                this._profiles[name].ombi = false;
                                            }
                                        });
                                    } else {
                                        window.modals.profiles.close();
                                        this._ombiProfiles.load(name);
                                    }
                                });
                            if (window.referralsEnabled)
                                this._profiles[name].setReferralFunc((enabled: boolean) => {
                                    if (enabled) {
                                        this.disableReferrals(name);
                                    } else {
                                        this.enableReferrals(name);
                                    }
                                });
                            this._table.appendChild(this._profiles[name].asElement());
                        }
                    }
                }
                this.default = resp.default_profile;
                window.modals.profiles.show();
            } else {
                window.notifications.customError("profileEditor", window.lang.notif("errorLoadProfiles"));
            }
        }
    })

    disableReferrals = (name: string) => _delete("/profiles/referral/" + name, null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        this.load();
    });

    enableReferrals = (name: string) => {
        const referralsInviteSelect = document.getElementById("enable-referrals-profile-invites") as HTMLSelectElement;
        const referralsExpiry = document.getElementById("enable-referrals-profile-expiry") as HTMLInputElement;
        _get("/invites", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4 || req.status != 200) return;

            let innerHTML = "";
            let invites = req.response["invites"] as Array<Invite>;
            window.availableProfiles = req.response["profiles"];
            if (invites) {
                for (let inv of invites) {
                    let name = inv.code;
                    if (inv.label) {
                        name = `${inv.label} (${inv.code})`;
                    }
                    innerHTML += `<option value="${inv.code}">${name}</option>`;
                }
            } else {
                innerHTML += `<option>${window.lang.strings("inviteNoInvites")}</option>`;
            }
            
            referralsInviteSelect.innerHTML = innerHTML;
        });

        const form = document.getElementById("form-enable-referrals-profile") as HTMLFormElement;
        const button = form.querySelector("span.submit") as HTMLSpanElement;
        form.onsubmit = (event: Event) => {
            event.preventDefault();
            toggleLoader(button);

            let send = {
                "profile": name,
                "invite": referralsInviteSelect.value
            };
            
            _post("/profiles/referral/" + send["profile"] + "/" + send["invite"] + "/" + (referralsExpiry.checked ? "with-expiry" : "none"), send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    if (req.status == 400) {
                        window.notifications.customError("unknownError", window.lang.notif("errorUnknown"));
                    } else if (req.status == 200 || req.status == 204) {
                        window.notifications.customSuccess("enableReferralsSuccess", window.lang.notif("referralsEnabled"));
                    }
                    window.modals.enableReferralsProfile.close();
                    this.load();
                }
            });
        };
        referralsExpiry.checked = false;
        window.modals.profiles.close();
        window.modals.enableReferralsProfile.show();
    };

    constructor() {
        (document.getElementById('setting-profiles') as HTMLSpanElement).onclick = this.load;
        document.addEventListener("profiles-default", (event: CustomEvent) => {
            const prevDefault = this.default;
            const newDefault = event.detail;
            _post("/profiles/default", { "name": newDefault }, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status == 200 || req.status == 204) {
                        this.default = newDefault;
                    } else {
                        this.default = prevDefault;
                        window.notifications.customError("profileDefault", window.lang.notif("errorSetDefaultProfile"));
                    }
                }
            });
        });
        document.addEventListener("profiles-delete", (event: CustomEvent) => {
            delete this._profiles[event.detail];
            this.load();
        });

        if (window.ombiEnabled)
            this._ombiProfiles = new ombiProfiles();

        this._createButton.onclick = () => _get("/users", null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200 || req.status == 204) {
                    let innerHTML = ``;
                    for (let user of req.response["users"]) {
                        innerHTML += `<option value="${user['id']}">${user['name']}</option>`;
                    }
                    this._userSelect.innerHTML = innerHTML;
                    this._storeHomescreen.checked = true;
                    window.modals.profiles.close();
                    window.modals.addProfile.show();
                } else {
                    window.notifications.customError("loadUsers", window.lang.notif("errorLoadUsers"));
                }
            }
        });

        this._createForm.onsubmit = (event: SubmitEvent) => {
            event.preventDefault();
            const button = this._createForm.querySelector("span.submit") as HTMLSpanElement;
            toggleLoader(button);
            let send = {
                "homescreen": this._storeHomescreen.checked,
                "id": this._userSelect.value,
                "name": this._profileName.value
            }
            _post("/profiles", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(button);
                    window.modals.addProfile.close();
                    if (req.status == 200 || req.status == 204) {
                        this.load();
                        window.notifications.customSuccess("createProfile", window.lang.var("notifications", "createProfile", `"${send['name']}"`));
                    } else {
                        window.notifications.customError("createProfile", window.lang.var("notifications", "errorCreateProfile", `"${send['name']}"`));
                    }
                    window.modals.profiles.show();
                }
            })
        };

    }
}

interface ombiUser {
    id: string;
    name: string;
}

export class ombiProfiles {
    private _form: HTMLFormElement;
    private _select: HTMLSelectElement;
    private _users: { [id: string]: string } = {};
    private _currentProfile: string;

    constructor() {
        this._form = document.getElementById("form-ombi-defaults") as HTMLFormElement;
        this._form.onsubmit = this.send;
        this._select = this._form.querySelector("select") as HTMLSelectElement;
    }
    send = () => {
        const button = this._form.querySelector("span.submit") as HTMLSpanElement;
        toggleLoader(button);
        let resp = {} as ombiUser;
        resp.id = this._select.value;
        resp.name = this._users[resp.id];
        _post("/profiles/ombi/" + this._currentProfile, resp, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                toggleLoader(button);
                if (req.status == 200 || req.status == 204) {
                    window.notifications.customSuccess("ombiDefaults", window.lang.notif("setOmbiProfile"));
                } else {
                    window.notifications.customError("ombiDefaults", window.lang.notif("errorSetOmbiProfile"));
                }
                window.modals.ombiProfile.close();
            }
        });
    }

    delete = (profile: string, post?: (req: XMLHttpRequest) => void) => _delete("/profiles/ombi/" + profile, null, post);

    load = (profile: string) => {
        this._currentProfile = profile;
        _get("/ombi/users", null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200 && "users" in req.response) {
                    const users = req.response["users"] as ombiUser[]; 
                    let innerHTML = "";
                    for (let user of users) {
                        this._users[user.id] = user.name;
                        innerHTML += `<option value="${user.id}">${user.name}</option>`;
                    }
                    this._select.innerHTML = innerHTML;
                    window.modals.ombiProfile.show();
                } else {
                    window.notifications.customError("ombiLoadError", window.lang.notif("errorLoadOmbiUsers"))
                }
            }
        });
    }
}
