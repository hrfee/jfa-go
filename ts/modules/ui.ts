declare var window: GlobalWindow;

export interface HiddenInputConf {
    container: HTMLElement;
    onSet: () => void;
    buttonOnLeft?: boolean;
    customContainerHTML?: string;
    input?: string;
    clickAwayShouldSave?: boolean;
}

export class HiddenInputField {
    public static editClass = "ri-edit-line";
    public static saveClass = "ri-check-line";

    private _c: HiddenInputConf;
    private _input: HTMLInputElement;
    private _content: HTMLElement;
    private _toggle: HTMLElement;

    previous: string;

    constructor(c: HiddenInputConf) {
        this._c = c;
        if (!this._c.customContainerHTML) {
            this._c.customContainerHTML = `<span class="hidden-input-content"></span>`;
        }
        if (!this._c.input) {
            this._c.input = `<input type="text" class="field ~neutral @low max-w-24 hidden-input-input">`;
        }
        this._c.container.innerHTML = `
        <div class="flex flex-row gap-2 items-baseline">
            ${this._c.buttonOnLeft ? "" : this._c.input}
            ${this._c.buttonOnLeft ? "" : this._c.customContainerHTML}
            <i class="hidden-input-toggle"></i>
            ${this._c.buttonOnLeft ? this._c.input : ""}
            ${this._c.buttonOnLeft ? this._c.customContainerHTML : ""}
        </div>
        `;

        this._input = this._c.container.querySelector(".hidden-input-input") as HTMLInputElement;
        this._input.classList.add("py-0.5", "px-1", "hidden");
        this._toggle = this._c.container.querySelector(".hidden-input-toggle");
        this._content = this._c.container.querySelector(".hidden-input-content");

        this._toggle.onclick = () => {
            this.editing = !this.editing;
        };
        this._input.addEventListener("keypress", (e: KeyboardEvent) => {
            if (e.key === "Enter") {
                e.preventDefault();
                this._toggle.click();
            }
        });

        this.setEditing(false, true);
    }

    // FIXME: not working
    outerClickListener = ((event: Event) => {
        if (
            !(
                event.target instanceof HTMLElement &&
                (this._input.contains(event.target) || this._toggle.contains(event.target))
            )
        ) {
            this.toggle(!this._c.clickAwayShouldSave);
        }
    }).bind(this);

    get editing(): boolean {
        return this._toggle.classList.contains(HiddenInputField.saveClass);
    }
    set editing(e: boolean) {
        this.setEditing(e);
    }

    setEditing(e: boolean, noEvent: boolean = false, noSave: boolean = false) {
        if (e) {
            document.addEventListener("click", this.outerClickListener);
            this.previous = this.value;
            this._input.value = this.value;
            this._toggle.classList.add(HiddenInputField.saveClass);
            this._toggle.classList.remove(HiddenInputField.editClass);
            this._input.classList.remove("hidden");
            this._input.focus();
            this._content.classList.add("hidden");
        } else {
            document.removeEventListener("click", this.outerClickListener);
            this.value = noSave ? this.previous : this._input.value;
            this._toggle.classList.add(HiddenInputField.editClass);
            this._toggle.classList.remove(HiddenInputField.saveClass);
            // done by set value()
            // this._content.classList.remove("hidden");
            this._input.classList.add("hidden");
            if (this.value != this.previous && !noEvent && !noSave) this._c.onSet();
        }
    }

    get value(): string {
        return this._content.textContent;
    }
    set value(v: string) {
        this._content.textContent = v;
        this._input.value = v;
        if (!v) this._content.classList.add("hidden");
        else this._content.classList.remove("hidden");
    }

    toggle(noSave: boolean = false) {
        this.setEditing(!this.editing, false, noSave);
    }
}

export interface RadioBasedTab {
    name: string;
    id?: string;
    // If passed, will be put inside the button instead of the name.
    buttonHTML?: string;
    // You must at least pass a content element or an onShow function.
    content?: HTMLElement;
    onShow?: () => void;
    onHide?: () => void;
}

interface RadioBasedTabItem {
    tab: RadioBasedTab;
    input: HTMLInputElement;
    button: HTMLElement;
}

export class RadioBasedTabSelector {
    private _id: string;
    private _container: HTMLElement;
    private _tabs: RadioBasedTabItem[];
    private _selected: string;
    constructor(container: HTMLElement, id: string, ...tabs: RadioBasedTab[]) {
        this._container = container;
        this._container.classList.add("flex", "flex-row", "gap-2");
        this._tabs = [];
        this._id = id;
        let i = 0;
        const frag = document.createDocumentFragment();
        for (let tab of tabs) {
            if (!tab.id) tab.id = tab.name;
            const label = document.createElement("label");
            label.classList.add("grow");
            label.innerHTML = `
                <input type="radio" name="${this._id}" value="${tab.name}" class="unfocused" ${i == 0 ? "checked" : ""}>
                <span class="button ~neutral ${i == 0 ? "@high" : "@low"} radio-tab-button supra w-full text-center">${tab.buttonHTML || tab.name}</span>
            `;
            let ft: RadioBasedTabItem = {
                tab: tab,
                input: label.getElementsByTagName("input")[0] as HTMLInputElement,
                button: label.getElementsByClassName("radio-tab-button")[0] as HTMLElement,
            };
            ft.input.onclick = () => {
                ft.input.checked = true;
                this.checkSource();
            };
            frag.appendChild(label);
            this._tabs.push(ft);

            i++;
        }
        this._container.replaceChildren(frag);
        this.selected = 0;
    }

    checkSource = () => {
        for (let tab of this._tabs) {
            if (tab.input.checked) {
                this._selected = tab.tab.id;
                tab.tab.content?.classList.remove("unfocused");
                tab.button.classList.add("@high");
                tab.button.classList.remove("@low");
                if (tab.tab.onShow) tab.tab.onShow();
            } else {
                tab.tab.content?.classList.add("unfocused");
                tab.button.classList.add("@low");
                tab.button.classList.remove("@high");
                if (tab.tab.onHide) tab.tab.onHide();
            }
        }
    };

    get selected(): string {
        return this._selected;
    }
    set selected(id: string | number) {
        if (typeof id !== "string") {
            id = this._tabs[id as number].tab.id;
        }
        for (let tab of this._tabs) {
            if (tab.tab.id == id) {
                this._selected = tab.tab.id;
                tab.input.checked = true;
                tab.tab.content?.classList.remove("unfocused");
                tab.button.classList.add("@high");
                tab.button.classList.remove("@low");
                if (tab.tab.onShow) tab.tab.onShow();
            } else {
                tab.input.checked = false;
                tab.tab.content?.classList.add("unfocused");
                tab.button.classList.add("@low");
                tab.button.classList.remove("@high");
                if (tab.tab.onHide) tab.tab.onHide();
            }
        }
    }
}

type TooltipPosition = "above" | "below" | "below-center" | "left" | "right";

export class Tooltip extends HTMLElement {
    private _content: HTMLElement;
    get content(): HTMLElement {
        if (!this._content) return this.getElementsByClassName("content")[0] as HTMLElement;
        return this._content;
    }

    connectedCallback() {
        this.setup();
    }

    get visible(): boolean {
        return this.classList.contains("shown");
    }

    get position(): TooltipPosition {
        return window.getComputedStyle(this).getPropertyValue("--tooltip-position").trim() as TooltipPosition;
    }

    toggle() {
        console.log("toggle!");
        this.visible ? this.close() : this.open();
    }

    clicked: boolean = false;

    private _listener = (event: MouseEvent | TouchEvent) => {
        if (event.target !== this && !this.contains(event.target as HTMLElement)) {
            this.close();
            document.removeEventListener("mousedown", this._listener);
            // document.removeEventListener("touchstart", this._listener);
        }
    };

    open() {
        this.fixWidth(() => {
            this.classList.add("shown");
            if (this.clicked) {
                document.addEventListener("mousedown", this._listener);
                // document.addEventListener("touchstart", this._listener);
            }
        });
    }

    close() {
        this.clicked = false;
        this.classList.remove("shown");
    }

    setup() {
        this._content = this.getElementsByClassName("content")[0] as HTMLElement;
        const clickEvent = () => {
            if (this.clicked) {
                console.log("clicked again!");
                this.toggle();
            } else {
                console.log("clicked!");
                this.clicked = true;
                this.open();
            }
        };
        /// this.addEventListener("touchstart", clickEvent);
        this.addEventListener("click", clickEvent);
        this.addEventListener("mouseover", () => {
            this.open();
        });
        this.addEventListener("mouseleave", () => {
            if (this.clicked) return;
            console.log("mouseleave");
            this.close();
        });
    }

    fixWidth(after?: () => void) {
        this._content.style.left = "";
        this._content.style.right = "";
        if (this.position == "below-center") {
            const offset = this.offsetLeft;
            const pw = (this.offsetParent as HTMLElement).offsetWidth;
            const cw = this._content.offsetWidth;
            const pos = -1 * offset + (pw - cw) / 2.0;
            this._content.style.left = pos + "px";
        }
        const [leftObscured, rightObscured] = wherePartiallyObscuredX(this._content);
        if (rightObscured) {
            const rect = this._content.getBoundingClientRect();
            this._content.style.left =
                "calc(-1rem + " + ((window.innerWidth || document.documentElement.clientHeight) - rect.right) + "px)";
        }
        if (leftObscured) {
            const rect = this._content.getBoundingClientRect();
            this._content.style.right = "calc(-1rem + " + rect.left + "px)";
            "calc(-1rem + " + ((window.innerWidth || document.documentElement.clientHeight) - rect.right) + "px)";
        }
        if (after) after();
    }
}

export function setupTooltips() {
    customElements.define("tool-tip", Tooltip);
}

export function isPartiallyObscuredX(el: HTMLElement): boolean {
    const rect = el.getBoundingClientRect();
    return rect.left < 0 || rect.right > (window.innerWidth || document.documentElement.clientWidth);
}

export function wherePartiallyObscuredX(el: HTMLElement): [boolean, boolean] {
    const rect = el.getBoundingClientRect();
    return [Boolean(rect.left < 0), Boolean(rect.right > (window.innerWidth || document.documentElement.clientWidth))];
}

export function isPartiallyObscuredY(el: HTMLElement): boolean {
    const rect = el.getBoundingClientRect();
    return rect.top < 0 || rect.bottom > (window.innerHeight || document.documentElement.clientHeight);
}

export function wherePartiallyObscuredY(el: HTMLElement): [boolean, boolean] {
    const rect = el.getBoundingClientRect();
    return [
        Boolean(rect.top < 0),
        Boolean(rect.bottom > (window.innerHeight || document.documentElement.clientHeight)),
    ];
}
