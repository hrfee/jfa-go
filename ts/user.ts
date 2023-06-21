import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { _get, _post, _delete, notificationBox, whichAnimationEvent, toDateString, toggleLoader } from "./modules/common.js";
import { Login } from "./modules/login.js";
import { Discord, Telegram, Matrix, ServiceConfiguration, MatrixConfiguration } from "./modules/account-linking.js";

interface userWindow extends Window {
    jellyfinID: string;
    username: string;
    emailRequired: boolean;
    discordRequired: boolean;
    telegramRequired: boolean;
    matrixRequired: boolean;
    discordServerName: string;
    discordInviteLink: boolean;
    matrixUserID: string;
    discordSendPINMessage: string;
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
    window.modals.email = new Modal(document.getElementById("modal-email"), false);
    if (window.discordEnabled) {
        window.modals.discord = new Modal(document.getElementById("modal-discord"), false);
    }
    if (window.telegramEnabled) {
        window.modals.telegram = new Modal(document.getElementById("modal-telegram"), false);
    }
    if (window.matrixEnabled) {
        window.modals.matrix = new Modal(document.getElementById("modal-matrix"), false);
    }
})();

window.notifications = new notificationBox(document.getElementById('notification-box') as HTMLDivElement, 5);

const grid = document.querySelector(".grid");
var rootCard = document.getElementById("card-user");
var contactCard = document.getElementById("card-contact");
var statusCard = document.getElementById("card-status");

interface MyDetailsContactMethod {
    value: string;
    enabled: boolean;
}

interface MyDetails {
    id: string;
    username: string;
    expiry: number;
    admin: boolean;
    accounts_admin: boolean;
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
    private _content: HTMLElement;
    private _buttons: { [name: string]: { element: HTMLElement, details: MyDetailsContactMethod } };

    constructor (card: HTMLElement) {
        this._card = card;
        this._content = this._card.querySelector(".content");
        this._buttons = {};
    }

    clear = () => {
        this._content.textContent = "";
        this._buttons = {};
    }

    append = (name: string, details: MyDetailsContactMethod, icon: string, addEditFunc?: (add: boolean) => void, required?: boolean) => {
        const row = document.createElement("div");
        row.classList.add("flex", "flex-expand", "my-2", "flex-nowrap");
        let innerHTML = `
            <div class="flex items-baseline flex-nowrap ellipsis">
                <span class="shield ~urge" alt="${name}">
                    <span class="icon">
                        ${icon}
                    </span>
                </span>
                <span class="ml-2 font-bold text-ellipsis overflow-hidden">${(details.value == "") ? window.lang.strings("notSet") : details.value}</span>
            </div>
            <div class="flex items-center ml-2">
                <button class="user-contact-enabled-disabled button ~neutral" ${details.value == "" ? "disabled" : ""}>
                    <input type="checkbox" class="mr-2" ${details.value == "" ? "disabled" : ""}>
                    <span>${window.lang.strings("enabled")}</span>
                </button>
        `;
        if (addEditFunc) {
            innerHTML += `
                <button class="user-contact-edit button ~info ml-2">
                    <i class="ri-${details.value == "" ? "add" : "edit"}-fill mr-2"></i>
                    <span>${details.value == "" ? window.lang.strings("add") : window.lang.strings("edit")}</span>
                </button>
            `;
        }

        if (!required && details.value != "") {
            innerHTML += `
                <button class="user-contact-delete button ~critical ml-2" alt="${window.lang.strings("delete")}" text="${window.lang.strings("delete")}">
                    &times;
                </button>
            `;
        }

        innerHTML += `
            </div>
        `;

        row.innerHTML = innerHTML;
        
        this._buttons[name] = {
            element: row,
            details: details
        };
        
        const button = row.querySelector(".user-contact-enabled-disabled") as HTMLButtonElement;
        const checkbox = button.querySelector("input[type=checkbox]") as HTMLInputElement;
        const setButtonAppearance = () => {
            if (checkbox.checked) {
                button.classList.add("~urge");
                button.classList.remove("~neutral");
            } else {
                button.classList.add("~neutral");
                button.classList.remove("~urge");
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

        if (addEditFunc) {
            const addEditButton = row.querySelector(".user-contact-edit") as HTMLButtonElement;
            addEditButton.onclick = () => addEditFunc(details.value == "");
        }
        
        if (!required && details.value != "") {
            const deleteButton = row.querySelector(".user-contact-delete") as HTMLButtonElement;
            deleteButton.onclick = () => _delete("/my/" + name, null, (req: XMLHttpRequest) => {
                if (req.readyState != 4) return;
                document.dispatchEvent(new CustomEvent("details-reload"));
            });
        }

        this._content.appendChild(row);
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

class ExpiryCard {
    private _card: HTMLElement;
    private _expiry: Date;
    private _aside: HTMLElement;
    private _countdown: HTMLElement;
    private _interval: number = null;

    constructor(card: HTMLElement) {
        this._card = card;
        this._aside = this._card.querySelector(".user-expiry") as HTMLElement;
        this._countdown = this._card.querySelector(".user-expiry-countdown") as HTMLElement;
    }

    private _drawCountdown = () => {
        let now = new Date();
        // Years, Months, Days
        let ymd = [0, 0, 0];
        while (now.getFullYear() != this._expiry.getFullYear()) {
            ymd[0] += 1;
            now.setFullYear(now.getFullYear()+1);
        }
        if (now.getMonth() > this._expiry.getMonth()) {
            ymd[0] -=1;
            now.setFullYear(now.getFullYear()-1);
        }
        while (now.getMonth() != this._expiry.getMonth()) {
            ymd[1] += 1;
            now.setMonth(now.getMonth() + 1);
        }
        if (now.getDate() > this._expiry.getDate()) {
            ymd[1] -=1;
            now.setMonth(now.getMonth()-1);
        }
        while (now.getDate() != this._expiry.getDate()) {
            ymd[2] += 1;
            now.setDate(now.getDate() + 1);
        }
        
        const langKeys = ["year", "month", "day"];
        let innerHTML = ``;
        for (let i = 0; i < langKeys.length; i++) {
            if (ymd[i] == 0) continue;
            const words = window.lang.quantity(langKeys[i], ymd[i]).split(" ");
            innerHTML += `
            <div class="row my-3">
                <div class="inline baseline">
                    <span class="text-2xl">${words[0]}</span> <span class="text-gray-400 text-lg">${words[1]}</span>
                </div>
            </div>
            `;
        }
        this._countdown.innerHTML = innerHTML;
    };

    get expiry(): Date { return this._expiry; };
    set expiry(expiryUnix: number) {
        if (this._interval !== null) {
            window.clearInterval(this._interval);
            this._interval = null;
        }
        if (expiryUnix == 0) return;
        this._expiry = new Date(expiryUnix * 1000);
        this._aside.textContent = window.lang.strings("yourAccountIsValidUntil").replace("{date}", toDateString(this._expiry));
        this._card.classList.remove("unfocused");

        this._interval = window.setInterval(this._drawCountdown, 60*1000);
        this._drawCountdown();
    }
}

var expiryCard = new ExpiryCard(statusCard);

var contactMethodList = new ContactMethods(contactCard);

const addEditEmail = (add: boolean): void => {
    const heading = window.modals.email.modal.querySelector(".heading");
    heading.innerHTML = (add ? window.lang.strings("addContactMethod") : window.lang.strings("editContactMethod")) + `<span class="modal-close">&times;</span>`;
    const input = document.getElementById("modal-email-input") as HTMLInputElement;
    input.value = "";
    const confirmationRequired = window.modals.email.modal.querySelector(".confirmation-required");
    confirmationRequired.classList.add("unfocused");
   
    const content = window.modals.email.modal.querySelector(".content");
    content.classList.remove("unfocused");

    const submit = window.modals.email.modal.querySelector(".modal-submit") as HTMLButtonElement;
    submit.onclick = () => {
        toggleLoader(submit);
        _post("/my/email", {"email": input.value}, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && (req.status == 303 || req.status == 200)) {
                document.dispatchEvent(new CustomEvent("details-reload"));
            }
        }, true, (req: XMLHttpRequest) => {
            if (req.readyState == 4 && req.status == 401) {
                content.classList.add("unfocused");
                confirmationRequired.classList.remove("unfocused");
            }
        });
    }

    window.modals.email.show();
}

const discordConf: ServiceConfiguration = {
    modal: window.modals.discord as Modal,
    pin: "",
    inviteURL: window.discordInviteLink ? "/my/discord/invite" : "",
    pinURL: "/my/pin/discord",
    verifiedURL: "/my/discord/verified/",
    invalidCodeError: window.lang.notif("errorInvalidPIN"),
    accountLinkedError: window.lang.notif("errorAccountLinked"),
    successError: window.lang.notif("verified"),
    successFunc: (modalClosed: boolean) => {
        if (modalClosed) document.dispatchEvent(new CustomEvent("details-reload"));
    }
};

let discord = new Discord(discordConf);

const telegramConf: ServiceConfiguration = {
    modal: window.modals.telegram as Modal,
    pin: "",
    pinURL: "/my/pin/telegram",
    verifiedURL: "/my/telegram/verified/",
    invalidCodeError: window.lang.notif("errorInvalidPIN"),
    accountLinkedError: window.lang.notif("errorAccountLinked"),
    successError: window.lang.notif("verified"),
    successFunc: (modalClosed: boolean) => {
        if (modalClosed) document.dispatchEvent(new CustomEvent("details-reload"));
    }
};

let telegram = new Telegram(telegramConf);

const matrixConf: MatrixConfiguration = {
    modal: window.modals.matrix as Modal,
    sendMessageURL: "/my/matrix/user",
    verifiedURL: "/my/matrix/verified/",
    invalidCodeError: window.lang.notif("errorInvalidPIN"),
    accountLinkedError: window.lang.notif("errorAccountLinked"),
    unknownError: window.lang.notif("errorUnknown"),
    successError: window.lang.notif("verified"),
    successFunc: () => {
        setTimeout(() => document.dispatchEvent(new CustomEvent("details-reload")), 1200);
    }
};

let matrix = new Matrix(matrixConf);
    

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

            // Note the weird format of the functions for discord/telegram:
            // "this" was being redefined within the onclick() method, so
            // they had to be wrapped in an anonymous function.
            const contactMethods: { name: string, icon: string, f: (add: boolean) => void, required: boolean }[] = [
                {name: "email", icon: `<i class="ri-mail-fill ri-lg"></i>`, f: addEditEmail, required: true},
                {name: "discord", icon: `<i class="ri-discord-fill ri-lg"></i>`, f: (add: boolean) => { discord.onclick(); }, required: window.discordRequired},
                {name: "telegram", icon: `<i class="ri-telegram-fill ri-lg"></i>`, f: (add: boolean) => { telegram.onclick() }, required: window.telegramRequired},
                {name: "matrix", icon: `<span class="font-bold">[m]</span>`, f: (add: boolean) => { matrix.show(); }, required: window.matrixRequired}
            ];
            
            for (let method of contactMethods) {
                if (method.name in details) {
                    contactMethodList.append(method.name, details[method.name], method.icon, method.f, method.required);
                }
            }

            expiryCard.expiry = details.expiry;


            let messageCard = document.getElementById("card-message");
            if (details.accounts_admin) {
                if (typeof(messageCard) == "undefined" || messageCard == null) {
                    messageCard = document.createElement("div");
                    messageCard.classList.add("card", "@low", "dark:~d_neutral", "content");
                    messageCard.id = "card-message";
                    contactCard.parentElement.insertBefore(messageCard, contactCard);
                }
                if (!messageCard.textContent) {
                    messageCard.innerHTML = `
                    <span class="heading mb-2">${window.lang.strings("customMessagePlaceholderHeader")} ✏️ </span>
                    <span class="block">${window.lang.strings("customMessagePlaceholderContent")}</span>
                    `;
                }
            }

            if (typeof(messageCard) != "undefined" && messageCard != null) {
                let largestNonMessageCardHeight = 0;
                const cards = grid.querySelectorAll(".card") as NodeListOf<HTMLElement>;
                for (let i = 0; i < cards.length; i++) {
                    if (cards[i].id == "card-message") continue;
                    if (computeRealHeight(cards[i]) > largestNonMessageCardHeight) {
                        largestNonMessageCardHeight = computeRealHeight(cards[i]);
                    }
                }
                

                let rowSpan = Math.ceil(computeRealHeight(messageCard) / largestNonMessageCardHeight);

                if (rowSpan > 0)
                    messageCard.style.gridRow = `span ${rowSpan}`;
            }
        }
    });
});

const login = new Login(window.modals.login as Modal, "/my/");
login.onLogin = () => {
    console.log("Logged in.");
    document.querySelector(".page-container").classList.remove("unfocused");
    document.dispatchEvent(new CustomEvent("details-reload"));
};

const computeRealHeight = (el: HTMLElement): number => {
    let children = el.children as HTMLCollectionOf<HTMLElement>;
    let total = 0;
    for (let i = 0; i < children.length; i++) {
        total += children[i].offsetHeight;
    }
    return total;
}

login.bindLogout(document.getElementById("logout-button"));

login.login("", "");
