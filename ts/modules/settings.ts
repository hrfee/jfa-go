import { _get, _post, toggleLoader } from "../modules/common.js";

interface settingsBoolEvent extends Event { 
    detail: boolean;
}

interface Meta {
    name: string;
    description: string;
    depends_true?: string;
    depends_false?: string;
}

interface Setting {
    name: string;
    description: string;
    required: boolean;
    requires_restart: boolean;
    type: string;
    value: string | boolean | number;
    depends_true?: string;
    depends_false?: string;

    asElement: () => HTMLElement;
    update: (s: Setting) => void;
}

const splitDependant = (section: string, dep: string): string[] => {
    let parts = dep.split("|");
    if (parts.length == 1) {
        parts = [section, parts[0]];
    }
    return parts
};

class DOMInput {
    protected _input: HTMLInputElement;
    private _container: HTMLDivElement;
    private _tooltip: HTMLDivElement;
    private _required: HTMLSpanElement;
    private _restart: HTMLSpanElement;

    get name(): string { return this._container.querySelector("span.setting-label").textContent; }
    set name(n: string) { this._container.querySelector("span.setting-label").textContent = n; }

    get description(): string { return this._tooltip.querySelector("span.content").textContent; } 
    set description(d: string) {
        const content = this._tooltip.querySelector("span.content") as HTMLSpanElement;
        content.textContent = d;
        if (d == "") {
            this._tooltip.classList.add("unfocused");
        } else {
            this._tooltip.classList.remove("unfocused");
        }
    }

    get required(): boolean { return this._required.classList.contains("badge"); }
    set required(state: boolean) {
        if (state) {
            this._required.classList.add("badge", "~critical");
            this._required.textContent = "*";
        } else {
            this._required.classList.remove("badge", "~critical");
            this._required.textContent = "";
        }
    }
    
    get requires_restart(): boolean { return this._restart.classList.contains("badge"); }
    set requires_restart(state: boolean) {
        if (state) {
            this._restart.classList.add("badge", "~info");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info");
            this._restart.textContent = "";
        }
    }

    constructor(inputType: string, setting: Setting, section: string, name: string) {
        this._container = document.createElement("div");
        this._container.classList.add("setting");
        this._container.innerHTML = `
        <label class="label">
            <span class="setting-label"></span> <span class="setting-required"></span> <span class="setting-restart"></span>
            <div class="setting-tooltip tooltip right unfocused">
                <i class="icon ri-information-line"></i>
                <span class="content sm"></span>
            </div>
            <input type="${inputType}" class="input ~neutral !normal mt-half mb-half">
        </label>
        `;
        this._tooltip = this._container.querySelector("div.setting-tooltip") as HTMLDivElement;
        this._required = this._container.querySelector("span.setting-required") as HTMLSpanElement;
        this._restart = this._container.querySelector("span.setting-restart") as HTMLSpanElement;
        this._input = this._container.querySelector("input[type=" + inputType + "]") as HTMLInputElement;
        if (setting.depends_false || setting.depends_true) {
            let dependant = splitDependant(section, setting.depends_true || setting.depends_false);
            let state = true;
            if (setting.depends_false) { state = false; }
            document.addEventListener(`settings-${dependant[0]}-${dependant[1]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    this._input.parentElement.classList.add("unfocused");
                } else {
                    this._input.parentElement.classList.remove("unfocused");
                }
            });
        }
        const onValueChange = () => {
            const event = new CustomEvent(`settings-${section}-${name}`, { "detail": this.value })
            document.dispatchEvent(event);
            if (this.requires_restart) { document.dispatchEvent(new CustomEvent("settings-requires-restart")); }
        };
        this._input.onchange = onValueChange;
        this.update(setting);
    }

    get value(): any { return this._input.value; }
    set value(v: any) { this._input.value = v; }

    update = (s: Setting) => {
        this.name = s.name;
        this.description = s.description;
        this.required = s.required;
        this.requires_restart = s.requires_restart;
        this.value = s.value;
    }
    
    asElement = (): HTMLDivElement => { return this._container; }
}

interface SText extends Setting {
    value: string;
}
class DOMText extends DOMInput implements SText {
    constructor(setting: Setting, section: string, name: string) { super("text", setting, section, name); }
    type: string = "text";
    get value(): string { return this._input.value }
    set value(v: string) { this._input.value = v; }
}

interface SPassword extends Setting {
    value: string;
}
class DOMPassword extends DOMInput implements SPassword {
    constructor(setting: Setting, section: string, name: string) { super("password", setting, section, name); }
    type: string = "password";
    get value(): string { return this._input.value }
    set value(v: string) { this._input.value = v; }
}

interface SEmail extends Setting {
    value: string;
}
class DOMEmail extends DOMInput implements SEmail {
    constructor(setting: Setting, section: string, name: string) { super("email", setting, section, name); }
    type: string = "email";
    get value(): string { return this._input.value }
    set value(v: string) { this._input.value = v; }
}

interface SNumber extends Setting {
    value: number;
}
class DOMNumber extends DOMInput implements SNumber {
    constructor(setting: Setting, section: string, name: string) { super("number", setting, section, name); }
    type: string = "number";
    get value(): number { return +this._input.value; }
    set value(v: number) { this._input.value = ""+v; }
}

interface SBool extends Setting {
    value: boolean;
}
class DOMBool implements SBool {
    protected _input: HTMLInputElement;
    private _container: HTMLDivElement;
    private _tooltip: HTMLDivElement;
    private _required: HTMLSpanElement;
    private _restart: HTMLSpanElement;
    type: string = "bool";

    get name(): string { return this._container.querySelector("span.setting-label").textContent; }
    set name(n: string) { this._container.querySelector("span.setting-label").textContent = n; }

    get description(): string { return this._tooltip.querySelector("span.content").textContent; } 
    set description(d: string) {
        const content = this._tooltip.querySelector("span.content") as HTMLSpanElement;
        content.textContent = d;
        if (d == "") {
            this._tooltip.classList.add("unfocused");
        } else {
            this._tooltip.classList.remove("unfocused");
        }
    }

    get required(): boolean { return this._required.classList.contains("badge"); }
    set required(state: boolean) {
        if (state) {
            this._required.classList.add("badge", "~critical");
            this._required.textContent = "*";
        } else {
            this._required.classList.remove("badge", "~critical");
            this._required.textContent = "";
        }
    }
    
    get requires_restart(): boolean { return this._restart.classList.contains("badge"); }
    set requires_restart(state: boolean) {
        if (state) {
            this._restart.classList.add("badge", "~info");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info");
            this._restart.textContent = "";
        }
    }
    get value(): boolean { return this._input.checked; }
    set value(state: boolean) { this._input.checked = state; }
    constructor(setting: SBool, section: string, name: string) {
        this._container = document.createElement("div");
        this._container.classList.add("setting");
        this._container.innerHTML = `
        <label class="switch mb-half">
            <input type="checkbox">
            <span class="setting-label"></span> <span class="setting-required"></span> <span class="setting-restart"></span>
            <div class="setting-tooltip tooltip right unfocused">
                <i class="icon ri-information-line"></i>
                <span class="content sm"></span>
            </div>
        </label>
        `;
        this._tooltip = this._container.querySelector("div.setting-tooltip") as HTMLDivElement;
        this._required = this._container.querySelector("span.setting-required") as HTMLSpanElement;
        this._restart = this._container.querySelector("span.setting-restart") as HTMLSpanElement;
        this._input = this._container.querySelector("input[type=checkbox]") as HTMLInputElement;
        const onValueChange = () => {
            const event = new CustomEvent(`settings-${section}-${name}`, { "detail": this.value })
            document.dispatchEvent(event);
        };
        this._input.onchange = () => { 
            onValueChange();
            if (this.requires_restart) { document.dispatchEvent(new CustomEvent("settings-requires-restart")); }
        };
        document.addEventListener(`settings-loaded`, onValueChange);

        if (setting.depends_false || setting.depends_true) {
            let dependant = splitDependant(section, setting.depends_true || setting.depends_false);
            let state = true;
            if (setting.depends_false) { state = false; }
            document.addEventListener(`settings-${dependant[0]}-${dependant[1]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    this._input.parentElement.classList.add("unfocused");
                } else {
                    this._input.parentElement.classList.remove("unfocused");
                }
            });
        }
        this.update(setting);
    }
    update = (s: SBool) => {
        this.name = s.name;
        this.description = s.description;
        this.required = s.required;
        this.requires_restart = s.requires_restart;
        this.value = s.value;
    }
    
    asElement = (): HTMLDivElement => { return this._container; }
}

interface SSelect extends Setting {
    options: string[][];
    value: string;
}
class DOMSelect implements SSelect {
    protected _select: HTMLSelectElement;
    private _container: HTMLDivElement;
    private _tooltip: HTMLDivElement;
    private _required: HTMLSpanElement;
    private _restart: HTMLSpanElement;
    private _options: string[][];
    type: string = "bool";

    get name(): string { return this._container.querySelector("span.setting-label").textContent; }
    set name(n: string) { this._container.querySelector("span.setting-label").textContent = n; }

    get description(): string { return this._tooltip.querySelector("span.content").textContent; } 
    set description(d: string) {
        const content = this._tooltip.querySelector("span.content") as HTMLSpanElement;
        content.textContent = d;
        if (d == "") {
            this._tooltip.classList.add("unfocused");
        } else {
            this._tooltip.classList.remove("unfocused");
        }
    }

    get required(): boolean { return this._required.classList.contains("badge"); }
    set required(state: boolean) {
        if (state) {
            this._required.classList.add("badge", "~critical");
            this._required.textContent = "*";
        } else {
            this._required.classList.remove("badge", "~critical");
            this._required.textContent = "";
        }
    }
    
    get requires_restart(): boolean { return this._restart.classList.contains("badge"); }
    set requires_restart(state: boolean) {
        if (state) {
            this._restart.classList.add("badge", "~info");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info");
            this._restart.textContent = "";
        }
    }
    get value(): string { return this._select.value; }
    set value(v: string) { this._select.value = v; }

    get options(): string[][] { return this._options; }
    set options(opt: string[][]) {
        this._options = opt;
        let innerHTML = "";
        for (let option of this._options) {
            innerHTML += `<option value="${option[0]}">${option[1]}</option>`;
        }
        this._select.innerHTML = innerHTML;
    }

    constructor(setting: SSelect, section: string, name: string) {
        this._options = [];
        this._container = document.createElement("div");
        this._container.classList.add("setting");
        this._container.innerHTML = `
        <label class="label">
            <span class="setting-label"></span> <span class="setting-required"></span> <span class="setting-restart"></span>
            <div class="setting-tooltip tooltip right unfocused">
                <i class="icon ri-information-line"></i>
                <span class="content sm"></span>
            </div>
            <div class="select ~neutral !normal mt-half mb-half">
                <select class="settings-select"></select>
            </div>
        </label>
        `;
        this._tooltip = this._container.querySelector("div.setting-tooltip") as HTMLDivElement;
        this._required = this._container.querySelector("span.setting-required") as HTMLSpanElement;
        this._restart = this._container.querySelector("span.setting-restart") as HTMLSpanElement;
        this._select = this._container.querySelector("select.settings-select") as HTMLSelectElement;
        if (setting.depends_false || setting.depends_true) {
            let dependant = splitDependant(section, setting.depends_true || setting.depends_false);
            let state = true;
            if (setting.depends_false) { state = false; }
            document.addEventListener(`settings-${dependant[0]}-${dependant[1]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    this._container.classList.add("unfocused");
                } else {
                    this._container.classList.remove("unfocused");
                }
            });
        }
        const onValueChange = () => {
            const event = new CustomEvent(`settings-${section}-${name}`, { "detail": this.value })
            document.dispatchEvent(event);
            if (this.requires_restart) { document.dispatchEvent(new CustomEvent("settings-requires-restart")); }
        };
        this._select.onchange = onValueChange;
        document.addEventListener(`settings-loaded`, onValueChange);

        const message = document.getElementById("settings-message") as HTMLElement;
        message.innerHTML = window.lang.var("strings",
                                            "settingsRequiredOrRestartMessage",
                                            `<span class="badge ~critical">*</span>`,
                                            `<span class="badge ~info">R</span>`
        );

        this.update(setting);
    }
    update = (s: SSelect) => {
        this.name = s.name;
        this.description = s.description;
        this.required = s.required;
        this.requires_restart = s.requires_restart;
        this.options = s.options;
        this.value = s.value;
    }

    asElement = (): HTMLDivElement => { return this._container; }
}

interface Section {
    meta: Meta;
    order: string[];
    settings: { [settingName: string]: Setting };
}

class sectionPanel {
    private _section: HTMLDivElement;
    private _settings: { [name: string]: Setting };
    private _sectionName: string;
    values: { [field: string]: string } = {};

    constructor(s: Section, sectionName: string) {
        this._sectionName = sectionName;
        this._settings = {};
        this._section = document.createElement("div") as HTMLDivElement;
        this._section.classList.add("settings-section", "unfocused");
        this._section.innerHTML = `
        <span class="heading">${s.meta.name}</span>
        <p class="support lg">${s.meta.description}</p>
        `;

        this.update(s);
    }
    update = (s: Section) => {
        for (let name of s.order) {
            let setting: Setting = s.settings[name];
            if (name in this._settings) {
                this._settings[name].update(setting);
            } else {
                switch (setting.type) {
                    case "text":
                        setting = new DOMText(setting, this._sectionName, name);
                        break;
                    case "password":
                        setting = new DOMPassword(setting, this._sectionName, name);
                        break;
                    case "email":
                        setting = new DOMEmail(setting, this._sectionName, name);
                        break;
                    case "number":
                        setting = new DOMNumber(setting, this._sectionName, name);
                        break;
                    case "bool":
                        setting = new DOMBool(setting as SBool, this._sectionName, name);
                        break;
                    case "select":
                        setting = new DOMSelect(setting as SSelect, this._sectionName, name);
                        break;
                }
                this.values[name] = ""+setting.value;
                document.addEventListener(`settings-${this._sectionName}-${name}`, (event: CustomEvent) => {
                    const oldValue = this.values[name];
                    this.values[name] = ""+event.detail;
                    document.dispatchEvent(new CustomEvent("settings-section-changed"));
                });
                this._section.appendChild(setting.asElement());
                this._settings[name] = setting;
            }
        }
    }
    
    get visible(): boolean { return !this._section.classList.contains("unfocused"); }
    set visible(s: boolean) {
        if (s) {
            this._section.classList.remove("unfocused");
        } else {
            this._section.classList.add("unfocused");
        }
    }

    asElement = (): HTMLDivElement => { return this._section; }
}

interface Settings {
    order: string[];
    sections: { [sectionName: string]: Section };
}

export class settingsList {
    private _saveButton = document.getElementById("settings-save") as HTMLSpanElement;
    private _saveNoRestart = document.getElementById("settings-apply-no-restart") as HTMLSpanElement;
    private _saveRestart = document.getElementById("settings-apply-restart") as HTMLSpanElement;

    private _panel = document.getElementById("settings-panel") as HTMLDivElement;
    private _sidebar = document.getElementById("settings-sidebar") as HTMLDivElement;
    private _sections: { [name: string]: sectionPanel }
    private _buttons: { [name: string]: HTMLSpanElement }
    private _needsRestart: boolean = false;

    addSection = (name: string, s: Section) => {
        const section = new sectionPanel(s, name);
        this._sections[name] = section;
        this._panel.appendChild(this._sections[name].asElement());
        const button = document.createElement("span") as HTMLSpanElement;
        button.classList.add("button", "~neutral", "!low", "settings-section-button", "mb-half");
        button.textContent = s.meta.name;
        button.onclick = () => { this._showPanel(name); };
        if (s.meta.depends_true || s.meta.depends_false) {
            let dependant = splitDependant(name, s.meta.depends_true || s.meta.depends_false);
            let state = true;
            if (s.meta.depends_false) { state = false; }
            document.addEventListener(`settings-${dependant[0]}-${dependant[1]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    button.classList.add("unfocused");
                } else {
                    button.classList.remove("unfocused");
                }
            });
        }
        this._buttons[name] = button;
        this._sidebar.appendChild(this._buttons[name]);
    }

    private _showPanel = (name: string) => {
        for (let n in this._sections) {
            if (n == name) {
                this._sections[name].visible = true;
                this._buttons[name].classList.add("selected");
            } else {
                this._sections[n].visible = false;
                this._buttons[n].classList.remove("selected");
            }
        }
    }

    private _save = () => {
        let config = {};
        for (let name in this._sections) {
            config[name] = this._sections[name].values;
        }
        if (this._needsRestart) { 
            this._saveRestart.onclick = () => {
                config["restart-program"] = true;
                this._send(config, () => {
                    window.modals.settingsRestart.close(); 
                    window.modals.settingsRefresh.show();
                });
            };
            this._saveNoRestart.onclick = () => {
                config["restart-program"] = false;
                this._send(config, window.modals.settingsRestart.close); 
            }
            window.modals.settingsRestart.show(); 
        } else {
            this._send(config);
        }
        // console.log(config);
    }

    private _send = (config: Object, run?: () => void) => _post("/config", config, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200 || req.status == 204) {
                window.notifications.customSuccess("settingsSaved", window.lang.notif("saveSettings"));
            } else {
                window.notifications.customError("settingsSaved", window.lang.notif("errorSaveSettings"));
            }
            this.reload();
            if (run) { run(); }
        }
    });

    constructor() {
        this._sections = {};
        this._buttons = {};
        document.addEventListener("settings-section-changed", () => this._saveButton.classList.remove("unfocused"));
        this._saveButton.onclick = this._save;
        document.addEventListener("settings-requires-restart", () => { this._needsRestart = true; });

        if (window.ombiEnabled) {
            let ombi = new ombiDefaults();
            this._sidebar.appendChild(ombi.button());
        }
    }

    reload = () => _get("/config", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                window.notifications.customError("settingsLoadError", window.lang.notif("errorLoadSettings"));
                return;
            }
            let settings = req.response as Settings;
            for (let name of settings.order) {
                if (name in this._sections) {
                    this._sections[name].update(settings.sections[name]);
                } else {
                    this.addSection(name, settings.sections[name]);
                }
            }
            this._showPanel(settings.order[0]);
            document.dispatchEvent(new CustomEvent("settings-loaded"));
            this._saveButton.classList.add("unfocused");
            this._needsRestart = false;
        }
    })
}

interface ombiUser {
    id: string;
    name: string;
}

class ombiDefaults {
    private _form: HTMLFormElement;
    private _button: HTMLSpanElement;
    private _select: HTMLSelectElement;
    private _users: { [id: string]: string } = {};
    constructor() {
        this._button = document.createElement("span") as HTMLSpanElement;
        this._button.classList.add("button", "~neutral", "!low", "settings-section-button", "mb-half");
        this._button.innerHTML = `<span class="flex">${window.lang.strings("ombiUserDefaults")} <i class="ri-link-unlink-m ml-half"></i></span>`;
        this._button.onclick = this.load;
        this._form = document.getElementById("form-ombi-defaults") as HTMLFormElement;
        this._form.onsubmit = this.send;
        this._select = this._form.querySelector("select") as HTMLSelectElement;
    }
    button = (): HTMLSpanElement => { return this._button; }
    send = () => {
        const button = this._form.querySelector("span.submit") as HTMLSpanElement;
        toggleLoader(button);
        let resp = {} as ombiUser;
        resp.id = this._select.value;
        resp.name = this._users[resp.id];
        _post("/ombi/defaults", resp, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                toggleLoader(button);
                if (req.status == 200 || req.status == 204) {
                    window.notifications.customSuccess("ombiDefaults", window.lang.notif("setOmbiDefaults"));
                } else {
                    window.notifications.customError("ombiDefaults", window.lang.notif("errorSetOmbiDefaults"));
                }
                window.modals.ombiDefaults.close();
            }
        });
    }

    load = () => {
        toggleLoader(this._button);
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
                    toggleLoader(this._button);
                    window.modals.ombiDefaults.show();
                } else {
                    toggleLoader(this._button);
                    window.notifications.customError("ombiLoadError", window.lang.notif("errorLoadOmbiUsers"))
                }
            }
        });
    }
}
