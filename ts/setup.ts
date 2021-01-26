import { _get, _post } from "./modules/common.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";

interface sWindow extends Window {
    messages: {};
}

declare var window: sWindow;

const get = (id: string): HTMLElement => document.getElementById(id);
const text = (id: string, val: string) => { document.getElementById(id).textContent = val; };
const html = (id: string, val: string) => { document.getElementById(id).innerHTML = val; };

interface boolEvent extends Event {
    detail: boolean;
}

class Input {
    private _el: HTMLInputElement;
    get value(): string { return ""+this._el.value; }
    set value(v: string) { this._el.value = v; }
    constructor(el: HTMLElement, placeholder?: any, value?: any, depends?: string, dependsTrue?: boolean, section?: string) {
        this._el = el as HTMLInputElement;
        if (placeholder) { this._el.placeholder = placeholder; }
        if (value) { this.value = value; }
        if (depends) {
            document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
                let el = this._el as HTMLElement;
                if (el.parentElement.tagName == "LABEL") { el = el.parentElement; }
                if (event.detail !== dependsTrue) {
                    el.classList.add("unfocused");
                } else {
                    el.classList.remove("unfocused");
                }
            });
        }
    }
}

class Checkbox {
    private _el: HTMLInputElement;
    get value(): string { return this._el.checked ? "true" : "false"; }
    set value(v: string) { this._el.checked = (v == "true") ? true : false; }
    constructor(el: HTMLElement, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._el = el as HTMLInputElement;
        if (section && setting) {
            this._el.onchange = () => {
                const ev = new CustomEvent(`settings-${section}-${setting}`, { "detail": this._el.checked })
                document.dispatchEvent(ev);
            };
        }
        if (depends) {
            document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
                let el = this._el as HTMLElement;
                if (el.parentElement.tagName == "LABEL") { el = el.parentElement; }
                if (event.detail !== dependsTrue) {
                    el.classList.add("unfocused");
                } else {
                    el.classList.remove("unfocused");
                }
            });
        }
    }
}

class BoolRadios {
    private _els: NodeListOf<HTMLInputElement>;
    get value(): string { return this._els[0].checked ? "true" : "false" }
    set value(v: string) { 
        const bool = (v == "true") ? true : false;
        this._els[0].checked = bool;
        this._els[1].checked = !bool;
    }
    constructor(name: string, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._els = document.getElementsByName(name) as NodeListOf<HTMLInputElement>;
        if (section && setting) {
            const onchange = () => {
                const ev = new CustomEvent(`settings-${section}-${setting}`, { "detail": this._els[0].checked })
                document.dispatchEvent(ev);
            };
            this._els[0].onchange = onchange;
            this._els[1].onchange = onchange;
        }
        if (depends) {
            document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
                if (event.detail !== dependsTrue) {
                    if (this._els[0].parentElement.tagName == "LABEL") {
                        this._els[0].parentElement.classList.add("unfocused");
                    }
                    if (this._els[1].parentElement.tagName == "LABEL") {
                        this._els[1].parentElement.classList.add("unfocused");
                    }
                } else {
                    if (this._els[0].parentElement.tagName == "LABEL") {
                        this._els[0].parentElement.classList.remove("unfocused");
                    }
                    if (this._els[1].parentElement.tagName == "LABEL") {
                        this._els[1].parentElement.classList.remove("unfocused");
                    }
                }
            });
        }
    }
}

class Radios {
    private _el: HTMLInputElement;
    get value(): string { return this._el.value; }
    set value(v: string) { this._el.value = v; }
    constructor(name: string, depends?: string, dependsTrue?: boolean, section?: string) {
        this._el = document.getElementsByName(name)[0] as HTMLInputElement;
        if (depends) {
            document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
                let el = this._el as HTMLElement;
                if (el.parentElement.tagName == "LABEL") { el = el.parentElement; }
                if (event.detail !== dependsTrue) {
                    el.classList.add("unfocused");
                } else {
                    el.classList.remove("unfocused");
                }
            });
        }
    }
}

class Select {
    private _el: HTMLSelectElement;
    get value(): string { return this._el.value; }
    set value(v: string) { this._el.value = v; }
    add = (val: string, label: string) => {
        const item = document.createElement("option") as HTMLOptionElement;
        item.value = val;
        item.textContent = label;
        this._el.appendChild(item);
    }
    set onchange(f: () => void) {
        this._el.addEventListener("change", f);
    }
    constructor(el: HTMLElement, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._el = el as HTMLSelectElement;
        if (section && setting) {
            this._el.addEventListener("change", () => {
                const ev = new CustomEvent(`settings-${section}-${setting}`, { "detail": this.value ? true : false })
                document.dispatchEvent(ev);
            });
        }
        if (depends) {
            document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
                let el = this._el as HTMLElement;
                if (el.parentElement.tagName == "LABEL") { el = el.parentElement; }
                if (event.detail !== dependsTrue) {
                    el.classList.add("unfocused");
                } else {
                    el.classList.remove("unfocused");
                }
            });
        }
    }
}

class LangSelect extends Select {
    constructor(page: string, el: HTMLElement) {
        super(el);
        _get("/lang/" + page, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && req.status == 200) {
                for (let code in req.response) {
                    this.add(code, req.response[code]);
                }
                this.value = "en-us";
            }
        });
    }
}

window.lang = new lang(window.langFile as LangFile);
html("language-description", window.lang.var("language", "description", `<a href="https://weblate.hrfee.pw">Weblate</a>`));
html("email-description", window.lang.var("email", "description", `<a href="https://mailgun.com">Mailgun</a>`));

const settings = {
    "jellyfin": {
        "type": new Select(get("jellyfin-type")),
        "server": new Input(get("jellyfin-server")),
        "public_server": new Input(get("jellyfin-public_server")),
        "username": new Input(get("jellyfin-username")),
        "password": new Input(get("jellyfin-password")),
        "substitute_jellyfin_strings": new Input(get("jellyfin-substitute_jellyfin_strings"))
    },
    "ui": {
        "language-form": new LangSelect("form", get("ui-language-form")),
        "language-admin": new LangSelect("admin", get("ui-language-admin")),
        "jellyfin_login": new BoolRadios("ui-jellyfin_login", "", false, "ui", "jellyfin_login"),
        "admin_only": new Checkbox(get("ui-admin_only"), "jellyfin_login", true, "ui"),
        "username": new Input(get("ui-username"), "", "", "jellyfin_login", false, "ui"),
        "password": new Input(get("ui-password"), "", "", "jellyfin_login", false, "ui"),
        "email": new Input(get("ui-email"), "", "", "jellyfin_login", false, "ui"),
        "contact_message": new Input(get("ui-contact_message"), window.messages["ui"]["contact_message"]),
        "help_message": new Input(get("ui-help_message"), window.messages["ui"]["help_message"]),
        "success_message": new Input(get("ui-success_message"), window.messages["ui"]["success_message"])
    },
    "password_validation": {
        "enabled": new Checkbox(get("password_validation-enabled"), "", false, "password_validation", "enabled"),
        "min_length": new Input(get("password_validation-min_length"), "", 8, "enabled", true, "password_validation"),
        "upper": new Input(get("password_validation-upper"), "", 1, "enabled", true, "password_validation"),
        "lower": new Input(get("password_validation-lower"), "", 0, "enabled", true, "password_validation"),
        "number": new Input(get("password_validation-number"), "", 1, "enabled", true, "password_validation"),
        "special": new Input(get("password_validation-special"), "", 0, "enabled", true, "password_validation")
    },
    "email": {
        "language": new LangSelect("email", get("email-language")),
        "no_username": new Checkbox(get("email-no_username"), "method", true, "email"),
        "use_24h": new BoolRadios("email-24h", "method", true, "email"),
        "date_format": new Input(get("email-date_format"), "", "%d/%m/%y", "method", true, "email"),
        "message": new Input(get("email-message"), window.messages["email"]["message"], "", "method", true, "email"),
        "method": new Select(get("email-method"), "", false, "email", "method"),
        "address": new Input(get("email-address"), "jellyfin@jellyf.in", "", "method", true, "email"),
        "from": new Input(get("email-from"), "", "Jellyfin", "method", true, "email")
    },
    "password_resets": {
        "enabled": new Checkbox(get("password_resets-enabled"), "", false, "password_resets", "enabled"),
        "watch_directory": new Input(get("password_resets-watch_directory"), "", "", "enabled", true, "password_resets"),
        "subject": new Input(get("password_resets-subject"), "", "", "enabled", true, "password_resets")
    },
    "notifications": {
        "enabled": new Checkbox(get("notifications-enabled"))
    },
    "welcome_email": {
        "enabled": new Checkbox(get("welcome_email-enabled"), "", false, "welcome_email", "enabled"),
        "subject": new Input(get("welcome_email-subject"), "", "", "enabled", true, "welcome_email")
    },
    "invite_emails": {
        "enabled": new Checkbox(get("invite_emails-enabled"), "", false, "invite_emails", "enabled"),
        "subject": new Input(get("invite_emails-subject"), "", "", "enabled", true, "invite_emails"),
        "url_base": new Input(get("invite_emails-url_base"), "", "", "enabled", true, "invite_emails")
    },
    "mailgun": {
        "api_url": new Input(get("mailgun-api_url")),
        "api_key": new Input(get("mailgun-api_key"))
    },
    "smtp": {
        "username": new Input(get("smtp-username")),
        "encryption": new Select(get("smtp-encryption")),
        "server": new Input(get("smtp-server")),
        "port": new Input(get("smtp-port")),
        "password": new Input(get("smtp-password"))
    },
};

const relatedToEmail = Array.from(document.getElementsByClassName("related-to-email"));
settings["email"]["method"].onchange = () => {
    const val = settings["email"]["method"].value;
    const smtp = document.getElementById("email-smtp");
    const mailgun = document.getElementById("email-mailgun");
    if (val == "smtp") {
        smtp.classList.remove("unfocused");
        mailgun.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.remove("unfocused");
        }
    } else if (val == "mailgun") {
        mailgun.classList.remove("unfocused");
        smtp.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.remove("unfocused");
        }
    } else {
        mailgun.classList.add("unfocused");
        smtp.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.add("unfocused");
        }
    }
};

loadLangSelector("setup");
