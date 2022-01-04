import { _get, _post, _delete, toggleLoader, addLoader, removeLoader, insertText } from "../modules/common.js";
import { Marked } from "@ts-stack/markdown";
import { stripMarkdown } from "../modules/stripmd.js";

interface settingsBoolEvent extends Event { 
    detail: boolean;
}

interface Meta {
    name: string;
    description: string;
    advanced?: boolean;
    depends_true?: string;
    depends_false?: string;
}

interface Setting {
    name: string;
    description: string;
    required: boolean;
    requires_restart: boolean;
    advanced?: boolean;
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
        parts = [section, dep];
    }
    return parts
};

class DOMInput {
    protected _input: HTMLInputElement;
    private _container: HTMLDivElement;
    private _tooltip: HTMLDivElement;
    private _required: HTMLSpanElement;
    private _restart: HTMLSpanElement;
    private _advanced: boolean;

    private _advancedListener = (event: settingsBoolEvent) => {
        if (!Boolean(event.detail)) {
            this._input.parentElement.classList.add("unfocused");
        } else {
            this._input.parentElement.classList.remove("unfocused");
        }
    }

    get advanced(): boolean { return this._advanced; }
    set advanced(advanced: boolean) {
        this._advanced = advanced;
        if (advanced) {
            document.addEventListener("settings-advancedState", this._advancedListener);
        } else {
            document.removeEventListener("settings-advancedState", this._advancedListener);
        }
    }

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
            this._restart.classList.add("badge", "~info", "dark:~d_warning");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info", "dark:~d_warning");
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
            <input type="${inputType}" class="input ~neutral @low mt-2 mb-2">
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
        this.advanced = s.advanced;
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
    private _advanced: boolean;

    private _advancedListener = (event: settingsBoolEvent) => {
        if (!Boolean(event.detail)) {
            this._input.parentElement.classList.add("unfocused");
        } else {
            this._input.parentElement.classList.remove("unfocused");
        }
    }

    get advanced(): boolean { return this._advanced; }
    set advanced(advanced: boolean) {
        this._advanced = advanced;
        if (advanced) {
            document.addEventListener("settings-advancedState", this._advancedListener);
        } else {
            document.removeEventListener("settings-advancedState", this._advancedListener);
        }
    }

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
            this._restart.classList.add("badge", "~info", "dark:~d_warning");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info", "dark:~d_warning");
            this._restart.textContent = "";
        }
    }
    get value(): boolean { return this._input.checked; }
    set value(state: boolean) { this._input.checked = state; }
    constructor(setting: SBool, section: string, name: string) {
        this._container = document.createElement("div");
        this._container.classList.add("setting");
        this._container.innerHTML = `
        <label class="switch mb-2">
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
        this.advanced = s.advanced;
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
    private _advanced: boolean;

    private _advancedListener = (event: settingsBoolEvent) => {
        if (!Boolean(event.detail)) {
            this._container.classList.add("unfocused");
        } else {
            this._container.classList.remove("unfocused");
        }
    }

    get advanced(): boolean { return this._advanced; }
    set advanced(advanced: boolean) {
        this._advanced = advanced;
        if (advanced) {
            document.addEventListener("settings-advancedState", this._advancedListener);
        } else {
            document.removeEventListener("settings-advancedState", this._advancedListener);
        }
    }

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
            this._restart.classList.add("badge", "~info", "dark:~d_warning");
            this._restart.textContent = "R";
        } else {
            this._restart.classList.remove("badge", "~info", "dark:~d_warning");
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
            <div class="select ~neutral @low mt-2 mb-2">
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
                                            `<span class="badge ~info dark:~d_warning">R</span>`
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
        <p class="support lg my-2">${s.meta.description}</p>
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
                    // const oldValue = this.values[name];
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
    private _emailEditor = new EmailEditor();

    addSection = (name: string, s: Section, subButton?: HTMLElement) => {
        const section = new sectionPanel(s, name);
        this._sections[name] = section;
        this._panel.appendChild(this._sections[name].asElement());
        const button = document.createElement("span") as HTMLSpanElement;
        button.classList.add("button", "~neutral", "@low", "settings-section-button", "justify-between", "mb-2");
        button.textContent = s.meta.name;
        if (subButton) { button.appendChild(subButton); }
        button.onclick = () => { this._showPanel(name); };
        if (s.meta.depends_true || s.meta.depends_false) {
            let dependant = splitDependant(name, s.meta.depends_true || s.meta.depends_false);
            let state = true;
            if (s.meta.depends_false) { state = false; }
            document.addEventListener(`settings-${dependant[0]}-${dependant[1]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    button.classList.add("unfocused");
                    document.dispatchEvent(new CustomEvent(`settings-${name}`, { detail: false }));
                } else {
                    button.classList.remove("unfocused");
                    document.dispatchEvent(new CustomEvent(`settings-${name}`, { detail: true }));
                }
            });
            document.addEventListener(`settings-${dependant[0]}`, (event: settingsBoolEvent) => {
                if (Boolean(event.detail) !== state) {
                    button.classList.add("unfocused");
                    document.dispatchEvent(new CustomEvent(`settings-${name}`, { detail: false }));
                }
            }); 
        }
        if (s.meta.advanced) {
            document.addEventListener("settings-advancedState", (event: settingsBoolEvent) => {
                if (!Boolean(event.detail)) {
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

    private _showLogs = () => _get("/logs", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4 && req.status == 200) {
            (document.getElementById("log-area") as HTMLPreElement).textContent = req.response["log"] as string;
            window.modals.logs.show();
        }
    });

    constructor() {
        this._sections = {};
        this._buttons = {};
        document.addEventListener("settings-section-changed", () => this._saveButton.classList.remove("unfocused"));
        document.getElementById("settings-restart").onclick = () => {
            _post("/restart", null, () => {});
            window.modals.settingsRefresh.modal.querySelector("span.heading").textContent = window.lang.strings("settingsRestarting");
            window.modals.settingsRefresh.show();
        };
        this._saveButton.onclick = this._save;
        document.addEventListener("settings-requires-restart", () => { this._needsRestart = true; });
        document.getElementById("settings-logs").onclick = this._showLogs;
        const advancedEnableToggle = document.getElementById("settings-advanced-enabled") as HTMLInputElement;
        advancedEnableToggle.onchange = () => {
            document.dispatchEvent(new CustomEvent("settings-advancedState", { detail: advancedEnableToggle.checked }));
            const parent = advancedEnableToggle.parentElement;
            if (advancedEnableToggle.checked) {
                parent.classList.add("~urge");
                parent.classList.remove("~neutral");
            } else {
                parent.classList.add("~neutral");
                parent.classList.remove("~urge");
            }
        };
        advancedEnableToggle.checked = false;
    }

    private _addMatrix = () => {
        // Modify the login modal, why not
        const modal = document.getElementById("form-matrix") as HTMLFormElement;
        modal.onsubmit = (event: Event) => {
            event.preventDefault();
            const button = modal.querySelector("span.submit") as HTMLSpanElement;
            addLoader(button);
            let send = {
                homeserver: (document.getElementById("matrix-homeserver") as HTMLInputElement).value,
                username: (document.getElementById("matrix-user") as HTMLInputElement).value,
                password: (document.getElementById("matrix-password") as HTMLInputElement).value
            }
            _post("/matrix/login", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    removeLoader(button);
                    if (req.status == 400) {
                        window.notifications.customError("errorUnknown", window.lang.notif(req.response["error"] as string));
                        return;
                    } else if (req.status == 401) {
                        window.notifications.customError("errorUnauthorized", req.response["error"] as string);
                        return;
                    } else if (req.status == 500) {
                        window.notifications.customError("errorAddMatrix", window.lang.notif("errorFailureCheckLogs"));
                        return;
                    }
                    window.modals.matrix.close();
                    _post("/restart", null, () => {});
                    window.location.reload();
                }
            }, true);
        };
        window.modals.matrix.show();
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
                    if (name == "messages") {
                        const editButton = document.createElement("div");
                        editButton.classList.add("tooltip", "left");
                        editButton.innerHTML = `
                        <span class="button ~neutral @low">
                            <i class="icon ri-edit-line"></i>
                        </span>
                        <span class="content sm">
                        ${window.lang.get("strings", "customizeMessages")}
                        </span>
                        `;
                        (editButton.querySelector("span.button") as HTMLSpanElement).onclick = this._emailEditor.showList;
                        this.addSection(name, settings.sections[name], editButton);
                    } else if (name == "updates") {
                        const icon = document.createElement("span") as HTMLSpanElement;
                        if (window.updater.updateAvailable) {
                            icon.classList.add("button", "~urge");
                            icon.innerHTML = `<i class="ri-download-line" title="${window.lang.strings("update")}"></i>`;
                            icon.onclick = () => window.updater.checkForUpdates(window.modals.updateInfo.show);
                        }
                        this.addSection(name, settings.sections[name], icon);
                    } else if (name == "matrix" && !window.matrixEnabled) {
                        const addButton = document.createElement("div");
                        addButton.classList.add("tooltip", "left");
                        addButton.innerHTML = `
                        <span class="button ~neutral @low">+</span>
                        <span class="content sm">
                        ${window.lang.strings("linkMatrix")}
                        </span>
                        `;
                        (addButton.querySelector("span.button") as HTMLSpanElement).onclick = this._addMatrix;
                        this.addSection(name, settings.sections[name], addButton);
                    } else {
                        this.addSection(name, settings.sections[name]);
                    }
                }
            }
            this._showPanel(settings.order[0]);
            document.dispatchEvent(new CustomEvent("settings-loaded"));
            document.dispatchEvent(new CustomEvent("settings-advancedState", { detail: false }));
            this._saveButton.classList.add("unfocused");
            this._needsRestart = false;
        }
    })
}

export interface templateEmail {
    content: string;
    variables: string[];
    conditionals: string[];
    values: { [key: string]: string };
    html: string;
    plaintext: string;
}

interface emailListEl {
    name: string;
    enabled: boolean;
}

class EmailEditor {
    private _currentID: string;
    private _names: { [id: string]: emailListEl };
    private _content: string;
    private _templ: templateEmail;
    private _form = document.getElementById("form-editor") as HTMLFormElement;
    private _header = document.getElementById("header-editor") as HTMLSpanElement;
    private _variables = document.getElementById("editor-variables") as HTMLDivElement;
    private _variablesLabel = document.getElementById("label-editor-variables") as HTMLElement;
    private _conditionals = document.getElementById("editor-conditionals") as HTMLDivElement;
    private _conditionalsLabel = document.getElementById("label-editor-conditionals") as HTMLElement;
    private _textArea = document.getElementById("textarea-editor") as HTMLTextAreaElement;
    private _preview = document.getElementById("editor-preview") as HTMLDivElement;
    private _previewContent: HTMLElement;
    // private _timeout: number;
    // private _finishInterval = 200;

    loadEditor = (id: string) => {
        this._currentID = id;
        _get("/config/emails/" + id, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    window.notifications.customError("loadTemplateError", window.lang.notif("errorFailureCheckLogs"));
                    return;
                }
                if (this._names[id] !== undefined) {
                    this._header.textContent = this._names[id].name;
                } 
                this._templ = req.response as templateEmail;
                this._textArea.value = this._templ.content;
                if (this._templ.html == "") {
                    this._preview.innerHTML = `<pre class="preview-content" class="font-mono bg-inherit"></pre>`;
                } else {
                    this._preview.innerHTML = this._templ.html;
                }
                this._previewContent = this._preview.getElementsByClassName("preview-content")[0] as HTMLElement;
                this.loadPreview();
                this._content = this._templ.content;
                const colors = ["info", "urge", "positive", "neutral"];
                let innerHTML = '';
                for (let i = 0; i < this._templ.variables.length; i++) {
                    let ci = i % colors.length;
                    innerHTML += '<span class="button ~' + colors[ci] +' @low mb-4" style="margin-left: 0.25rem; margin-right: 0.25rem;"></span>'
                }
                if (this._templ.variables.length == 0) {
                    this._variablesLabel.classList.add("unfocused");
                } else {
                    this._variablesLabel.classList.remove("unfocused");
                }
                this._variables.innerHTML = innerHTML
                let buttons = this._variables.querySelectorAll("span.button") as NodeListOf<HTMLSpanElement>;
                for (let i = 0; i < this._templ.variables.length; i++) {
                    buttons[i].innerHTML = `<span class="font-mono bg-inherit">` + this._templ.variables[i] + `</span>`;
                    buttons[i].onclick = () => {
                        insertText(this._textArea, this._templ.variables[i]);
                        this.loadPreview();
                        // this._timeout = setTimeout(this.loadPreview, this._finishInterval);
                    }
                }

                innerHTML = '';
                if (this._templ.conditionals == null || this._templ.conditionals.length == 0) {
                    this._conditionalsLabel.classList.add("unfocused");
                } else {
                    for (let i = this._templ.conditionals.length-1; i >= 0; i--) {
                        let ci = i % colors.length;
                        innerHTML += '<span class="button ~' + colors[ci] +' @low mb-4" style="margin-left: 0.25rem; margin-right: 0.25rem;"></span>'
                    }
                    this._conditionalsLabel.classList.remove("unfocused");
                    this._conditionals.innerHTML = innerHTML
                    buttons = this._conditionals.querySelectorAll("span.button") as NodeListOf<HTMLSpanElement>;
                    for (let i = 0; i < this._templ.conditionals.length; i++) {
                        buttons[i].innerHTML = `<span class="font-mono bg-inherit">{if ` + this._templ.conditionals[i].slice(1) + `</span>`;
                        buttons[i].onclick = () => {
                            insertText(this._textArea, "{if " + this._templ.conditionals[i].slice(1) + "{endif}");
                            this.loadPreview();
                            // this._timeout = setTimeout(this.loadPreview, this._finishInterval);
                        }
                    }
                }
                window.modals.editor.show();
            }
        })
    }
    loadPreview = () => {
        let content = this._textArea.value;
        if (this._templ.variables) {
            for (let variable of this._templ.variables) {
                let value = this._templ.values[variable.slice(1, -1)];
                if (value === undefined) { value = variable; }
                content = content.replace(new RegExp(variable, "g"), value);
            }
        }
        if (this._templ.html == "") {
            content = stripMarkdown(content);
            this._previewContent.textContent = content;
        } else {
            content = Marked.parse(content);
            this._previewContent.innerHTML = content;
        }
        // _post("/config/emails/" + this._currentID + "/test", { "content": this._textArea.value }, (req: XMLHttpRequest) => {
        //     if (req.readyState == 4) {
        //         if (req.status != 200) {
        //             window.notifications.customError("loadTemplateError", window.lang.notif("errorFailureCheckLogs"));
        //             return;
        //         }
        //         this._preview.innerHTML = (req.response as Email).html;
        //     }
        // }, true);
    }

    showList = () => {
        _get("/config/emails?lang=" + window.language, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    window.notifications.customError("loadTemplateError", window.lang.notif("errorFailureCheckLogs"));
                    return;
                }
                this._names = req.response;
                const list = document.getElementById("customize-list") as HTMLDivElement;
                list.textContent = '';
                for (let id in this._names) {
                    const tr = document.createElement("tr") as HTMLTableRowElement;
                    let resetButton = ``;
                    if (this._names[id].enabled) {
                        resetButton = `<i class="icon ri-restart-line" title="${window.lang.get("strings", "reset")}"></i>`;
                    }
                    tr.innerHTML = `
                    <td>${this._names[id].name}</td>
                    <td class="table-inline justify-center"><span class="customize-reset">${resetButton}</span></td>
                    <td><span class="button ~info @low" title="${window.lang.get("strings", "edit")}"><i class="icon ri-edit-line"></i></span></td>
                    `;
                    (tr.querySelector("span.button") as HTMLSpanElement).onclick = () => {
                        window.modals.customizeEmails.close()
                        this.loadEditor(id);
                    };
                    if (this._names[id].enabled) {
                        const rb = tr.querySelector("span.customize-reset") as HTMLElement;
                        rb.classList.add("button");
                        rb.onclick = () => _post("/config/emails/" + id + "/state/disable", null, (req: XMLHttpRequest) => {
                            if (req.readyState == 4) {
                                if (req.status != 200 && req.status != 204) {
                                    window.notifications.customError("setEmailStateError", window.lang.notif("errorFailureCheckLogs"));
                                    return;
                                }
                                rb.remove();
                            }
                        });
                    }
                    list.appendChild(tr);
                }
                window.modals.customizeEmails.show();
            }
        });
    }

    constructor() {
        this._textArea.onkeyup = () => {
            // clearTimeout(this._timeout);
            // this._timeout = setTimeout(this.loadPreview, this._finishInterval);
            this.loadPreview();
        };
        // this._textArea.onkeydown = () => {
        //     clearTimeout(this._timeout);
        // };

        this._form.onsubmit = (event: Event) => {
            event.preventDefault()
            if (this._textArea.value == this._content && this._names[this._currentID].enabled) {
                window.modals.editor.close();
                return;
            }
            _post("/config/emails/" + this._currentID, { "content": this._textArea.value }, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    window.modals.editor.close();
                    if (req.status != 200) {
                        window.notifications.customError("saveEmailError", window.lang.notif("errorSaveEmail"));
                        return;
                    }
                    window.notifications.customSuccess("saveEmail", window.lang.notif("saveEmail"));
                }
            });
        };
    }
}
