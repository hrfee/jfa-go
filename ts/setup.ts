import { _get, _post, toggleLoader } from "./modules/common.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";

interface sWindow extends Window {
    messages: {};
}

declare var window: sWindow;
window.URLBase = "";

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
    // Nothing depends on input, but we add an empty broadcast function so we can just loop over all settings to fix dependents on start.
    broadcast = () => {}
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
    
    private _section: string;
    private _setting: string;
    broadcast = () => {
        if (this._section && this._setting) {
            const ev = new CustomEvent(`settings-${this._section}-${this._setting}`, { "detail": this._el.checked })
            document.dispatchEvent(ev);
        }
    }
    constructor(el: HTMLElement, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._el = el as HTMLInputElement;
        if (section && setting) {
            this._section = section;
            this._setting = setting;
            this._el.onchange = this.broadcast; 
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
    
    private _section: string;
    private _setting: string;
    broadcast = () => {
        if (this._section && this._setting) {
            const ev = new CustomEvent(`settings-${this._section}-${this._setting}`, { "detail": this._els[0].checked })
            document.dispatchEvent(ev);
        }
    }
    constructor(name: string, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._els = document.getElementsByName(name) as NodeListOf<HTMLInputElement>;
        if (section && setting) {
            this._section = section;
            this._setting = setting;
            this._els[0].onchange = this.broadcast;
            this._els[1].onchange = this.broadcast;
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

// class Radios {
//     private _el: HTMLInputElement;
//     get value(): string { return this._el.value; }
//     set value(v: string) { this._el.value = v; }
//     constructor(name: string, depends?: string, dependsTrue?: boolean, section?: string) {
//         this._el = document.getElementsByName(name)[0] as HTMLInputElement;
//         if (depends) {
//             document.addEventListener(`settings-${section}-${depends}`, (event: boolEvent) => {
//                 let el = this._el as HTMLElement;
//                 if (el.parentElement.tagName == "LABEL") { el = el.parentElement; }
//                 if (event.detail !== dependsTrue) {
//                     el.classList.add("unfocused");
//                 } else {
//                     el.classList.remove("unfocused");
//                 }
//             });
//         }
//     }
// }

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
    
    private _section: string;
    private _setting: string;
    broadcast = () => {
        if (this._section && this._setting) {
            const ev = new CustomEvent(`settings-${this._section}-${this._setting}`, { "detail": this.value ? true : false })
            document.dispatchEvent(ev);
        }
    }
    constructor(el: HTMLElement, depends?: string, dependsTrue?: boolean, section?: string, setting?: string) {
        this._el = el as HTMLSelectElement;
        if (section && setting) {
            this._section = section;
            this._setting = setting;
            this._el.addEventListener("change", this.broadcast);
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
        "host": new Input(get("ui-host")),
        "port": new Input(get("ui-port")),
        "url_base": new Input(get("ui-url_base")),
        "theme": new Select(get("ui-theme")),
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
    "ombi": {
        "enabled": new Checkbox(get("ombi-enabled"), "", false, "ombi", "enabled"),
        "server": new Input(get("ombi-server"), "", "", "enabled", true, "ombi"),
        "api_key": new Input(get("ombi-api_key"), "", "", "enabled", true, "ombi")
    },
    "advanced": {
        "tls": new Checkbox(get("advanced-tls"), "", false, "advanced", "tls"),
        "tls_port": new Input(get("advanced-tls_port"), "", "", "tls", true, "advanced"),
        "tls_cert": new Input(get("advanced-tls_cert"), "", "", "tls", true, "advanced"),
        "tls_key": new Input(get("advanced-tls_key"), "", "", "tls", true, "advanced")
    }
};

(() => {
    const checkTheme = () => {
        if (settings["ui"]["theme"].value.includes("Dark")) {
            document.documentElement.classList.add("dark-theme");
            document.documentElement.classList.remove("light-theme");
        } else {
            document.documentElement.classList.add("light-theme");
            document.documentElement.classList.remove("dark-theme");
        }
    };
    settings["ui"]["theme"].onchange = checkTheme;
    checkTheme();
})();


const restartButton = document.getElementById("restart") as HTMLSpanElement;
const serialize = () => {
    toggleLoader(restartButton);
    let config = {};
    for (let section in settings) {
        config[section] = {};
        for (let setting in settings[section]) {
            if (settings[section][setting].value) {
                config[section][setting] = settings[section][setting].value;
            }
        }
    }
    config["restart-program"] = true;
    _post("/config", config, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            toggleLoader(restartButton);
            restartButton.parentElement.querySelector("span.back").classList.add("unfocused");
            restartButton.classList.add("unfocused");
            const refresh = document.getElementById("refresh") as HTMLSpanElement;
            refresh.classList.remove("unfocused");
            refresh.onclick = () => {
                let host = window.location.href.split("#")[0].split("?")[0] + settings["ui"]["url_base"].value;
                window.location.href = host;
            };
        }
    }, true, () => {});
}
restartButton.onclick = serialize;

const relatedToEmail = Array.from(document.getElementsByClassName("related-to-email"));
const emailMethodChange = () => {
    const val = settings["email"]["method"].value;
    const smtp = document.getElementById("email-smtp");
    const mailgun = document.getElementById("email-mailgun");
    if (val == "smtp") {
        smtp.classList.remove("unfocused");
        mailgun.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.remove("hidden");
        }
    } else if (val == "mailgun") {
        mailgun.classList.remove("unfocused");
        smtp.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.remove("hidden");
        }
    } else {
        mailgun.classList.add("unfocused");
        smtp.classList.add("unfocused");
        for (let el of relatedToEmail) {
            el.classList.add("hidden");
        }
    }
};
settings["email"]["method"].onchange = emailMethodChange;
emailMethodChange();

(window as any).settings = settings;

for (let section in settings) {
    for (let setting in settings[section]) {
        settings[section][setting].broadcast();
    }
}

const pageNames: string[][] = [];

window.history.replaceState("welcome", "Setup - jfa-go");

const changePage = (title: string, pageTitle: string) => {
    const urlParams = new URLSearchParams(window.location.search);
    const lang = urlParams.get("lang");
    let page = "/#" + title;
    if (lang) { page += "?lang=" + lang; }
    window.history.pushState(title || "welcome", pageTitle, page);
};
const cards = Array.from(document.getElementById("page-container").getElementsByClassName("card")) as Array<HTMLDivElement>;
(window as any).cards = cards;
window.onpopstate = (event: PopStateEvent) => {
    if (event.state === "welcome") {
        cards[0].classList.remove("unfocused");
        for (let i = 1; i < cards.length; i++) { cards[i].classList.add("unfocused"); }
        return;
    }
    for (let i = 0; i < cards.length; i++) {
        if (event.state === pageNames[i][0]) {
            cards[i].classList.remove("unfocused");
        } else {
            cards[i].classList.add("unfocused");
        }
    }
};

(() => {
    for (let i = 0; i < cards.length; i++) {
        const card = cards[i];
        const back = card.getElementsByClassName("back")[0] as HTMLSpanElement;
        const next = card.getElementsByClassName("next")[0] as HTMLSpanElement;
        console.log(cards[i]);
        const titleEl = cards[i].querySelector("span.heading") as HTMLElement;
        let title = titleEl.textContent.replace("/", "_").replace(" ", "-");
        if (titleEl.classList.contains("welcome")) {
            title = "";
        }
        let pageTitle = titleEl.textContent + " - jfa-go";
        pageNames.push([title, pageTitle]);
        if (back) { back.addEventListener("click", () => {
            let found = false;
            for (let ind = cards.length - 1; ind >= 0; ind--) {
                cards[ind].classList.add("unfocused");
                if (ind < i && !(cards[ind].classList.contains("hidden")) && !found) {
                    cards[ind].classList.remove("unfocused");
                    changePage(pageNames[ind][0], pageNames[ind][1]);
                    found = true;
                }
            }
            window.scrollTo(0, 0);
        }); }
        if (next) { next.addEventListener("click", () => {
            let found = false;
            for (let ind = 0; ind < cards.length; ind++) {
                cards[ind].classList.add("unfocused");
                if (ind > i && !(cards[ind].classList.contains("hidden")) && !found) {
                    cards[ind].classList.remove("unfocused");
                    changePage(pageNames[ind][0], pageNames[ind][1]);
                    found = true;
                }
            }
            window.scrollTo(0, 0);
        }); }
    }
})();

(() => {
    const button = document.getElementById("jellyfin-test-connection") as HTMLSpanElement;
    const ogText = button.textContent;
    const nextButton = button.parentElement.querySelector("span.next") as HTMLSpanElement;
    button.onclick = () => {
        toggleLoader(button);
        let send = {
            "type": settings["jellyfin"]["type"].value,
            "server": settings["jellyfin"]["server"].value,
            "username": settings["jellyfin"]["username"].value,
            "password": settings["jellyfin"]["password"].value
        };
        _post("/jellyfin/test", send, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                toggleLoader(button);
                const success = req.response["success"] as boolean;
                if (success) {
                    nextButton.removeAttribute("disabled");
                    button.textContent = window.lang.strings("success");
                    button.classList.add("~positive");
                    button.classList.remove("~urge");
                    setTimeout(() => {
                        button.textContent = ogText;
                        button.classList.add("~urge");
                        button.classList.remove("~positive");
                    }, 5000);
                } else {
                    nextButton.setAttribute("disabled", "");
                    button.textContent = window.lang.strings("error");
                    button.classList.add("~critical");
                    button.classList.remove("~urge");
                    setTimeout(() => {
                        button.textContent = ogText;
                        button.classList.add("~urge");
                        button.classList.remove("~critical");
                    }, 5000);
                }
            }
        }, true, () => {});
    };
})();

loadLangSelector("setup");
