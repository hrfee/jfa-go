export interface HiddenInputConf {
    container: HTMLElement;
    onSet: () => void;
    buttonOnLeft?: boolean;
    customContainerHTML?: string;
    input?: string;
    clickAwayShouldSave?: boolean;
};

export class HiddenInputField {
    public static editClass = "ri-edit-line";
    public static saveClass = "ri-check-line";

    private _c: HiddenInputConf;
    private _input: HTMLInputElement;
    private _content: HTMLElement
    private _toggle: HTMLElement;

    previous: string;

    constructor(c: HiddenInputConf) {
        this._c = c;
        if (!(this._c.customContainerHTML)) {
            this._c.customContainerHTML = `<span class="hidden-input-content"></span>`;
        }
        if (!(this._c.input)) {
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

        this.setEditing(false, true);
    }

    // FIXME: not working
    outerClickListener = ((event: Event) => {
        if (!(event.target instanceof HTMLElement && (this._input.contains(event.target) || this._toggle.contains(event.target)))) {
            this.toggle(!(this._c.clickAwayShouldSave));
        }
    }).bind(this);

    get editing(): boolean  { return this._toggle.classList.contains(HiddenInputField.saveClass); }
    set editing(e: boolean) { this.setEditing(e); }

    setEditing(e: boolean, noEvent: boolean = false, noSave: boolean = false) {
        if (e) {
            document.addEventListener("click", this.outerClickListener);
            this.previous = this.value;
            this._input.value = this.value;
            this._toggle.classList.add(HiddenInputField.saveClass);
            this._toggle.classList.remove(HiddenInputField.editClass);
            this._input.classList.remove("hidden");
            this._content.classList.add("hidden");
        } else {
            document.removeEventListener("click", this.outerClickListener);
            this.value = noSave ? this.previous : this._input.value;
            this._toggle.classList.add(HiddenInputField.editClass);
            this._toggle.classList.remove(HiddenInputField.saveClass);
            // done by set value()
            // this._content.classList.remove("hidden");
            this._input.classList.add("hidden");
            if (this.value != this.previous && !noEvent && !noSave) this._c.onSet()
        }
    }

    get value(): string { return this._content.textContent; };
    set value(v: string) {
        this._content.textContent = v;
        this._input.value = v;
        if (!v) this._content.classList.add("hidden");
        else this._content.classList.remove("hidden");
    }

    toggle(noSave: boolean = false) { this.setEditing(!this.editing, false, noSave); }
}
/*
    * class GenericNumber<NumType> {
  zeroValue: NumType;
  add: (x: NumType, y: NumType) => NumType;
}
 
let myGenericNumber = new GenericNumber<number>();
myGenericNumber.zeroValue = 0;
myGenericNumber.add = function (x, y) {
  return x + y;
};*/

// Simple radio tabs, when each "button" is a label containing an input[type="radio"] and some element[class="button"].
export class SimpleRadioTabs {
    tabs: Array<HTMLElement>;
    radios: Array<HTMLInputElement>;


    // Pass nothing, or a list of the tab container elements and either the "name" field used on the radios, of a list of them.
    constructor(tabs?: Array<HTMLElement>, radios?: Array<HTMLInputElement>|string) {
        this.tabs = tabs || new Array<HTMLElement>();

        if (radios) {
            if (typeof radios === "string") {
                this.radios = Array.from(document.querySelectorAll(`input[name=${radios}]`)) as Array<HTMLInputElement>;
            } else {
                this.radios = radios as Array<HTMLInputElement>;
            }
            this.radios.forEach((radio) => { radio.onchange = this.onChange });
        }
    }

    onChange = () => {
        for (let i = 0; i < this.radios.length; i++) {
            const buttonEl = this.radios[i].nextElementSibling;
            if (this.radios[i].checked) {
                buttonEl.classList.add("@high");
                buttonEl.classList.remove("@low");
                this.tabs[i].classList.remove("unfocused");
            } else {
                buttonEl.classList.add("@low");
                buttonEl.classList.remove("@high");
                this.tabs[i].classList.add("unfocused");
            }
        }
    };

    select(i: number) {
        for (let j = 0; j < this.radios.length; j++) {
            this.radios[j].checked = i == j;
        }
        this.onChange();
    }

    push(tab: HTMLElement, radio: HTMLInputElement) {
        this.tabs.push(tab);
        radio.onchange = this.onChange;
        this.radios.push(radio);
    }
}
