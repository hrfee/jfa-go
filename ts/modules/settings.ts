import {
    _get,
    _post,
    _delete,
    _download,
    _upload,
    toggleLoader,
    addLoader,
    removeLoader,
    insertText,
    toClipboard,
    toDateString,
    SetupCopyButton,
} from "../modules/common.js";
import { Marked } from "@ts-stack/markdown";
import { stripMarkdown } from "../modules/stripmd.js";

declare var window: GlobalWindow;

const toBool = (s: string): boolean => {
    return s == "false" ? false : Boolean(s);
};

interface BackupDTO {
    size: string;
    name: string;
    path: string;
    date: number;
    commit: string;
}

interface settingsChangedEvent extends Event {
    detail: {
        value: string;
        hidden: boolean;
    };
}

interface advancedEvent extends Event {
    detail: boolean;
}

const changedEvent = (section: string, setting: string, value: string, hidden: boolean = false) => {
    return new CustomEvent(`settings-${section}-${setting}`, {
        detail: {
            value: value,
            hidden: hidden,
        },
    });
};

type SettingType = string;

const BoolType: SettingType = "bool";
const SelectType: SettingType = "select";
const TextType: SettingType = "text";
const PasswordType: SettingType = "password";
const NumberType: SettingType = "number";
const NoteType: SettingType = "note";
const EmailType: SettingType = "email";
const ListType: SettingType = "list";

interface Meta {
    name: string;
    description: string;
    advanced?: boolean;
    disabled?: boolean;
    depends_true?: string;
    depends_false?: string;
    wiki_link?: string;
    aliases?: string[];
}

interface Setting {
    setting: string;
    name: string;
    description: string;
    required?: boolean;
    requires_restart?: boolean;
    advanced?: boolean;
    type: string;
    value: string | boolean | number | string[];
    depends_true?: string;
    depends_false?: string;
    wiki_link?: string;
    deprecated?: boolean;
    aliases?: string[];

    asElement: () => HTMLElement;
    update: (s: Setting) => void;

    hidden: boolean;

    valueAsString: () => string;
}

const splitDependant = (section: string, dep: string): string[] => {
    let parts = dep.split("|");
    if (parts.length == 1) {
        parts = [section, dep];
    }
    return parts;
};

let RestartRequiredBadge: HTMLElement;
let RequiredBadge: HTMLElement;

class DOMSetting {
    protected _hideEl: HTMLElement;
    protected _input: HTMLInputElement;
    protected _container: HTMLDivElement;
    protected _tooltip: HTMLDivElement;
    protected _required: HTMLSpanElement;
    protected _restart: HTMLSpanElement;
    protected _advanced: boolean;
    protected _section: string;
    protected _s: Setting;
    setting: string;

    get hidden(): boolean {
        return this._hideEl.classList.contains("unfocused");
    }
    set hidden(v: boolean) {
        if (v) {
            this._hideEl.classList.add("unfocused");
        } else {
            this._hideEl.classList.remove("unfocused");
        }
        document.dispatchEvent(changedEvent(this._section, this.setting, this.valueAsString(), v));
        console.log(`dispatched settings-${this._section}-${this.setting} = ${this.valueAsString()}/${v}`);
    }

    private _advancedListener = (event: advancedEvent) => {
        this.hidden = !event.detail;
    };

    get advanced(): boolean {
        return this._advanced;
    }
    set advanced(advanced: boolean) {
        this._advanced = advanced;
        if (advanced) {
            document.addEventListener("settings-advancedState", this._advancedListener);
        } else {
            document.removeEventListener("settings-advancedState", this._advancedListener);
        }
    }

    get name(): string {
        return this._container.querySelector("span.setting-label").textContent;
    }
    set name(n: string) {
        this._container.querySelector("span.setting-label").textContent = n;
    }

    get description(): string {
        return this._tooltip.querySelector("span.content").textContent;
    }
    set description(d: string) {
        const content = this._tooltip.querySelector("span.content") as HTMLSpanElement;
        content.textContent = d;
        if (d == "") {
            this._tooltip.classList.add("unfocused");
        } else {
            this._tooltip.classList.remove("unfocused");
        }
    }

    get required(): boolean {
        return !this._required.classList.contains("unfocused");
    }
    set required(state: boolean) {
        if (state) {
            this._required.classList.remove("unfocused");
            this._required.innerHTML = RequiredBadge.outerHTML;
        } else {
            this._required.classList.add("unfocused");
            this._required.textContent = ``;
        }
    }

    get requires_restart(): boolean {
        return !this._restart.classList.contains("unfocused");
    }
    set requires_restart(state: boolean) {
        if (state) {
            this._restart.classList.remove("unfocused");
            this._restart.innerHTML = RestartRequiredBadge.outerHTML;
        } else {
            this._restart.classList.add("unfocused");
            this._restart.textContent = ``;
        }
    }

    get depends_true(): string {
        return this._s.depends_true;
    }
    set depends_true(v: string) {
        this._s.depends_true = v;
        this._registerDependencies();
    }

    get depends_false(): string {
        return this._s.depends_false;
    }
    set depends_false(v: string) {
        this._s.depends_false = v;
        this._registerDependencies();
    }

    get aliases(): string[] {
        return this._s.aliases;
    }

    protected _registerDependencies() {
        // Doesn't re-register dependencies, but that isn't important in this application
        if (!(this._s.depends_true || this._s.depends_false)) return;
        let [sect, dependant] = splitDependant(this._section, this._s.depends_true || this._s.depends_false);
        let state = !Boolean(this._s.depends_false);
        document.addEventListener(`settings-${sect}-${dependant}`, (event: settingsChangedEvent) => {
            this.hidden = event.detail.hidden || toBool(event.detail.value) !== state;
        });
    }

    valueAsString = (): string => {
        return "" + this.value;
    };

    onValueChange = () => {
        document.dispatchEvent(changedEvent(this._section, this.setting, this.valueAsString(), this.hidden));
        const setEvent = new CustomEvent(`settings-set-${this._section}-${this.setting}`, {
            detail: this.valueAsString(),
        });
        document.dispatchEvent(setEvent);
        if (this.requires_restart) {
            document.dispatchEvent(new CustomEvent("settings-requires-restart"));
        }
    };

    constructor(input: string, setting: Setting, section: string, name: string, inputOnTop: boolean = false) {
        this._section = section;
        this.setting = name;
        this._container = document.createElement("div");
        this._container.classList.add("setting");
        this._container.setAttribute("data-name", name);
        this._container.innerHTML = `
        <label class="label flex flex-col gap-2">
            ${inputOnTop ? input : ""}
            <div class="flex flex-row gap-2 items-baseline">
                <span class="setting-label"></span>
                <div class="setting-tooltip tooltip right unfocused">
                    <i class="icon ri-information-line align-[-0.05rem]"></i>
                    <span class="content sm"></span>
                </div>
                <span class="setting-required unfocused"></span>
                <span class="setting-restart unfocused"></span>
            </div>
            ${inputOnTop ? "" : input}
        </label>
        `;
        this._tooltip = this._container.querySelector("div.setting-tooltip") as HTMLDivElement;
        this._required = this._container.querySelector("span.setting-required") as HTMLSpanElement;
        this._restart = this._container.querySelector("span.setting-restart") as HTMLSpanElement;
        // "input" variable should supply the HTML of an element with class "setting-input"
        this._input = this._container.querySelector(".setting-input") as HTMLInputElement;
        this._input.onchange = this.onValueChange;
        document.addEventListener(`settings-loaded`, this.onValueChange);
        this._hideEl = this._container;
    }

    get value(): any {
        return this._input.value;
    }
    set value(v: any) {
        this._input.value = v;
    }

    update(s: Setting) {
        this.name = s.name;
        this.description = s.description;
        this.required = s.required;
        this.requires_restart = s.requires_restart;
        this.value = s.value;
        this.advanced = s.advanced;
        if (!this._s || s.depends_true != this._s.depends_true || s.depends_false != this._s.depends_false) {
            this._s = s;
            this._registerDependencies();
        }
        this._s = s;
    }

    asElement = (): HTMLDivElement => {
        return this._container;
    };
}

class DOMInput extends DOMSetting {
    constructor(inputType: string, setting: Setting, section: string, name: string) {
        super(`<input type="${inputType}" class="input setting-input ~neutral @low">`, setting, section, name);
        // this._hideEl = this._input.parentElement;
        this.update(setting);
    }
}

interface SText extends Setting {
    value: string;
}
class DOMText extends DOMInput implements SText {
    constructor(setting: Setting, section: string, name: string) {
        super("text", setting, section, name);
    }
    type: SettingType = TextType;
    get value(): string {
        return this._input.value;
    }
    set value(v: string) {
        this._input.value = v;
    }
}

interface SPassword extends Setting {
    value: string;
}
class DOMPassword extends DOMInput implements SPassword {
    constructor(setting: Setting, section: string, name: string) {
        super("password", setting, section, name);
    }
    type: SettingType = PasswordType;
    get value(): string {
        return this._input.value;
    }
    set value(v: string) {
        this._input.value = v;
    }
}

interface SEmail extends Setting {
    value: string;
}
class DOMEmail extends DOMInput implements SEmail {
    constructor(setting: Setting, section: string, name: string) {
        super("email", setting, section, name);
    }
    type: SettingType = EmailType;
    get value(): string {
        return this._input.value;
    }
    set value(v: string) {
        this._input.value = v;
    }
}

interface SNumber extends Setting {
    value: number;
}
class DOMNumber extends DOMInput implements SNumber {
    constructor(setting: Setting, section: string, name: string) {
        super("number", setting, section, name);
    }
    type: SettingType = NumberType;
    get value(): number {
        return +this._input.value;
    }
    set value(v: number) {
        this._input.value = "" + v;
    }
}

interface SList extends Setting {
    value: string[];
}
class DOMList extends DOMSetting implements SList {
    protected _inputs: HTMLDivElement;
    type: SettingType = ListType;

    valueAsString = (): string => {
        return this.value.join("|");
    };

    get value(): string[] {
        let values = [];
        const inputs = this._input.querySelectorAll("input") as NodeListOf<HTMLInputElement>;
        for (let i in inputs) {
            if (inputs[i].value) values.push(inputs[i].value);
        }
        return values;
    }
    set value(v: string[]) {
        this._input.textContent = ``;
        for (let val of v) {
            let input = this.inputRow(val);
            this._input.appendChild(input);
        }
        const addDummy = () => {
            const dummyRow = this.inputRow();
            const input = dummyRow.querySelector("input") as HTMLInputElement;
            input.placeholder = window.lang.strings("add");
            const onDummyChange = () => {
                if (!input.value) return;
                addDummy();
                input.removeEventListener("change", onDummyChange);
                input.removeEventListener("keyup", onDummyChange);
                input.placeholder = ``;
            };
            input.addEventListener("change", onDummyChange);
            input.addEventListener("keyup", onDummyChange);
            this._input.appendChild(dummyRow);
        };
        addDummy();
    }

    private inputRow(v: string = ""): HTMLDivElement {
        let container = document.createElement("div") as HTMLDivElement;
        container.classList.add("flex", "flex-row", "justify-between");
        container.innerHTML = `
            <input type="text" class="input ~neutral @low">
            <button class="button ~neutral @low center inside-input rounded-s-none aria-label="${window.lang.strings("delete")}" title="${window.lang.strings("delete")}">
                <i class="ri-close-line"></i>
            </button>
        `;
        const input = container.querySelector("input") as HTMLInputElement;
        input.value = v;
        input.onchange = this.onValueChange;
        const removeRow = container.querySelector("button") as HTMLButtonElement;
        removeRow.onclick = () => {
            if (!container.nextElementSibling) return;
            container.remove();
            this.onValueChange();
        };
        return container;
    }

    constructor(setting: Setting, section: string, name: string) {
        super(`<div class="setting-input flex flex-col gap-2"></div>`, setting, section, name);
        // this._hideEl = this._input.parentElement;
        this.update(setting);
    }
}

interface SBool extends Setting {
    value: boolean;
}
class DOMBool extends DOMSetting implements SBool {
    type: SettingType = BoolType;

    get value(): boolean {
        return this._input.checked;
    }
    set value(state: boolean) {
        this._input.checked = state;
    }

    constructor(setting: SBool, section: string, name: string) {
        super(`<input type="checkbox" class="setting-input">`, setting, section, name, true);
        const label = this._container.getElementsByTagName("LABEL")[0];
        label.classList.remove("flex-col");
        label.classList.add("flex-row");
        // this._hideEl = this._input.parentElement;
        this.update(setting);
    }
}

interface SSelect extends Setting {
    options: string[][];
    value: string;
}
class DOMSelect extends DOMSetting implements SSelect {
    type: SettingType = SelectType;
    private _options: string[][];

    get options(): string[][] {
        return this._options;
    }
    set options(opt: string[][]) {
        this._options = opt;
        let innerHTML = "";
        for (let option of this._options) {
            innerHTML += `<option value="${option[0]}">${option[1]}</option>`;
        }
        this._input.innerHTML = innerHTML;
    }

    update(s: SSelect) {
        this.options = s.options;
        super.update(s);
    }

    constructor(setting: SSelect, section: string, name: string) {
        super(
            `<div class="select ~neutral @low">
                <select class="setting-select setting-input"></select>
            </div>`,
            setting,
            section,
            name,
        );
        this._options = [];
        // this._hideEl = this._container;
        this.update(setting);
    }
}

interface SNote extends Setting {
    value: string;
    style?: string;
}
class DOMNote extends DOMSetting implements SNote {
    private _nameEl: HTMLElement;
    private _description: HTMLElement;
    type: SettingType = NoteType;
    private _style: string;

    // We're a note, no one depends on us so we don't need to broadcast a state change.
    get hidden(): boolean {
        return this._container.classList.contains("unfocused");
    }
    set hidden(v: boolean) {
        if (v) {
            this._container.classList.add("unfocused");
        } else {
            this._container.classList.remove("unfocused");
        }
    }

    get name(): string {
        return this._nameEl.textContent;
    }
    set name(n: string) {
        this._nameEl.textContent = n;
    }

    get description(): string {
        return this._description.textContent;
    }
    set description(d: string) {
        this._description.innerHTML = d;
    }

    valueAsString = (): string => {
        return "";
    };

    get value(): string {
        return "";
    }
    set value(_: string) {
        return;
    }

    get required(): boolean {
        return false;
    }
    set required(_: boolean) {
        return;
    }

    get requires_restart(): boolean {
        return false;
    }
    set requires_restart(_: boolean) {
        return;
    }

    get style(): string {
        return this._style;
    }
    set style(s: string) {
        this._input.classList.remove("~" + this._style);
        this._style = s;
        this._input.classList.add("~" + this._style);
    }

    constructor(setting: SNote, section: string) {
        super(
            `
            <aside class="aside setting-input">
                <span class="font-bold setting-name"></span>
                <span class="content setting-description">
            </aside>
            `,
            setting,
            section,
            "",
        );
        // this._hideEl = this._container;
        this._nameEl = this._container.querySelector(".setting-name");
        this._description = this._container.querySelector(".setting-description");
        this.update(setting);
    }

    update(s: SNote) {
        this.name = s.name;
        this.description = s.description;
        this.style = "style" in s && s.style ? s.style : "info";
    }

    asElement = (): HTMLDivElement => {
        return this._container;
    };
}

interface Group {
    group: string;
    name: string;
    description: string;
    members: Member[];
}

abstract class groupableItem {
    protected _el: HTMLElement;
    asElement = () => {
        return this._el;
    };
    remove = () => {
        this._el.remove();
    };
    inGroup = (): string | null => {
        return this._el.parentElement.getAttribute("data-group");
    };
    get hidden(): boolean {
        return this._el.classList.contains("unfocused");
    }
    set hidden(v: boolean) {
        if (v) {
            this._el.classList.add("unfocused");
            if (this.inGroup()) {
                document.dispatchEvent(new CustomEvent(`settings-group-${this.inGroup()}-child-hidden`));
            }
        } else {
            this._el.classList.remove("unfocused");
            if (this.inGroup()) {
                document.dispatchEvent(new CustomEvent(`settings-group-${this.inGroup()}-child-visible`));
            }
        }
    }
}

class groupButton extends groupableItem {
    button: HTMLElement;
    private _dropdown: HTMLElement;
    private _icon: HTMLElement;
    private _check: HTMLInputElement;
    private _group: Group;
    private _indent: number;
    private _parentSidebar: HTMLElement;

    // one of the few sanctioned uses of ml/mr. Looks worse with ms/me.
    private static readonly _margin = "ml-6";
    private _indentClasses = ["h-11", "h-10", "h-9"];
    private _indentClass = () => {
        const classes = [["h-10"], ["h-9"]];
        return classes[Math.min(this.indent, classes.length - 1)];
    };

    asElement = () => {
        return this._el;
    };

    remove = () => {
        this._el.remove();
    };

    update = (g: Group) => {
        this._group = g;
        this.group = g.group;
        this.name = g.name;
        this.description = g.description;
    };

    append(item: HTMLElement | groupButton) {
        if (item instanceof groupButton) {
            item.button.classList.remove(...this._indentClasses);
            item.button.classList.add(...this._indentClass());
            this._dropdown.appendChild(item.asElement());
        } else {
            item.classList.remove(...this._indentClasses);
            item.classList.add(...this._indentClass());
            this._dropdown.appendChild(item);
        }
    }

    get name(): string {
        return this._group.name;
    }
    set name(v: string) {
        this._group.name = v;
        this.button.querySelector(".group-button-name").textContent = v;
    }

    get group(): string {
        return this._group.group;
    }
    set group(v: string) {
        document.removeEventListener(`settings-group-${this.group}-child-visible`, this._childVisible);
        document.removeEventListener(`settings-group-${this.group}-child-hidden`, this._childHidden);
        this._group.group = v;
        document.addEventListener(`settings-group-${this.group}-child-visible`, this._childVisible);
        document.addEventListener(`settings-group-${this.group}-child-hidden`, this._childHidden);
        this._el.setAttribute("data-group", v);
        this.button.setAttribute("data-group", v);
        this._check.setAttribute("data-group", v);
        this._dropdown.setAttribute("data-group", v);
    }

    get description(): string {
        return this._group.description;
    }
    set description(v: string) {
        this._group.description = v;
    }

    get indent(): number {
        return this._indent;
    }
    set indent(v: number) {
        this._dropdown.classList.remove(groupButton._margin);
        this._indent = v;
        this._dropdown.classList.add(groupButton._margin);
        for (let child of this._dropdown.children) {
            child.classList.remove(...this._indentClasses);
            child.classList.add(...this._indentClass());
        }
    }

    get open(): boolean {
        return this._check.checked;
    }
    set open(v: boolean) {
        this.openCloseWithAnimation(v);
    }

    openCloseWithAnimation(v: boolean) {
        this._check.checked = v;
        // When groups are nested, the outer group's scrollHeight will obviously change when an
        // inner group is opened/closed. Instead of traversing the tree and adjusting the maxHeight property
        // each open/close, just set the maxHeight to 9999px once the animation is completed.
        // On close, quickly set maxHeight back to ~scrollHeight, then animate to 0.
        if (this._check.checked) {
            this._icon.classList.add("rotated");
            this._icon.classList.remove("not-rotated");
            // Hide the scrollbar while we animate
            this._parentSidebar.style.overflowY = "hidden";
            this._dropdown.classList.remove("unfocused");
            const fullHeight = () => {
                this._dropdown.removeEventListener("transitionend", fullHeight);
                this._dropdown.style.maxHeight = "9999px";
                // Return the scrollbar (or whatever, just don't hide it)
                this._parentSidebar.style.overflowY = "";
            };
            this._dropdown.addEventListener("transitionend", fullHeight);
            this._dropdown.style.maxHeight = 1.2 * this._dropdown.scrollHeight + "px";
            this._dropdown.style.opacity = "100%";
        } else {
            this._icon.classList.add("not-rotated");
            this._icon.classList.remove("rotated");
            const mainTransitionEnd = () => {
                this._dropdown.removeEventListener("transitionend", mainTransitionEnd);
                this._dropdown.classList.add("unfocused");
                // Return the scrollbar (or whatever, just don't hide it)
                this._parentSidebar.style.overflowY = "";
            };
            const mainTransitionStart = () => {
                this._dropdown.removeEventListener("transitionend", mainTransitionStart);
                this._dropdown.style.transitionDuration = "";
                this._dropdown.addEventListener("transitionend", mainTransitionEnd);
                this._dropdown.style.maxHeight = "0";
                this._dropdown.style.opacity = "0";
            };
            // Hide the scrollbar while we animate
            this._parentSidebar.style.overflowY = "hidden";
            // Disabling transitions then going from 9999 - scrollHeight doesn't work in firefox to me,
            // so instead just make the transition duration really short.
            this._dropdown.style.transitionDuration = "1ms";
            this._dropdown.addEventListener("transitionend", mainTransitionStart);
            this._dropdown.style.maxHeight = 1.2 * this._dropdown.scrollHeight + "px";
        }
    }

    openCloseWithoutAnimation(v: boolean) {
        this._check.checked = v;
        if (this._check.checked) {
            this._icon.classList.add("rotated");
            this._dropdown.style.maxHeight = "9999px";
            this._dropdown.style.opacity = "100%";
            this._dropdown.classList.remove("unfocused");
        } else {
            this._icon.classList.remove("rotated");
            this._dropdown.style.maxHeight = "0";
            this._dropdown.style.opacity = "0";
            this._dropdown.classList.add("unfocused");
        }
    }

    private _childVisible = () => {
        this.hidden = false;
    };

    private _childHidden = () => {
        for (let el of this._dropdown.children) {
            if (!el.classList.contains("unfocused")) {
                return;
            }
        }
        // All children are hidden, so hide ourself
        this.hidden = true;
    };

    // Takes sidebar as we need to disable scrolling on it when animation starts.
    constructor(parentSidebar: HTMLElement) {
        super();
        this._parentSidebar = parentSidebar;

        this._el = document.createElement("div");
        this._el.classList.add("flex", "flex-col", "gap-2");

        this.button = document.createElement("span") as HTMLSpanElement;
        this._el.appendChild(this.button);
        this.button.classList.add("button", "~neutral", "@low", "settings-section-button", "h-11", "justify-between");
        this.button.innerHTML = `
        <span class="group-button-name"></span>
        <label class="button border-none shadow-none">
            <i class="icon ri-arrow-down-s-line"></i>
            <input class="unfocused" type="checkbox">
        </label>
        `;

        this._dropdown = document.createElement("div") as HTMLDivElement;
        this._el.appendChild(this._dropdown);
        this._dropdown.style.maxHeight = "0";
        this._dropdown.style.opacity = "0";
        this._dropdown.classList.add("settings-dropdown", "unfocused", "flex", "flex-col", "gap-2", "transition-all");

        this._icon = this.button.querySelector("i.icon");
        this._check = this.button.querySelector("input[type=checkbox]") as HTMLInputElement;

        this.button.onclick = (event: Event) => {
            if (event.target != this._icon && event.target != this._check) this.open = !this.open;
        };
        this._check.onclick = () => {
            this.open = this.open;
        };

        this.openCloseWithoutAnimation(false);
    }
}

interface Section {
    section: string;
    meta: Meta;
    settings: Setting[];
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
        this._section.classList.add("settings-section", "unfocused", "flex", "flex-col", "gap-2");
        this._section.setAttribute("data-section", sectionName);
        let innerHTML = `
        <div class="flex flex-row justify-between">
            <span class="heading">${s.meta.name}</span>
        `;
        if (s.meta.wiki_link) {
            innerHTML += `<a class="button ~urge dark:~d_info @low justify-center" target="_blank" href="${s.meta.wiki_link}" title="${window.lang.strings("wiki")}"><i class="ri-book-shelf-line"></i></a>`;
        }

        innerHTML += `
        </div>
        <p class="support lg settings-section-description">${s.meta.description}</p>
        `;
        this._section.innerHTML = innerHTML;

        this.update(s);
    }
    update = (s: Section) => {
        for (let setting of s.settings) {
            if (setting.setting in this._settings) {
                this._settings[setting.setting].update(setting);
            } else {
                if (setting.deprecated) continue;
                switch (setting.type) {
                    case TextType:
                        setting = new DOMText(setting, this._sectionName, setting.setting);
                        break;
                    case PasswordType:
                        setting = new DOMPassword(setting, this._sectionName, setting.setting);
                        break;
                    case EmailType:
                        setting = new DOMEmail(setting, this._sectionName, setting.setting);
                        break;
                    case NumberType:
                        setting = new DOMNumber(setting, this._sectionName, setting.setting);
                        break;
                    case BoolType:
                        setting = new DOMBool(setting as SBool, this._sectionName, setting.setting);
                        break;
                    case SelectType:
                        setting = new DOMSelect(setting as SSelect, this._sectionName, setting.setting);
                        break;
                    case NoteType:
                        setting = new DOMNote(setting as SNote, this._sectionName);
                        break;
                    case ListType:
                        setting = new DOMList(setting as SList, this._sectionName, setting.setting);
                        break;
                }
                if (setting.type != "note") {
                    this.values[setting.setting] = "" + setting.value;
                    // settings-section-name: Implies the setting changed or was shown/hidden.
                    // settings-set-section-name: Implies the setting changed.
                    document.addEventListener(
                        `settings-set-${this._sectionName}-${setting.setting}`,
                        (event: CustomEvent) => {
                            // const oldValue = this.values[name];
                            this.values[setting.setting] = event.detail;
                            document.dispatchEvent(new CustomEvent("settings-section-changed"));
                        },
                    );
                }
                this._section.appendChild(setting.asElement());
                this._settings[setting.setting] = setting;
            }
        }
    };

    get visible(): boolean {
        return !this._section.classList.contains("unfocused");
    }
    set visible(s: boolean) {
        if (s) {
            this._section.classList.remove("unfocused");
        } else {
            this._section.classList.add("unfocused");
        }
    }

    asElement = (): HTMLDivElement => {
        return this._section;
    };
}

type Member = { group: string } | { section: string };

class sectionButton extends groupableItem {
    section: string;
    private _name: HTMLElement;
    private _subButton: HTMLElement;
    private _meta: Meta;

    update = (section: string, sm: Meta) => {
        this.section = section;
        this._meta = sm;
        this.name = sm.name;
        this.advanced = sm.advanced;
        this._registerDependencies();
    };

    get subButton(): HTMLElement {
        return this._subButton.children[0] as HTMLElement;
    }
    set subButton(v: HTMLElement) {
        this._subButton.replaceChildren(v);
    }

    get name(): string {
        return this._meta.name;
    }
    set name(v: string) {
        this._meta.name = v;
        this._name.textContent = v;
    }

    get depends_true(): string {
        return this._meta.depends_true;
    }
    set depends_true(v: string) {
        this._meta.depends_true = v;
        this._registerDependencies();
    }

    get depends_false(): string {
        return this._meta.depends_false;
    }
    set depends_false(v: string) {
        this._meta.depends_false = v;
        this._registerDependencies();
    }

    get selected(): boolean {
        return this._el.classList.contains("selected");
    }
    set selected(v: boolean) {
        if (v) this._el.classList.add("selected");
        else this._el.classList.remove("selected");
    }

    select = () => {
        document.dispatchEvent(new CustomEvent("settings-show-panel", { detail: this.section }));
    };

    private _registerDependencies() {
        // Doesn't re-register dependencies, but that isn't important in this application
        if (!(this._meta.depends_true || this._meta.depends_false)) return;

        let [sect, dependant] = splitDependant(this.section, this._meta.depends_true || this._meta.depends_false);
        let state = !Boolean(this._meta.depends_false);
        document.addEventListener(`settings-${sect}-${dependant}`, (event: settingsChangedEvent) => {
            console.log(
                `recieved settings-${sect}-${dependant} = ${event.detail.value} = ${toBool(event.detail.value)} / ${event.detail.hidden}`,
            );
            const hide = event.detail.hidden || toBool(event.detail.value) !== state;
            this.hidden = hide;
            document.dispatchEvent(new CustomEvent(`settings-${name}`, { detail: !hide }));
        });
        document.addEventListener(`settings-${sect}`, (event: settingsChangedEvent) => {
            if (event.detail.hidden || toBool(event.detail.value) !== state) {
                this.hidden = true;
                document.dispatchEvent(new CustomEvent(`settings-${name}`, { detail: false }));
            }
        });
    }

    private _advancedListener = (event: advancedEvent) => {
        if (!event.detail) {
            this._el.classList.add("unfocused");
        } else {
            this._el.classList.remove("unfocused");
        }
        document.dispatchEvent(new CustomEvent("settings-re-search"));
    };

    get advanced(): boolean {
        return this._meta.advanced;
    }
    set advanced(v: boolean) {
        this._meta.advanced = v;
        if (v) document.addEventListener("settings-advancedState", this._advancedListener);
        else document.removeEventListener("settings-advancedState", this._advancedListener);
    }

    constructor(section?: string, sectionMeta?: Meta) {
        super();
        this._el = document.createElement("span") as HTMLSpanElement;
        this._el.classList.add("button", "~neutral", "@low", "settings-section-button", "h-11", "justify-between");
        this._el.innerHTML = `
            <span class="settings-section-button-name"></span>
            <div class="settings-section-button-sub-button"></div>
        `;
        this._name = this._el.getElementsByClassName("settings-section-button-name")[0] as HTMLElement;
        this._subButton = this._el.getElementsByClassName("settings-section-button-sub-button")[0] as HTMLElement;

        this._el.onclick = this.select;

        if (sectionMeta) this.update(section, sectionMeta);
    }
}

interface Settings {
    groups: Group[];
    sections: Section[];
    order?: Member[];
}

export class settingsList {
    private _saveButton = document.getElementById("settings-save") as HTMLSpanElement;
    private _saveNoRestart = document.getElementById("settings-apply-no-restart") as HTMLSpanElement;
    private _saveRestart = document.getElementById("settings-apply-restart") as HTMLSpanElement;

    private _loader = document.getElementById("settings-loader") as HTMLDivElement;

    private _panel = document.getElementById("settings-panel") as HTMLDivElement;
    private _sidebar = document.getElementById("settings-sidebar-items") as HTMLDivElement;
    private _visibleSection: string;
    private _sections: { [name: string]: sectionPanel };
    private _buttons: { [name: string]: sectionButton };

    private _groups: { [name: string]: Group };
    private _groupButtons: { [name: string]: groupButton };

    private _needsRestart: boolean = false;
    private _messageEditor = new MessageEditor();
    private _settings: Settings;
    private _advanced: boolean = false;

    private _searchbox = document.getElementById("settings-search") as HTMLInputElement;
    private _clearSearchboxButtons = Array.from(
        document.getElementsByClassName("settings-search-clear"),
    ) as Array<HTMLButtonElement>;

    private _noResultsPanel: HTMLElement = document.getElementById("settings-not-found");

    private _backupSortDirection = document.getElementById("settings-backups-sort-direction") as HTMLButtonElement;
    private _backupSortAscending = true;

    private _tasksList: TasksList;
    private _tasksButton = document.getElementById("settings-tasks") as HTMLButtonElement;

    // Must be called -after- all section have been added.
    // Takes all groups at once since members might contain each other.
    addGroups = (groups: Group[]) => {
        groups.forEach((g) => {
            this._groups[g.group] = g;
        });
        const addGroup = (g: Group, indent: number = 0): groupButton => {
            if (g.group in this._groupButtons) return null;

            const container = new groupButton(this._sidebar);
            container.update(g);
            container.indent = indent;

            for (const member of g.members) {
                if ("group" in member) {
                    let subgroup = addGroup(this._groups[member.group], indent + 1);
                    if (!subgroup) {
                        subgroup = this._groupButtons[member.group];
                        // Remove from page
                        subgroup.remove();
                    }
                    container.append(subgroup);
                } else if ("section" in member) {
                    const subsection = this._buttons[member.section];
                    // Remove from page
                    subsection.remove();
                    container.append(subsection.asElement());
                }
            }

            this._groupButtons[g.group] = container;
            return container;
        };
        for (let g of groups) {
            const container = addGroup(g);
            if (container) {
                this._sidebar.appendChild(container.asElement());
                container.openCloseWithoutAnimation(false);
            }
        }
    };

    addSection = (name: string, s: Section, subButton?: HTMLElement) => {
        const section = new sectionPanel(s, name);
        this._sections[name] = section;
        this._panel.appendChild(this._sections[name].asElement());
        const button = new sectionButton(name, s.meta);
        if (subButton) button.subButton = subButton;
        this._buttons[name] = button;
        this._sidebar.appendChild(button.asElement());
    };

    private _traverseMemberList = (list: Member[], func: (sect: string) => void) => {
        for (const member of list) {
            if ("group" in member) {
                for (const group of this._settings.groups) {
                    if (group.group == member.group) {
                        this._traverseMemberList(group.members, func);
                        break;
                    }
                }
            } else {
                func(member.section);
            }
        }
    };

    setUIOrder(order: Member[]) {
        this._sidebar.textContent = ``;
        for (const member of order) {
            if ("group" in member) {
                this._sidebar.appendChild(this._groupButtons[member.group].asElement());
                this._groupButtons[member.group].openCloseWithoutAnimation(false);
            } else if ("section" in member) {
                if (member.section in this._buttons) {
                    this._sidebar.appendChild(this._buttons[member.section].asElement());
                } else {
                    console.warn("Settings section specified in order but missing:", member.section);
                }
            }
        }
    }

    private _showPanel = (name: string) => {
        for (let n in this._sections) {
            this._sections[n].visible = n == name;
            this._buttons[name].selected = n == name;
            if (n == name) {
                this._visibleSection = name;
            }
        }
    };

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
            };
            window.modals.settingsRestart.show();
        } else {
            this._send(config);
        }
        // console.log(config);
    };

    private _send = (config: Object, run?: () => void) =>
        _post("/config", config, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 200 || req.status == 204) {
                    window.notifications.customSuccess("settingsSaved", window.lang.notif("saveSettings"));
                } else {
                    window.notifications.customError("settingsSaved", window.lang.notif("errorSaveSettings"));
                }
                this.reload();
                if (run) {
                    run();
                }
            }
        });

    private _showLogs = () =>
        _get("/logs", null, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && req.status == 200) {
                (document.getElementById("log-area") as HTMLPreElement).textContent = req.response["log"] as string;
                window.modals.logs.show();
            }
        });

    setBackupSort = (ascending: boolean) => {
        this._backupSortAscending = ascending;
        this._backupSortDirection.innerHTML = `${window.lang.strings("sortDirection")} <i class="${ascending ? "ri-arrow-up-s-line" : "ri-arrow-down-s-line"}"></i>`;
        this._getBackups();
    };

    private _backup = () =>
        _post(
            "/backups",
            null,
            (req: XMLHttpRequest) => {
                if (req.readyState != 4 || req.status != 200) return;
                const backupDTO = req.response as BackupDTO;
                if (backupDTO.path == "") {
                    window.notifications.customError("backupError", window.lang.strings("errorFailureCheckLogs"));
                    return;
                }
                const location = document.getElementById("settings-backed-up-location");
                const download = document.getElementById("settings-backed-up-download");
                location.innerHTML = window.lang
                    .strings("backupCanBeFound")
                    .replace(
                        "{filepath}",
                        `<span class="text-black dark:text-white font-mono bg-inherit">"` + backupDTO.path + `"</span>`,
                    );
                download.innerHTML = `
        <i class="ri-download-line"></i>
        <span>${window.lang.strings("download")}</span>
        <span class="badge ~info @low">${backupDTO.size}</span>
        `;

                download.parentElement.onclick = () => _download("/backups/" + backupDTO.name, backupDTO.name);
                window.modals.backedUp.show();
            },
            true,
        );

    private _getBackups = () =>
        _get("/backups", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4 || req.status != 200) return;
            const backups = req.response["backups"] as BackupDTO[];
            const table = document.getElementById("backups-list");
            table.textContent = ``;
            if (!this._backupSortAscending) {
                backups.reverse();
            }
            for (let b of backups) {
                const tr = document.createElement("tr") as HTMLTableRowElement;
                tr.classList.add("align-middle");
                tr.innerHTML = `
            <td class="whitespace-nowrap"><span class="text-black dark:text-white font-mono bg-inherit">${b.name}</span> <button class="backup-copy m-2"></button></td>
            <td>${toDateString(new Date(b.date * 1000))}</td>
            <td class="font-mono">${b.commit || "?"}</td>
            <td><div class="flex flex-row gap-2 items-stretch justify-center">
                <span class="backup-download button ~positive @low flex flex-row gap-2" title="${window.lang.strings("backupDownload")}">
                    <i class="ri-download-line"></i>
                    <span class="badge ~positive @low">${b.size}</span>
                </span>
                <span class="backup-restore button ~critical @low" title="${window.lang.strings("backupRestore")}"><i class="icon ri-restart-line"></i></span>
            </div></td>
            `;
                SetupCopyButton(tr.querySelector(".backup-copy"), b.path, null, window.lang.notif("pathCopied"));
                tr.querySelector(".backup-download").addEventListener("click", () =>
                    _download("/backups/" + b.name, b.name),
                );
                tr.querySelector(".backup-restore").addEventListener("click", () => {
                    _post("/backups/restore/" + b.name, null, () => {});
                    window.modals.backups.close();
                    window.modals.settingsRefresh.modal.querySelector("span.heading").textContent =
                        window.lang.strings("settingsRestarting");
                    window.modals.settingsRefresh.show();
                });
                table.appendChild(tr);
            }
        });

    constructor() {
        this._groups = {};
        this._groupButtons = {};
        this._sections = {};
        this._buttons = {};
        document.addEventListener("settings-section-changed", () => this._saveButton.classList.remove("unfocused"));
        document.getElementById("settings-restart").onclick = () => {
            _post("/restart", null, () => {});
            window.modals.settingsRefresh.modal.querySelector("span.heading").textContent =
                window.lang.strings("settingsRestarting");
            window.modals.settingsRefresh.show();
        };
        this._saveButton.onclick = this._save;
        document.addEventListener("settings-requires-restart", () => {
            this._needsRestart = true;
        });
        document.getElementById("settings-logs").onclick = this._showLogs;
        document.getElementById("settings-backups-backup").onclick = () => {
            window.modals.backups.close();
            this._backup();
        };

        this._tasksList = new TasksList();
        this._tasksButton.onclick = this._tasksList.load;

        document.addEventListener("settings-show-panel", (event: CustomEvent) => {
            this._showPanel(event.detail as string);
        });

        document.getElementById("settings-backups").onclick = () => {
            this.setBackupSort(this._backupSortAscending);
            window.modals.backups.show();
        };
        this._backupSortDirection.onclick = () => this.setBackupSort(!this._backupSortAscending);
        const advancedEnableToggle = document.getElementById("settings-advanced-enabled") as HTMLInputElement;

        const filedlg = document.getElementById("backups-file") as HTMLInputElement;
        document.getElementById("settings-backups-upload").onclick = () => {
            filedlg.click();
        };
        filedlg.addEventListener("change", () => {
            if (filedlg.files.length == 0) return;
            const form = new FormData();
            form.append("backups-file", filedlg.files[0], filedlg.files[0].name);
            _upload("/backups/restore", form);
            window.modals.backups.close();
            window.modals.settingsRefresh.modal.querySelector("span.heading").textContent =
                window.lang.strings("settingsRestarting");
            window.modals.settingsRefresh.show();
        });

        advancedEnableToggle.onchange = () => {
            document.dispatchEvent(new CustomEvent("settings-advancedState", { detail: advancedEnableToggle.checked }));
        };
        document.addEventListener("settings-advancedState", () => {
            const parent = advancedEnableToggle.parentElement;
            this._advanced = advancedEnableToggle.checked;
            if (this._advanced) {
                parent.classList.add("~urge");
                parent.classList.remove("~neutral");
                this._tasksButton.classList.remove("unfocused");
            } else {
                parent.classList.add("~neutral");
                parent.classList.remove("~urge");
                this._tasksButton.classList.add("unfocused");
            }
            this._searchbox.oninput(null);
        });
        advancedEnableToggle.checked = false;

        this._searchbox.oninput = () => {
            this.search(this._searchbox.value);
        };

        document.addEventListener("settings-re-search", () => {
            this._searchbox.oninput(null);
        });

        for (let b of this._clearSearchboxButtons) {
            b.onclick = () => {
                this._searchbox.value = "";
                this._searchbox.oninput(null);
            };
        }

        // Create (restart)required badges (can't do on load as window.lang is unset)
        RestartRequiredBadge = (() => {
            const rr = document.createElement("span");
            rr.classList.add("tooltip", "below", "force-ltr");
            rr.innerHTML = `
                <span class="badge ~info dark:~d_warning align-[0.08rem]"><i class="icon ri-refresh-line h-full"></i></span>
                <span class="content sm">${window.lang.strings("restartRequired")}</span>
            `;

            return rr;
        })();
        RequiredBadge = (() => {
            const r = document.createElement("span");
            r.classList.add("tooltip", "below", "force-ltr");
            r.innerHTML = `
                <span class="badge ~critical align-[0.08rem]"><i class="icon ri-asterisk h-full"></i></span>
                <span class="content sm">${window.lang.strings("required")}</span>
            `;

            return r;
        })();
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
                password: (document.getElementById("matrix-password") as HTMLInputElement).value,
            };
            _post(
                "/matrix/login",
                send,
                (req: XMLHttpRequest) => {
                    if (req.readyState == 4) {
                        removeLoader(button);
                        if (req.status == 400) {
                            window.notifications.customError(
                                "errorUnknown",
                                window.lang.notif(req.response["error"] as string),
                            );
                            return;
                        } else if (req.status == 401) {
                            window.notifications.customError("errorUnauthorized", req.response["error"] as string);
                            return;
                        } else if (req.status == 500) {
                            window.notifications.customError(
                                "errorAddMatrix",
                                window.lang.notif("errorFailureCheckLogs"),
                            );
                            return;
                        }
                        window.modals.matrix.close();
                        _post("/restart", null, () => {});
                        window.location.reload();
                    }
                },
                true,
            );
        };
        window.modals.matrix.show();
    };

    reload = () => {
        for (let i = 0; i < this._loader.children.length; i++) {
            this._loader.children[i].classList.add("invisible");
        }
        addLoader(this._loader, false, true);
        _get("/config", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                window.notifications.customError("settingsLoadError", window.lang.notif("errorLoadSettings"));
                return;
            }
            this._settings = req.response as Settings;
            for (let section of this._settings.sections) {
                if (section.meta.disabled) continue;
                if (section.section in this._sections) {
                    this._sections[section.section].update(section);
                } else {
                    if (section.section == "messages" || section.section == "user_page") {
                        const editButton = document.createElement("div");
                        editButton.classList.add("tooltip", "left", "h-full", "force-ltr");
                        editButton.innerHTML = `
                        <span class="button ~neutral @low h-full">
                            <i class="icon ri-edit-line"></i>
                        </span>
                        <span class="content sm">
                        ${window.lang.get("strings", "customizeMessages")}
                        </span>
                        `;
                        (editButton.querySelector("span.button") as HTMLSpanElement).onclick = () => {
                            this._messageEditor.showList(section.section == "messages" ? "email" : "user");
                        };
                        this.addSection(section.section, section, editButton);
                    } else if (section.section == "updates") {
                        const icon = document.createElement("span") as HTMLSpanElement;
                        if (window.updater.updateAvailable) {
                            icon.classList.add("button", "~urge");
                            icon.innerHTML = `<i class="ri-download-line" title="${window.lang.strings("update")}"></i>`;
                            icon.onclick = () => window.updater.checkForUpdates(window.modals.updateInfo.show);
                            // Put us first
                            if ("order" in this._settings && this._settings.order) {
                                let i = -1;
                                for (let j = 0; j < this._settings.order.length; j++) {
                                    const member = this._settings.order[j];
                                    if ("section" in member && member.section == "updates") {
                                        i = j;
                                        break;
                                    }
                                }
                                if (i != -1) {
                                    this._settings.order.splice(i, 1);
                                    this._settings.order.unshift({ section: "updates" });
                                }
                            }
                        }
                        this.addSection(section.section, section, icon);
                    } else if (section.section == "matrix" && !window.matrixEnabled) {
                        const addButton = document.createElement("div");
                        addButton.classList.add("tooltip", "left", "h-full", "force-ltr");
                        addButton.innerHTML = `
                        <span class="button ~neutral h-full"><i class="icon ri-links-line"></i></span>
                        <span class="content sm">
                        ${window.lang.strings("linkMatrix")}
                        </span>
                        `;
                        (addButton.querySelector("span.button") as HTMLSpanElement).onclick = this._addMatrix;
                        this.addSection(section.section, section, addButton);
                    } else {
                        this.addSection(section.section, section);
                    }
                }
            }

            this.addGroups(this._settings.groups);

            if ("order" in this._settings && this._settings.order) this.setUIOrder(this._settings.order);

            removeLoader(this._loader);
            for (let i = 0; i < this._loader.children.length; i++) {
                this._loader.children[i].classList.remove("invisible");
            }
            for (let s of this._settings.sections) {
                if (s.meta.disabled) continue;
                this._showPanel(s.section);
                break;
            }
            document.dispatchEvent(new CustomEvent("settings-loaded"));
            document.dispatchEvent(new CustomEvent("settings-advancedState", { detail: false }));
            this._saveButton.classList.add("unfocused");
            this._needsRestart = false;
        });
    };

    private _query: string;
    // FIXME: Fix searching groups
    // FIXME: Search "About" & "User profiles", pseudo-search "User profiles" for things like "Ombi", "Referrals", etc.
    search = (query: string) => {
        query = query.toLowerCase().trim();
        // Make sure a blank search is detected when there's just whitespace.
        if (query.replace(/\s+/g, "") == "") query = "";
        const noChange = query == this._query;

        let firstVisibleSection = "";

        // Close and hide all groups to start with
        for (const groupButton of Object.values(this._groupButtons)) {
            // Leave these opened/closed if the query didn't change
            // (this is overridden anyway if an actual search is happening,
            // so we'll only do it if the search is blank, implying something else
            // changed like advanced settings being enabled).
            if (noChange && query == "") continue;
            groupButton.openCloseWithoutAnimation(false);
            groupButton.hidden = !(
                groupButton.group.toLowerCase().includes(query) ||
                groupButton.name.toLowerCase().includes(query) ||
                groupButton.description.toLowerCase().includes(query)
            );
        }

        const searchSection = (section: Section) => {
            // Section might be disabled at build-time (like Updates), or deprecated and so not appear.
            if (!(section.section in this._sections)) {
                // console.log(`Couldn't find section "${section.section}"`);
                return;
            }
            const sectionElement = this._sections[section.section].asElement();
            let dependencyCard = sectionElement.querySelector(".settings-dependency-message");
            if (dependencyCard) dependencyCard.remove();
            dependencyCard = null;
            let dependencyList = null;

            // hide button, unhide if matched
            const button = this._buttons[section.section];
            button.hidden = true;
            const parentGroup = button.inGroup();
            let parentGroupButton: groupButton = null;
            let matchedGroup = false;
            if (parentGroup) {
                parentGroupButton = this._groupButtons[parentGroup];
                matchedGroup = !parentGroupButton.hidden;
            }

            const show = () => {
                button.hidden = false;
                if (parentGroupButton) {
                    if (query != "") parentGroupButton.openCloseWithoutAnimation(true);
                }
            };
            const hide = () => {
                button.hidden = true;
            };

            let matchedSection =
                matchedGroup ||
                section.section.toLowerCase().includes(query) ||
                section.meta.name.toLowerCase().includes(query) ||
                section.meta.description.toLowerCase().includes(query);
            if (section.meta.aliases)
                section.meta.aliases.forEach((term: string) => (matchedSection ||= term.toLowerCase().includes(query)));
            matchedSection &&= (section.meta.advanced && this._advanced) || !section.meta.advanced;

            if (matchedSection) {
                show();
                firstVisibleSection = firstVisibleSection || section.section;
            }

            for (let setting of section.settings) {
                if (setting.type == "note") continue;
                const element = sectionElement.querySelector(`div[data-name="${setting.setting}"]`) as HTMLElement;
                // Again, setting might be disabled at build-time (if we have such a mechanism) or deprecated (like the old duplicate 'url_base's)
                if (element == null) {
                    continue;
                }

                // If we match the whole section, don't bother searching settings.
                if (matchedSection) {
                    element.classList.remove("opacity-50", "pointer-events-none");
                    element.setAttribute("aria-disabled", "false");
                    continue;
                }

                element.classList.add("opacity-50", "pointer-events-none");
                element.setAttribute("aria-disabled", "true");
                let matchedSetting =
                    setting.setting.toLowerCase().includes(query) ||
                    setting.name.toLowerCase().includes(query) ||
                    setting.description.toLowerCase().includes(query) ||
                    String(setting.value).toLowerCase().includes(query);
                if (setting.aliases)
                    setting.aliases.forEach((term: string) => (matchedSetting ||= term.toLowerCase().includes(query)));
                if (matchedSetting) {
                    if ((section.meta.advanced && this._advanced) || !section.meta.advanced) {
                        show();
                        firstVisibleSection = firstVisibleSection || section.section;
                    }
                    const shouldShow = query != "" && ((setting.advanced && this._advanced) || !setting.advanced);
                    if (shouldShow || query == "") {
                        element.classList.remove("opacity-50", "pointer-events-none");
                        element.setAttribute("aria-disabled", "false");
                    }
                    if (
                        query != "" &&
                        ((shouldShow && element.querySelector("label").classList.contains("unfocused")) || !shouldShow)
                    ) {
                        // Add a note explaining why the setting is hidden
                        if (!dependencyCard) {
                            dependencyCard = document.createElement("aside");
                            dependencyCard.classList.add("aside", "my-2", "~warning", "settings-dependency-message");
                            dependencyCard.innerHTML = `
                            <div class="content text-sm">
                                <span class="font-bold">${window.lang.strings("settingsHiddenDependency")}</span>
                        
                                <ul class="settings-dependency-list"></ul>
                            </div>
                            `;
                            dependencyList = dependencyCard.querySelector(
                                ".settings-dependency-list",
                            ) as HTMLUListElement;
                            // Insert it right after the description
                            this._sections[section.section]
                                .asElement()
                                .insertBefore(
                                    dependencyCard,
                                    this._sections[section.section]
                                        .asElement()
                                        .querySelector(".settings-section-description").nextElementSibling,
                                );
                        }
                        const li = document.createElement("li");
                        if (shouldShow) {
                            const depCode = setting.depends_true || setting.depends_false;
                            const dep = splitDependant(section.section, depCode);

                            let depName = this._settings.sections[dep[0]].settings[dep[1]].name;
                            if (dep[0] != section.section) {
                                depName = this._settings.sections[dep[0]].meta.name + " > " + depName;
                            }

                            li.textContent = window.lang
                                .strings("settingsDependsOn")
                                .replace("{setting}", `"` + setting.name + `"`)
                                .replace("{dependency}", `"` + depName + `"`);
                        } else {
                            li.textContent = window.lang
                                .strings("settingsAdvancedMode")
                                .replace("{setting}", `"` + setting.name + `"`);
                        }
                        dependencyList.appendChild(li);
                    }
                }
            }
        };

        for (let section of this._settings.sections) {
            searchSection(section);
        }

        if (firstVisibleSection && (query != "" || this._visibleSection == "")) {
            this._buttons[firstVisibleSection].select();
            this._noResultsPanel.classList.add("unfocused");
        } else if (query != "") {
            this._noResultsPanel.classList.remove("unfocused");
            if (this._visibleSection) {
                this._sections[this._visibleSection].visible = false;
                this._buttons[this._visibleSection].selected = false;
                this._visibleSection = "";
            }
        }

        // We can use this later to tell if we should leave groups expanded/closed as they were.
        this._query = query;
    };
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
    description: string;
}

class MessageEditor {
    private _currentID: string;
    private _names: { [id: string]: emailListEl };
    private _content: string;
    private _templ: templateEmail;
    private _form = document.getElementById("form-editor") as HTMLFormElement;
    private _header = document.getElementById("header-editor") as HTMLSpanElement;
    private _aside = document.getElementById("aside-editor") as HTMLElement;
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
                this._aside.classList.add("unfocused");
                if (this._names[id].description != "") {
                    this._aside.textContent = this._names[id].description;
                    this._aside.classList.remove("unfocused");
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
                let innerHTML = "";
                for (let i = 0; i < this._templ.variables.length; i++) {
                    let ci = i % colors.length;
                    innerHTML += '<span class="button ~' + colors[ci] + ' @low"></span>';
                }
                if (this._templ.variables.length == 0) {
                    this._variablesLabel.classList.add("unfocused");
                } else {
                    this._variablesLabel.classList.remove("unfocused");
                }
                this._variables.innerHTML = innerHTML;
                let buttons = this._variables.querySelectorAll("span.button") as NodeListOf<HTMLSpanElement>;
                for (let i = 0; i < this._templ.variables.length; i++) {
                    buttons[i].innerHTML =
                        `<span class="font-mono bg-inherit">` + "{" + this._templ.variables[i] + "}" + `</span>`;
                    buttons[i].onclick = () => {
                        insertText(this._textArea, "{" + this._templ.variables[i] + "}");
                        this.loadPreview();
                        // this._timeout = setTimeout(this.loadPreview, this._finishInterval);
                    };
                }

                innerHTML = "";
                if (this._templ.conditionals == null || this._templ.conditionals.length == 0) {
                    this._conditionalsLabel.classList.add("unfocused");
                    this._conditionals.textContent = ``;
                } else {
                    for (let i = this._templ.conditionals.length - 1; i >= 0; i--) {
                        let ci = i % colors.length;
                        // FIXME: Store full color strings (with ~) so tailwind sees them.
                        innerHTML += '<span class="button ~' + colors[ci] + ' @low"></span>';
                    }
                    this._conditionalsLabel.classList.remove("unfocused");
                    this._conditionals.innerHTML = innerHTML;
                    buttons = this._conditionals.querySelectorAll("span.button") as NodeListOf<HTMLSpanElement>;
                    for (let i = 0; i < this._templ.conditionals.length; i++) {
                        buttons[i].innerHTML =
                            `<span class="font-mono bg-inherit">{if ` + this._templ.conditionals[i] + "}" + `</span>`;
                        buttons[i].onclick = () => {
                            insertText(this._textArea, "{if " + this._templ.conditionals[i] + "}" + "{endif}");
                            this.loadPreview();
                            // this._timeout = setTimeout(this.loadPreview, this._finishInterval);
                        };
                    }
                }
                window.modals.editor.show();
            }
        });
    };
    loadPreview = () => {
        let content = this._textArea.value;
        if (this._templ.variables) {
            for (let variable of this._templ.variables) {
                let value = this._templ.values[variable];
                if (value === undefined) {
                    value = "{" + variable + "}";
                }
                content = content.replace(new RegExp("{" + variable + "}", "g"), value);
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
    };

    showList = (filter?: string) => {
        _get(
            "/config/emails?lang=" + window.language + (filter ? "&filter=" + filter : ""),
            null,
            (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status != 200) {
                        window.notifications.customError(
                            "loadTemplateError",
                            window.lang.notif("errorFailureCheckLogs"),
                        );
                        return;
                    }
                    this._names = req.response;
                    const list = document.getElementById("customize-list") as HTMLDivElement;
                    list.textContent = "";
                    for (let id in this._names) {
                        const tr = document.createElement("tr") as HTMLTableRowElement;
                        let resetButton = ``;
                        if (this._names[id].enabled) {
                            resetButton = `<i class="icon ri-restart-line" title="${window.lang.get("strings", "reset")}"></i>`;
                        }
                        let innerHTML = `
                    <td>
                        ${this._names[id].name}
                    `;
                        if (this._names[id].description != "")
                            innerHTML += `
                        <div class="tooltip right">
                            <i class="icon ri-information-line"></i>
                            <span class="content sm">${this._names[id].description}</span>
                        </div>
                    `;
                        innerHTML += `
                    </td>
                    <td class="table-inline justify-center"><span class="customize-reset">${resetButton}</span></td>
                    <td><span class="button ~info @low" title="${window.lang.get("strings", "edit")}"><i class="icon ri-edit-line"></i></span></td>
                    `;
                        tr.innerHTML = innerHTML;
                        (tr.querySelector("span.button") as HTMLSpanElement).onclick = () => {
                            window.modals.customizeEmails.close();
                            this.loadEditor(id);
                        };
                        if (this._names[id].enabled) {
                            const rb = tr.querySelector("span.customize-reset") as HTMLElement;
                            rb.classList.add("button");
                            rb.onclick = () =>
                                _post("/config/emails/" + id + "/state/disable", null, (req: XMLHttpRequest) => {
                                    if (req.readyState == 4) {
                                        if (req.status != 200 && req.status != 204) {
                                            window.notifications.customError(
                                                "setEmailStateError",
                                                window.lang.notif("errorFailureCheckLogs"),
                                            );
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
            },
        );
    };

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
            event.preventDefault();
            if (this._textArea.value == this._content && this._names[this._currentID].enabled) {
                window.modals.editor.close();
                return;
            }
            _post("/config/emails/" + this._currentID, { content: this._textArea.value }, (req: XMLHttpRequest) => {
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

        const descriptions = document.getElementsByClassName(
            "editor-syntax-description",
        ) as HTMLCollectionOf<HTMLParagraphElement>;
        for (let el of descriptions) {
            el.innerHTML = window.lang.template("strings", "syntaxDescription", {
                variable: `<span class="font-mono font-bold">{varname}</span>`,
                ifTruth: `<span class="font-mono font-bold">{if address}Message sent to {address}{end}</span>`,
                ifCompare: `<span class="font-mono font-bold">{if profile == "Friends"}Friend{else if profile != "Admins"}User{end}</span>`,
            });
        }

        // Get rid of nasty CSS
        window.modals.editor.onclose = () => {
            this._preview.textContent = ``;
        };
    }
}

class TasksList {
    private _list: HTMLElement = document.getElementById("modal-tasks-list");

    load = () =>
        _get("/tasks", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) return;
            let resp = req.response["tasks"] as TaskDTO[];
            this._list.textContent = "";
            for (let t of resp) {
                const task = new Task(t);
                this._list.appendChild(task.asElement());
            }
            window.modals.tasks.show();
        });
}

interface TaskDTO {
    url: string;
    name: string;
    description: string;
}

class Task {
    private _el: HTMLElement;
    asElement = () => {
        return this._el;
    };
    constructor(t: TaskDTO) {
        this._el = document.createElement("div");
        this._el.classList.add("aside", "flex", "flex-row", "gap-4", "justify-between", "dark:shadow-md");
        this._el.innerHTML = `
        <div class="flex flex-col gap-1">
            <div class="flex flex-row gap-2 items-baseline w-max">
                <h2 class="heading text-2xl">${t.name}</h2>
                <span class="text-sm font-mono">${t.url}</span>
            </div>
            <p class="max-w-[40ch] wrap-break-word text-justify">${t.description}</p>
        </div>
        <button class="button ~urge @low p-6">${window.lang.strings("run")}</button>
        `;
        const button = this._el.querySelector("button") as HTMLButtonElement;
        button.onclick = () => {
            addLoader(button);
            _post(t.url, null, (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                removeLoader(button);
                setTimeout(window.modals.tasks.close, 1000);
                if (req.status != 204) {
                    window.notifications.customError("errorRunTask", window.lang.notif("errorFailureCheckLogs"));
                    return;
                }
                window.notifications.customSuccess("runTask", window.lang.notif("runTask"));
            });
        };
    }
}
