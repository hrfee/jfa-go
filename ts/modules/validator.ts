interface valWindow extends Window {
    validationStrings: pwValStrings;
    invalidPassword: string;
    messages: { [key: string]: string };
}

interface pwValString {
    singular: string;
    plural: string;
}

export interface ValidatorRespDTO {
    response: boolean;
    error: string;
}

interface pwValStrings {
    length: pwValString;
    uppercase: pwValString;
    lowercase: pwValString;
    number: pwValString;
    special: pwValString;
    [ type: string ]: pwValString;
}

declare var window: valWindow;

class Requirement {
    private _name: string;
    protected _minCount: number;
    private _content: HTMLSpanElement;
    private _valid: HTMLSpanElement;
    private _li: HTMLLIElement;

    get valid(): boolean { return this._valid.classList.contains("~positive"); }
    set valid(state: boolean) {
        if (state) {
            this._valid.classList.add("~positive");
            this._valid.classList.remove("~critical");
            this._valid.innerHTML = `<i class="icon ri-check-line" title="valid"></i>`;
        } else {
            this._valid.classList.add("~critical");
            this._valid.classList.remove("~positive");
            this._valid.innerHTML = `<i class="icon ri-close-line" title="invalid"></i>`;
        }
    }

    constructor(name: string, el: HTMLLIElement) {
        this._name = name;
        this._li = el;
        this._content = this._li.querySelector("span.requirement-content") as HTMLSpanElement;
        this._valid = this._li.querySelector("span.requirement-valid") as HTMLSpanElement;
        this.valid = false;
        this._minCount = +this._li.getAttribute("min");

        let text = "";
        if (this._minCount == 1) {
            text = window.validationStrings[this._name].singular.replace("{n}", "1");
        } else {
            text = window.validationStrings[this._name].plural.replace("{n}", ""+this._minCount);
        }
        this._content.textContent = text;
    }

    validate = (count: number) => { this.valid = (count >= this._minCount); }
}

export interface ValidatorConf {
    passwordField: HTMLInputElement;
    rePasswordField: HTMLInputElement;
    submitInput?: HTMLInputElement;
    submitButton: HTMLSpanElement;
    validatorFunc?: (oncomplete: (valid: boolean) => void) => void;
}

export interface Validation { [name: string]: number }
export interface Requirements { [category: string]: Requirement };

export class Validator {
    private _conf: ValidatorConf;
    private _requirements: Requirements = {};
    private _defaultPwValStrings: pwValStrings = {
        length: {
            singular: "Must have at least {n} character",
            plural: "Must have at least {n} characters"
        },
        uppercase: {
            singular: "Must have at least {n} uppercase character",
            plural: "Must have at least {n} uppercase characters"
        },
        lowercase: {
            singular: "Must have at least {n} lowercase character",
            plural: "Must have at least {n} lowercase characters"
        },
        number: {
            singular: "Must have at least {n} number",
            plural: "Must have at least {n} numbers"
        },
        special: {
            singular: "Must have at least {n} special character",
            plural: "Must have at least {n} special characters"
        }
    };

    private _checkPasswords = () => {
        return this._conf.passwordField.value == this._conf.rePasswordField.value;
    }

    validate = () => {
        const pw = this._checkPasswords();
        this._conf.validatorFunc((valid: boolean) => {
            if (pw && valid) {
                this._conf.rePasswordField.setCustomValidity("");
                if (this._conf.submitInput) this._conf.submitInput.disabled = false;
                this._conf.submitButton.removeAttribute("disabled");
            } else if (!pw) {
                this._conf.rePasswordField.setCustomValidity(window.invalidPassword);
                if (this._conf.submitInput) this._conf.submitInput.disabled = true;
                this._conf.submitButton.setAttribute("disabled", "");
            } else {
                this._conf.rePasswordField.setCustomValidity("");
                if (this._conf.submitInput) this._conf.submitInput.disabled = true;
                this._conf.submitButton.setAttribute("disabled", "");
            }
        });
    };

    private _isInt = (s: string): boolean => { return (s >= '0' && s <= '9'); }
    
    private _testStrings = (f: pwValString): boolean => {
        const testString = (s: string): boolean => {
            if (s == "" || !s.includes("{n}")) { return false; }
            return true;
        }
        return testString(f.singular) && testString(f.plural);
    }

    private _validate = (s: string): Validation => {
        let v: Validation = {};
        for (let criteria of ["length", "lowercase", "uppercase", "number", "special"]) { v[criteria] = 0; }
        v["length"] = s.length;
        for (let c of s) {
            if (this._isInt(c)) { v["number"]++; }
            else {
                const upper = c.toUpperCase();
                if (upper == c.toLowerCase()) { v["special"]++; }
                else {
                    if (upper == c) { v["uppercase"]++; }
                    else if (upper != c) { v["lowercase"]++; }
                }
            }
        }
        return v
    }
    
    private _bindRequirements = () => {
        for (let category in window.validationStrings) {
            if (!this._testStrings(window.validationStrings[category])) {
                window.validationStrings[category] = this._defaultPwValStrings[category];
            }
            const el = document.getElementById("requirement-" + category);
            if (typeof(el) === 'undefined' || el == null) continue;
            this._requirements[category] = new Requirement(category, el as HTMLLIElement);
        }
    };

    get requirements(): Requirements { return this._requirements }; 

    constructor(conf: ValidatorConf) {
        this._conf = conf;
        if (!(this._conf.validatorFunc)) {
            this._conf.validatorFunc = (oncomplete: (valid: boolean) => void) => { oncomplete(true); };
        }
        this._conf.rePasswordField.addEventListener("keyup", this.validate);
        this._conf.passwordField.addEventListener("keyup", this.validate);
        this._conf.passwordField.addEventListener("keyup", () => {
            const v = this._validate(this._conf.passwordField.value);
            for (let criteria in this._requirements) {
                this._requirements[criteria].validate(v[criteria]);
            }
        });
        if (!window.validationStrings) {
            window.validationStrings = this._defaultPwValStrings;
        } else {
            this._bindRequirements();
        }
    }
}
