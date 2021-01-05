import { _get, _post, _delete, toggleLoader } from "../modules/common.js";

interface Profile {
    admin: boolean;
    libraries: string;
    fromUser: string;
}

class profile implements Profile {
    private _row: HTMLTableRowElement;
    private _name: HTMLElement;
    private _adminChip: HTMLSpanElement;
    private _libraries: HTMLTableDataCellElement;
    private _fromUser: HTMLTableDataCellElement;
    private _defaultRadio: HTMLInputElement;

    get name(): string { return this._name.textContent; }
    set name(v: string) { this._name.textContent = v; }

    get admin(): boolean { return this._adminChip.classList.contains("chip"); }
    set admin(state: boolean) {
        if (state) {
            this._adminChip.classList.add("chip", "~info", "ml-half");
            this._adminChip.textContent = "Admin";
        } else {
            this._adminChip.classList.remove("chip", "~info", "ml-half");
            this._adminChip.textContent = "";
        }
    }

    get libraries(): string { return this._libraries.textContent; }
    set libraries(v: string) { this._libraries.textContent = v; }

    get fromUser(): string { return this._fromUser.textContent; }
    set fromUser(v: string) { this._fromUser.textContent = v; }
    
    get default(): boolean { return this._defaultRadio.checked; }
    set default(v: boolean) { this._defaultRadio.checked = v; }

    constructor(name: string, p: Profile) {
        this._row = document.createElement("tr") as HTMLTableRowElement;
        this._row.innerHTML = `
            <td><b class="profile-name"></b> <span class="profile-admin"></span></td>
            <td><input type="radio" name="profile-default"></td>
            <td class="profile-from ellipsis"></td>
            <td class="profile-libraries"></td>
            <td><span class="button ~critical !normal">Delete</span></td>
        `;
        this._name = this._row.querySelector("b.profile-name");
        this._adminChip = this._row.querySelector("span.profile-admin") as HTMLSpanElement;
        this._libraries = this._row.querySelector("td.profile-libraries") as HTMLTableDataCellElement;
        this._fromUser = this._row.querySelector("td.profile-from") as HTMLTableDataCellElement;
        this._defaultRadio = this._row.querySelector("input[type=radio]") as HTMLInputElement;
        this._defaultRadio.onclick = () => document.dispatchEvent(new CustomEvent("profiles-default", { detail: this.name }));
        (this._row.querySelector("span.button") as HTMLSpanElement).onclick = this.delete;

        this.update(name, p);
    }
    
    update = (name: string, p: Profile) => {
        this.name = name;
        this.admin = p.admin;
        this.fromUser = p.fromUser;
        this.libraries = p.libraries;
    }

    remove = () => { document.dispatchEvent(new CustomEvent("profiles-delete", { detail: this._name })); this._row.remove(); }

    delete = () => _delete("/profiles", { "name": this.name }, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200 || req.status == 204) {
                this.remove();
            } else {
                window.notifications.customError("profileDelete", `Failed to delete profile "${this.name}"`);
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

    private _createForm = document.getElementById("form-add-profile") as HTMLFormElement;
    private _profileName = document.getElementById("add-profile-name") as HTMLInputElement;
    private _userSelect = document.getElementById("add-profile-user") as HTMLSelectElement;
    private _storeHomescreen = document.getElementById("add-profile-homescreen") as HTMLInputElement;

    get empty(): boolean { return (Object.keys(this._table.children).length == 0) }
    set empty(state: boolean) {
        if (state) {
            this._table.innerHTML = `<tr><td class="empty">None</td></tr>`
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
                            this._table.appendChild(this._profiles[name].asElement());
                        }
                    }
                }
                this.default = resp.default_profile;
                window.modals.profiles.show();
            } else {
                window.notifications.customError("profileEditor", "Failed to load profiles.");
            }
        }
    })

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
                        window.notifications.customError("profileDefault", "Failed to set default profile.");
                    }
                }
            });
        });
        document.addEventListener("profiles-delete", (event: CustomEvent) => {
            delete this._profiles[event.detail];
            this.load();
        });

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
                    window.notifications.customError("loadUsers", "Failed to load users.");
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
                        window.notifications.customPositive("createProfile", "Success:", `created profile "${send['name']}"`);
                    } else {
                        window.notifications.customError("createProfile", `Failed to create profile "${send['name']}"`);
                    }
                    window.modals.profiles.show();
                }
            })
        };

    }
}
