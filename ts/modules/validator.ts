interface valWindow extends Window {
    validationStrings: pwValStrings;
    invalidPassword: string;
    messages: { [key: string]: string };
}

interface pwValString {
    singular: string;
    plural: string;
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

export function initValidator(passwordField: HTMLInputElement, rePasswordField: HTMLInputElement, submitButton: HTMLInputElement, submitSpan: HTMLSpanElement, validatorFunc?: (oncomplete: (valid: boolean) => void) => void):  ({ [category: string]: Requirement }|(() => void))[] {
    var defaultPwValStrings: pwValStrings = {
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
    }

    const checkPasswords = () => {
        return passwordField.value == rePasswordField.value;
    }

    const checkValidity = () => {
        const pw = checkPasswords();
        validatorFunc((valid: boolean) => {
            if (pw && valid) {
                rePasswordField.setCustomValidity("");
                submitButton.disabled = false;
                submitSpan.removeAttribute("disabled");
            } else if (!pw) {
                rePasswordField.setCustomValidity(window.invalidPassword);
                submitButton.disabled = true;
                submitSpan.setAttribute("disabled", "");
            } else {
                rePasswordField.setCustomValidity("");
                submitButton.disabled = true;
                submitSpan.setAttribute("disabled", "");
            }
        });
    };

    rePasswordField.addEventListener("keyup", checkValidity);
    passwordField.addEventListener("keyup", checkValidity);


    // Incredible code right here
    const isInt = (s: string): boolean => { return (s in ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]); }

    const testStrings = (f: pwValString): boolean => {
        const testString = (s: string): boolean => {
            if (s == "" || !s.includes("{n}")) { return false; }
            return true;
        }
        return testString(f.singular) && testString(f.plural);
    }

    interface Validation { [name: string]: number }

    const validate = (s: string): Validation => {
        let v: Validation = {};
        for (let criteria of ["length", "lowercase", "uppercase", "number", "special"]) { v[criteria] = 0; }
        v["length"] = s.length;
        for (let c of s) {
            if (isInt(c)) { v["number"]++; }
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
    passwordField.addEventListener("keyup", () => {
        const v = validate(passwordField.value);
        for (let criteria in requirements) {
            requirements[criteria].validate(v[criteria]);
        }
    });

    var requirements: { [category: string]: Requirement } = {};

    if (!window.validationStrings) {
        window.validationStrings = defaultPwValStrings;
    } else {
        for (let category in window.validationStrings) {
            if (!testStrings(window.validationStrings[category])) {
                window.validationStrings[category] = defaultPwValStrings[category];
            }
            const el = document.getElementById("requirement-" + category);
            if (el) {
                requirements[category] = new Requirement(category, el as HTMLLIElement);
            }
        }
    }
    return [requirements, checkValidity]
}
