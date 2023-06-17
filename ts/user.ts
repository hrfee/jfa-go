import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { _get, _post, notificationBox, whichAnimationEvent } from "./modules/common.js";
import { Login } from "./modules/login.js";

interface userWindow extends Window {
    jellyfinID: string;
    username: string;
}

declare var window: userWindow;

const theme = new ThemeManager(document.getElementById("button-theme"));

window.lang = new lang(window.langFile as LangFile);

loadLangSelector("user");

window.animationEvent = whichAnimationEvent();

window.token = "";

window.modals = {} as Modals;

(() => {
    window.modals.login = new Modal(document.getElementById("modal-login"), true);
})();

window.notifications = new notificationBox(document.getElementById('notification-box') as HTMLDivElement, 5);

var rootCard = document.getElementById("card-user");
var contactCard = document.getElementById("card-contact");

interface MyDetailsContactMethod {
    value: string;
    enabled: boolean;
}

interface MyDetails {
    id: string;
    username: string;
    expiry: number;
    admin: boolean;
    disabled: boolean;
    email?: MyDetailsContactMethod;
    discord?: MyDetailsContactMethod;
    telegram?: MyDetailsContactMethod;
    matrix?: MyDetailsContactMethod;
}

interface ContactDTO {
    email?: boolean;
    discord?: boolean;
    telegram?: boolean;
    matrix?: boolean;
}

class ContactMethods {
    private _card: HTMLElement;
    private _buttons: { [name: string]: { element: HTMLElement, details: MyDetailsContactMethod } };

    constructor (card: HTMLElement) {
        this._card = card;
        this._buttons = {};
    }

    clear = () => {
        this._card.textContent = "";
        this._buttons = {};
    }

    append = (name: string, details: MyDetailsContactMethod, icon: string) => {
        const row = document.createElement("div");
        row.classList.add("row", "flex-expand", "my-2");
        row.innerHTML = `
            <div class="inline align-middle">
                <span class="shield ~info" alt="${name}">
                    <span class="icon">
                        ${icon}
                    </span>
                </span>
                <span class="ml-2 font-bold">${details.value}</span>
            </div>
            <div class="flex items-center">
                <button class="user-contact-enabled-disabled button ~neutral">
                    <input type="checkbox" class="mr-2">
                    <span>${window.lang.strings("enabled")}</span>
                </button>
            </div>
        `;
        
        this._buttons[name] = {
            element: row,
            details: details
        };
        
        const button = row.querySelector(".user-contact-enabled-disabled") as HTMLButtonElement;
        const checkbox = button.querySelector("input[type=checkbox]") as HTMLInputElement;
        const setButtonAppearance = () => {
            if (checkbox.checked) {
                button.classList.add("~info");
                button.classList.remove("~neutral");
            } else {
                button.classList.add("~neutral");
                button.classList.remove("~info");
            }
        };
        const onPress = () => {
            this._buttons[name].details.enabled = checkbox.checked;
            setButtonAppearance();
            this._save();
        };

        checkbox.onchange = onPress;
        button.onclick = () => {
            checkbox.checked = !checkbox.checked;
            onPress();
        };

        checkbox.checked = details.enabled;
        setButtonAppearance();

        this._card.appendChild(row);
    };

    private _save = () => {
        let data: ContactDTO = {};
        for (let method of Object.keys(this._buttons)) {
            data[method] = this._buttons[method].details.enabled;
        }

        _post("/my/contact", data, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    window.notifications.customError("errorSetNotify", window.lang.notif("errorSaveSettings"));
                    document.dispatchEvent(new CustomEvent("details-reload"));
                }
            }
        });
    };
}

var contactMethodList = new ContactMethods(contactCard);

document.addEventListener("details-reload", () => {
    _get("/my/details", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                window.notifications.customError("myDetailsError", req.response["error"]);
                return;
            }
            const details: MyDetails = req.response as MyDetails;
            window.jellyfinID = details.id;
            window.username = details.username;
            let innerHTML = `
            <span>${window.lang.strings("welcomeUser").replace("{user}", window.username)}</span>
            `;
            if (details.admin) {
                innerHTML += `<span class="chip ~info ml-4">${window.lang.strings("admin")}</span>`;
            }
            if (details.disabled) {
                innerHTML += `<span class="chip ~warning ml-4">${window.lang.strings("disabled")}</span>`;
            }

            rootCard.querySelector(".heading").innerHTML = innerHTML;

            contactMethodList.clear(); 

            const contactMethods = [
                ["email", `<i class="ri-mail-fill"></i>`],
                ["discord", `<i class="ri-discord-fill"></i>`],
                ["telegram", `<i class="ri-telegram-fill"></i>`],
                ["matrix", `[m]`]
            ];
            
            for (let method of contactMethods) {
                if (method[0] in details) {
                    contactMethodList.append(method[0], details[method[0]], method[1]);
                }
            }
        }
    });
});

const login = new Login(window.modals.login as Modal, "/my/");
login.onLogin = () => {
    console.log("Logged in.");
    document.dispatchEvent(new CustomEvent("details-reload"));
};



login.bindLogout(document.getElementById("logout-button"));

login.login("", "");
