interface pwValString {
    singular: string;
    plural: string;
}

interface pwValStrings {
    length, uppercase, lowercase, number, special: pwValString;
}

const _post = (url: string, data: Object, onreadystatechange: () => void): void => {
    let req = new XMLHttpRequest();
    req.open("POST", url, true);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.responseType = 'json';
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
};

const toggleSpinner = (): void => {
    const submitButton = document.getElementById('submitButton') as HTMLButtonElement;
    if (document.getElementById('createAccountSpinner')) {
        submitButton.innerHTML = `<span>Create Account</span>`;
        submitButton.disabled = false;
    } else {
        submitButton.innerHTML = ` 
        <span id="createAccountSpinner" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>Creating...
        `;
    }
};

const rmAttr = (el: HTMLElement, attr: string): void => {
    if (el.classList.contains(attr)) {
        el.classList.remove(attr);
    }
};

const addAttr = (el: HTMLElement, attr: string): void => el.classList.add(attr);

var validationStrings: pwValStrings;
var bsVersion: number;

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

for (let key in validationStrings) {
    console.log(key);
    if (validationStrings[key].singular == "" || !(validationStrings[key].plural.includes("{n}"))) {
        validationStrings[key].singular = defaultPwValStrings[key].singular;
    }
    if (validationStrings[key].plural == "" || !(validationStrings[key].plural.includes("{n}"))) {
        validationStrings[key].plural = defaultPwValStrings[key].plural;
    }
    let el = document.getElementById(key) as HTMLUListElement;
    if (el) {
        const min: number = +el.getAttribute("min");
        let text = "";
        if (min == 1) {
            text = validationStrings[key].singular.replace("{n}", "1");
        } else {
            text = validationStrings[key].plural.replace("{n}", min.toString());
        }
        (document.getElementById(key).children[0] as HTMLDivElement).textContent = text;
    }
}

interface Modal {
    show: () => void;
    hide: () => void;
}

var successBox: Modal;

if (bsVersion == 5) {
    var bootstrap: any;
    successBox = new bootstrap.Modal(document.getElementById('successBox'));
} else if (bsVersion == 4) {
    successBox = {
        show: (): void => {
            ($('#successBox') as any).modal('show');
        },
        hide: (): void => {
            ($('#successBox') as any).modal('hide');
        }
    };
}

var code = window.location.href.split('/').pop();
var usernameEnabled: boolean;

(document.getElementById('accountForm') as HTMLFormElement).addEventListener('submit', (event: any): boolean => {
    event.preventDefault();
    const el = document.getElementById('errorMessage');
    if (el) {
        el.remove();
    }
    toggleSpinner();
    let send: Object = serializeForm('accountForm');
    send["code"] = code;
    if (!usernameEnabled) {
        send["email"] = send["username"];
    }
    _post("/newUser", send, function (): void {
        if (this.readyState == 4) {
            toggleSpinner();
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
    });
    return false;
});













