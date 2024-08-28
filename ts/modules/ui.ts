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
        <div class="flex flex-row gap-2 items-center">
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
