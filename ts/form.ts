import { Modal } from "./modules/modal.js";
import { notificationBox, whichAnimationEvent } from "./modules/common.js";
import { _get, _post, toggleLoader, toDateString } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";

interface formWindow extends Window {
    validationStrings: pwValStrings;
    invalidPassword: string;
    successModal: Modal;
    telegramModal: Modal;
    discordModal: Modal;
    confirmationModal: Modal
    code: string;
    messages: { [key: string]: string };
    confirmation: boolean;
    telegramRequired: boolean;
    telegramPIN: string;
    discordRequired: boolean;
    discordPIN: string;
    discordStartCommand: string;
    discordInviteLink: boolean;
    discordServerName: string;
    userExpiryEnabled: boolean;
    userExpiryMonths: number;
    userExpiryDays: number;
    userExpiryHours: number;
    userExpiryMinutes: number;
    userExpiryMessage: string;
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

loadLangSelector("form");

window.notifications = new notificationBox(document.getElementById("notification-box") as HTMLDivElement);

window.animationEvent = whichAnimationEvent();

window.successModal = new Modal(document.getElementById("modal-success"), true);

var telegramVerified = false;
if (window.telegramEnabled) {
    window.telegramModal = new Modal(document.getElementById("modal-telegram"), window.telegramRequired);
    const telegramButton = document.getElementById("link-telegram") as HTMLSpanElement;
    telegramButton.onclick = () => {
        const waiting = document.getElementById("telegram-waiting") as HTMLSpanElement;
        toggleLoader(waiting);
        window.telegramModal.show();
        let modalClosed = false;
        window.telegramModal.onclose = () => {
            modalClosed = true;
            toggleLoader(waiting);
        }
        const checkVerified = () => _get("/invite/" + window.code + "/telegram/verified/" + window.telegramPIN, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 401) {
                    window.telegramModal.close();
                    window.notifications.customError("invalidCodeError", window.messages["errorInvalidCode"]);
                    return;
                } else if (req.status == 200) {
                    if (req.response["success"] as boolean) {
                        telegramVerified = true;
                        waiting.classList.add("~positive");
                        waiting.classList.remove("~info");
                        window.notifications.customPositive("telegramVerified", "", window.messages["verified"]); 
                        setTimeout(window.telegramModal.close, 2000);
                        telegramButton.classList.add("unfocused");
                        document.getElementById("contact-via").classList.remove("unfocused");
                        const radio = document.getElementById("contact-via-telegram") as HTMLInputElement;
                        radio.checked = true;
                    } else if (!modalClosed) {
                        setTimeout(checkVerified, 1500);
                    }
                }
            }
        });
        checkVerified();
    };
}

interface DiscordInvite {
    invite: string;
    icon: string;
}

var discordVerified = false;
if (window.discordEnabled) {
    window.discordModal = new Modal(document.getElementById("modal-discord"), window.discordRequired);
    const discordButton = document.getElementById("link-discord") as HTMLSpanElement;
    if (window.discordInviteLink) {
        _get("/invite/" + window.code + "/discord/invite", null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    return;
                }
                const inv = req.response as DiscordInvite;
                const link = document.getElementById("discord-invite") as HTMLAnchorElement;
                link.classList.add("subheading", "link-center");
                link.href = inv.invite;
                link.target = "_blank";
                link.innerHTML = `<span class="img-circle lg mr-1"><img class="img-circle" src="${inv.icon}" width="64" height="64"></span>${window.discordServerName}`;
            }
        });
    }
    discordButton.onclick = () => {
        const waiting = document.getElementById("discord-waiting") as HTMLSpanElement;
        toggleLoader(waiting);
        window.discordModal.show();
        let modalClosed = false;
        window.discordModal.onclose = () => {
            modalClosed = true;
            toggleLoader(waiting);
        }
        const checkVerified = () => _get("/invite/" + window.code + "/discord/verified/" + window.discordPIN, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 401) {
                    window.discordModal.close();
                    window.notifications.customError("invalidCodeError", window.messages["errorInvalidCode"]);
                    return;
                } else if (req.status == 200) {
                    if (req.response["success"] as boolean) {
                        discordVerified = true;
                        waiting.classList.add("~positive");
                        waiting.classList.remove("~info");
                        window.notifications.customPositive("discordVerified", "", window.messages["verified"]); 
                        setTimeout(window.discordModal.close, 2000);
                        discordButton.classList.add("unfocused");
                        document.getElementById("contact-via").classList.remove("unfocused");
                        const radio = document.getElementById("contact-via-discord") as HTMLInputElement;
                        radio.checked = true;
                    } else if (!modalClosed) {
                        setTimeout(checkVerified, 1500);
                    }
                }
            }
        });
        checkVerified();
    };
}

if (window.confirmation) {
    window.confirmationModal = new Modal(document.getElementById("modal-confirmation"), true);
}
declare var window: formWindow;

if (window.userExpiryEnabled) {
    const messageEl = document.getElementById("user-expiry-message") as HTMLElement;
    const calculateTime = () => {
        let time = new Date()
        time.setMonth(time.getMonth() + window.userExpiryMonths);
        time.setDate(time.getDate() + window.userExpiryDays);
        time.setHours(time.getHours() + window.userExpiryHours);
        time.setMinutes(time.getMinutes() + window.userExpiryMinutes);
        messageEl.textContent = window.userExpiryMessage.replace("{date}", toDateString(time));
        setTimeout(calculateTime, 1000);
    };
    calculateTime();
}

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

const form = document.getElementById("form-create") as HTMLFormElement;
const submitButton = form.querySelector("input[type=submit]") as HTMLInputElement;
const submitSpan = form.querySelector("span.submit") as HTMLSpanElement;
let usernameField = document.getElementById("create-username") as HTMLInputElement;
const emailField = document.getElementById("create-email") as HTMLInputElement;
if (!window.usernameEnabled) { usernameField.parentElement.remove(); usernameField = emailField; }
const passwordField = document.getElementById("create-password") as HTMLInputElement;
const rePasswordField = document.getElementById("create-reenter-password") as HTMLInputElement;

const checkPasswords = () => {
    if (passwordField.value != rePasswordField.value) {
        rePasswordField.setCustomValidity(window.invalidPassword);
        submitButton.disabled = true;
        submitSpan.setAttribute("disabled", "");
    } else {
        rePasswordField.setCustomValidity("");
        submitButton.disabled = false;
        submitSpan.removeAttribute("disabled");
    }
};
rePasswordField.addEventListener("keyup", checkPasswords);
passwordField.addEventListener("keyup", checkPasswords);

interface respDTO {
    response: boolean;
    error: string;
}

interface sendDTO {
    code: string;
    email: string;
    username: string;
    password: string;
    telegram_pin?: string;
    telegram_contact?: boolean;
    discord_pin?: string;
    discord_contact?: boolean;
}

const create = (event: SubmitEvent) => {
    event.preventDefault();
    toggleLoader(submitSpan);
    let send: sendDTO = {
        code: window.code,
        username: usernameField.value,
        email: emailField.value,
        password: passwordField.value
    };
    if (telegramVerified) {
        send.telegram_pin = window.telegramPIN;
        const radio = document.getElementById("contact-via-telegram") as HTMLInputElement;
        if (radio.checked) {
            send.telegram_contact = true;
        }
    }
    if (discordVerified) {
        send.discord_pin = window.discordPIN;
        const radio = document.getElementById("contact-via-discord") as HTMLInputElement;
        if (radio.checked) {
            send.discord_contact = true;
        }
    }
    _post("/newUser", send, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            let vals = req.response as respDTO;
            let valid = true;
            for (let type in vals) {
                if (requirements[type]) { requirements[type].valid = vals[type]; }
                if (!vals[type]) { valid = false; }
            }
            if (req.status == 200 && valid) {
                window.successModal.show();
            } else {
                submitSpan.classList.add("~critical");
                submitSpan.classList.remove("~urge");
                setTimeout(() => {
                    submitSpan.classList.add("~urge");
                    submitSpan.classList.remove("~critical");
                }, 1000);
            }
        }
    }, true, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            toggleLoader(submitSpan);
            if (req.status == 401) {
                if (req.response["error"] as string) {
                    if (req.response["error"] == "confirmEmail") {
                        window.confirmationModal.show();
                        return;
                    }
                    const old = submitSpan.textContent;
                    if (req.response["error"] in window.messages) {
                        submitSpan.textContent = window.messages[req.response["error"]];
                    } else {
                        submitSpan.textContent = req.response["error"];
                    }
                    setTimeout(() => { submitSpan.textContent = old; }, 1000);
                }
            }
        }
    });
};

form.onsubmit = create;

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

var requirements: { [category: string]: Requirement} = {};

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
