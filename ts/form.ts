import { serializeForm, _post, _get, _delete, addAttr, rmAttr } from "./modules/common.js";
import { BS5 } from "./modules/bs5.js";
import { BS4 } from "./modules/bs4.js";

interface formWindow extends Window {
    usernameEnabled: boolean;
    validationStrings: pwValStrings;
    checkPassword(): void;
    invalidPassword: string;
}

declare var window: formWindow;

interface pwValString {
    singular: string;
    plural: string;
}

interface pwValStrings {
    length, uppercase, lowercase, number, special: pwValString;
}

var defaultPwValStrings: pwValStrings = {
    length: {
        singular: "Must have at least {n} character",
        plural: "Must have a least {n} characters"
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

const toggleSpinner = (ogText?: string): string => {
    const submitButton = document.getElementById('submitButton') as HTMLButtonElement;
    if (document.getElementById('createAccountSpinner')) {
        submitButton.innerHTML = ogText ? ogText : `<span>Create Account</span>`;
        submitButton.disabled = false;
        return "";
    } else {
        let ogText = submitButton.innerHTML;
        submitButton.innerHTML = `
        <span id="createAccountSpinner" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>Creating...
        `;
        return ogText;
    }
};

for (let key in window.validationStrings) {
    if (window.validationStrings[key].singular == "" || !(window.validationStrings[key].plural.includes("{n}"))) {
        window.validationStrings[key].singular = defaultPwValStrings[key].singular;
    }
    if (window.validationStrings[key].plural == "" || !(window.validationStrings[key].plural.includes("{n}"))) {
        window.validationStrings[key].plural = defaultPwValStrings[key].plural;
    }
    let el = document.getElementById(key) as HTMLUListElement;
    if (el) {
        const min: number = +el.getAttribute("min");
        let text = "";
        if (min == 1) {
            text = window.validationStrings[key].singular.replace("{n}", "1");
        } else {
            text = window.validationStrings[key].plural.replace("{n}", min.toString());
        }
        (document.getElementById(key).children[0] as HTMLDivElement).textContent = text;
    }
}

window.BS = window.bs5 ? new BS5 : new BS4;
var successBox: BSModal = window.BS.newModal('successBox');;

var code = window.location.href.split('/').pop();

(document.getElementById('accountForm') as HTMLFormElement).addEventListener('submit', (event: any): boolean => {
    event.preventDefault();
    const el = document.getElementById('errorMessage');
    if (el) {
        el.remove();
    }
    const ogText = toggleSpinner();
    let send: Object = serializeForm('accountForm');
    send["code"] = code;
    if (!window.usernameEnabled) {
        send["email"] = send["username"];
    }
    _post("/newUser", send, function (): void {
        if (this.readyState == 4) {
            toggleSpinner(ogText);
            let data: Object = this.response;
            const errorGiven = ("error" in data)
            if (errorGiven || data["success"] === false) {
                let errorMessage = "Unknown Error";
                if (errorGiven && errorGiven != true) {
                    errorMessage = data["error"];
                }
                document.getElementById('errorBox').innerHTML += `
                <button id="errorMessage" class="btn btn-outline-danger" disabled>${errorMessage}</button>
                `;
            } else {
                let valid = true;
                for (let key in data) {
                    if (data.hasOwnProperty(key)) {
                        const criterion = document.getElementById(key);
                        if (criterion) {
                            if (data[key] === false) {
                                valid = false;
                                addAttr(criterion, "list-group-item-danger");
                                rmAttr(criterion, "list-group-item-success");
                            } else {
                                addAttr(criterion, "list-group-item-success");
                                rmAttr(criterion, "list-group-item-danger");
                            }
                        }
                    }
                }
                if (valid) {
                    successBox.show();
                }
            }
        }
    }, true);
    return false;
});

window.checkPassword = (): void => {
    const entry = document.getElementById('inputPassword') as HTMLInputElement;
    if (entry.value != "") {
        const reentry = document.getElementById('reInputPassword') as HTMLInputElement;
        const identical = (entry.value == reentry.value);
        const submitButton = document.getElementById('submitButton') as HTMLButtonElement;
        if (identical) {
            reentry.setCustomValidity('');
            rmAttr(submitButton, "btn-outline-danger");
            addAttr(submitButton, "btn-outline-primary");
        } else {
            reentry.setCustomValidity(window.invalidPassword);
            addAttr(submitButton, "btn-outline-danger");
            rmAttr(submitButton, "btn-outline-primary");
        }
    }
}
