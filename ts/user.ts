import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { _get, _post, _delete, notificationBox, whichAnimationEvent, toDateString, toggleLoader, addLoader, removeLoader, toClipboard } from "./modules/common.js";
import { Login } from "./modules/login.js";
import { Discord, Telegram, Matrix, ServiceConfiguration, MatrixConfiguration } from "./modules/account-linking.js";
import { Validator, ValidatorConf, ValidatorRespDTO } from "./modules/validator.js";

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
    pwrEnabled: string;
    referralsEnabled: boolean;
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
    if (window.pwrEnabled) {
        window.modals.pwr = new Modal(document.getElementById("modal-pwr"), false);
        window.modals.pwr.onclose = () => {
            window.modals.login.show();
        };
        const resetButton = document.getElementById("modal-login-pwr");
        resetButton.onclick = () => {
            const usernameInput = document.getElementById("login-user") as HTMLInputElement;
            const input = document.getElementById("pwr-address") as HTMLInputElement;
            input.value = usernameInput.value;
            window.modals.login.close();
            window.modals.pwr.show();
        }
    }
})();

window.notifications = new notificationBox(document.getElementById('notification-box') as HTMLDivElement, 5);

if (window.pwrEnabled && window.linkResetEnabled) {
    const submitButton = document.getElementById("pwr-submit");
    const input = document.getElementById("pwr-address") as HTMLInputElement;
    submitButton.onclick = () => {
        toggleLoader(submitButton);
        _post("/my/password/reset/" + input.value, null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            toggleLoader(submitButton);
            if (req.status != 204) {
                window.notifications.customError("unkownError", window.lang.notif("errorUnknown"));;
                window.modals.pwr.close();
                return;
            }
            window.modals.pwr.modal.querySelector(".heading").textContent = window.lang.strings("resetSent");
            window.modals.pwr.modal.querySelector(".content").textContent = window.lang.strings("resetSentDescription");
            submitButton.classList.add("unfocused");
            input.classList.add("unfocused");
        });
    };
}

const grid = document.querySelector(".grid");
var rootCard = document.getElementById("card-user");
var contactCard = document.getElementById("card-contact");
var statusCard = document.getElementById("card-status");
var passwordCard = document.getElementById("card-password");

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
    has_referrals: boolean;
}

interface MyReferral {
    code: string;
    remaining_uses: number;
    no_limit: boolean;
    expiry: number;
    use_expiry: boolean;
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
        row.classList.add("flex", "flex-row", "justify-between", "my-2", "flex-nowrap");
        let innerHTML = `
            <div class="flex items-baseline flex-nowrap truncate">
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

class ReferralCard {
    private _card: HTMLElement;
    private _code: string;
    private _url: string;
    private _expiry: Date;
    private _expiryUnix: number;
    private _useExpiry: boolean;
    private _remainingUses: number;
    private _noLimit: boolean;

    private _button: HTMLButtonElement;
    private _infoArea: HTMLDivElement;
    private _remainingUsesEl: HTMLSpanElement;
    private _expiryEl: HTMLSpanElement;
    private _descriptionEl: HTMLSpanElement;

    get code(): string { return this._code; }
    set code(c: string) {
        this._code = c;
        let url = window.location.href;
        for (let split of ["#", "?", "account", "my"]) {
            url = url.split(split)[0];
        }
        if (url.slice(-1) != "/") { url += "/"; }
        url = url + "invite/" + this._code;
        this._url = url;
    }

    get remaining_uses(): number { return this._remainingUses; }
    set remaining_uses(v: number) { 
        this._remainingUses = v;
        if (v > 0 && !(this._noLimit))
            this._remainingUsesEl.textContent = `${v}`;
    }

    get no_limit(): boolean { return this._noLimit; }
    set no_limit(v: boolean) {
        this._noLimit = v;
        if (v)
            this._remainingUsesEl.textContent = `∞`;
        else
            this._remainingUsesEl.textContent = `${this._remainingUses}`;
    }

    get expiry(): Date { return this._expiry; };
    set expiry(expiryUnix: number) {
        this._expiryUnix = expiryUnix;
        this._expiry = new Date(expiryUnix * 1000);
        this._expiryEl.textContent = toDateString(this._expiry);
    }

    get use_expiry(): boolean { return this._useExpiry; }
    set use_expiry(v: boolean) {
        this._useExpiry = v;
        if (v) {
            this._descriptionEl.textContent = window.lang.strings("referralsWithExpiryDescription");
        } else {
            this._descriptionEl.textContent = window.lang.strings("referralsDescription");
        }
    }
    
    constructor(card: HTMLElement) {
        this._card = card;
        this._button = this._card.querySelector(".user-referrals-button") as HTMLButtonElement;
        this._infoArea = this._card.querySelector(".user-referrals-info") as HTMLDivElement;
        this._descriptionEl = this._card.querySelector(".user-referrals-description") as HTMLSpanElement;

        this._infoArea.innerHTML = `
        <div class="row my-3">
            <div class="inline baseline">
                <span class="text-2xl referral-remaining-uses"></span> <span class="text-gray-400 text-lg">${window.lang.strings("inviteRemainingUses")}</span>
            </div>
        </div>
        <div class="row my-3">
            <div class="inline baseline">
                <span class="text-gray-400 text-lg">${window.lang.strings("expiry")}</span> <span class="text-2xl referral-expiry"></span>
            <div>
        </div>
        `;
    
        this._remainingUsesEl = this._infoArea.querySelector(".referral-remaining-uses") as HTMLSpanElement;
        this._expiryEl = this._infoArea.querySelector(".referral-expiry") as HTMLSpanElement;
        
        document.addEventListener("timefmt-change", () => {
            this.expiry = this._expiryUnix;
        });

        this._button.addEventListener("click", () => {
            toClipboard(this._url);
            const content = this._button.innerHTML;
            this._button.innerHTML = `
            ${window.lang.strings("copied")} <i class="ri-check-line ml-2"></i>
            `;
            this._button.classList.add("~positive");
            this._button.classList.remove("~info");
            setTimeout(() => {
                this._button.classList.add("~info");
                this._button.classList.remove("~positive");
                this._button.innerHTML = content;
            }, 2000);
        });
    }

    hide = () => this._card.classList.add("unfocused");

    update = (referral: MyReferral) => {
        this.code = referral.code;
        this.remaining_uses = referral.remaining_uses;
        this.no_limit = referral.no_limit;
        this.expiry = referral.expiry;
        this._card.classList.remove("unfocused");
        this.use_expiry = referral.use_expiry;
    };
}

class ExpiryCard {
    private _card: HTMLElement;
    private _expiry: Date;
    private _aside: HTMLElement;
    private _countdown: HTMLElement;
    private _interval: number = null;
    private _expiryUnix: number = 0;

    constructor(card: HTMLElement) {
        this._card = card;
        this._aside = this._card.querySelector(".user-expiry") as HTMLElement;
        this._countdown = this._card.querySelector(".user-expiry-countdown") as HTMLElement;

        document.addEventListener("timefmt-change", () => {
            this.expiry = this._expiryUnix;
        });
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
        this._expiryUnix = expiryUnix;
        if (expiryUnix == 0) {
            this._card.classList.add("unfocused");
            return;
        }
        this._expiry = new Date(expiryUnix * 1000);
        this._aside.textContent = window.lang.strings("yourAccountIsValidUntil").replace("{date}", toDateString(this._expiry));
        this._card.classList.remove("unfocused");

        this._interval = window.setInterval(this._drawCountdown, 60*1000);
        this._drawCountdown();
    }
}

var expiryCard = new ExpiryCard(statusCard);

var referralCard: ReferralCard;
if (window.referralsEnabled) referralCard = new ReferralCard(document.getElementById("card-referrals"));

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
                window.modals.email.close();
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

let discord: Discord;
if (window.discordEnabled) discord = new Discord(discordConf);

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

let telegram: Telegram;
if (window.telegramEnabled) telegram = new Telegram(telegramConf);

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

let matrix: Matrix;
if (window.matrixEnabled) matrix = new Matrix(matrixConf);


const oldPasswordField = document.getElementById("user-old-password") as HTMLInputElement;
const newPasswordField = document.getElementById("user-new-password") as HTMLInputElement;
const rePasswordField = document.getElementById("user-reenter-new-password") as HTMLInputElement;
const changePasswordButton = document.getElementById("user-password-submit") as HTMLSpanElement;

let baseValidator = (oncomplete: (valid: boolean) => void): void => {
    if (oldPasswordField.value.length == 0) return oncomplete(false);
    oncomplete(true);
};

let validatorConf: ValidatorConf = {
    passwordField: newPasswordField,
    rePasswordField: rePasswordField,
    submitButton: changePasswordButton,
    validatorFunc: baseValidator
};

let validator = new Validator(validatorConf);
// let requirements = validator.requirements;

oldPasswordField.addEventListener("keyup", validator.validate);
changePasswordButton.addEventListener("click", () => {
    addLoader(changePasswordButton);
    _post("/my/password", { old: oldPasswordField.value, new: newPasswordField.value }, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        removeLoader(changePasswordButton);
        if (req.status == 400) {
            window.notifications.customError("errorPassword", window.lang.notif("errorPassword"));
        } else if (req.status == 500) {
            window.notifications.customError("errorUnknown", window.lang.notif("errorUnknown"));
        } else if (req.status == 204) {
            window.notifications.customSuccess("passwordChanged", window.lang.notif("passwordChanged"));
            setTimeout(() => { window.location.reload() }, 2000);
        }
    }, true, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        if (req.status == 401) {
            removeLoader(changePasswordButton);
            window.notifications.customError("oldPasswordError", window.lang.notif("errorOldPassword"));
            return;
        }
    });
});

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
            const contactMethods: { name: string, icon: string, f: (add: boolean) => void, required: boolean, enabled: boolean }[] = [
                {name: "email", icon: `<i class="ri-mail-fill ri-lg"></i>`, f: addEditEmail, required: true, enabled: true},
                {name: "discord", icon: `<i class="ri-discord-fill ri-lg"></i>`, f: (add: boolean) => { discord.onclick(); }, required: window.discordRequired, enabled: window.discordEnabled},
                {name: "telegram", icon: `<i class="ri-telegram-fill ri-lg"></i>`, f: (add: boolean) => { telegram.onclick() }, required: window.telegramRequired, enabled: window.telegramEnabled},
                {name: "matrix", icon: `<span class="font-bold">[m]</span>`, f: (add: boolean) => { matrix.show(); }, required: window.matrixRequired, enabled: window.matrixEnabled}
            ];
            
            for (let method of contactMethods) {
                if (!(method.enabled)) continue;
                if (method.name in details) {
                    contactMethodList.append(method.name, details[method.name], method.icon, method.f, method.required);
                }
            }

            expiryCard.expiry = details.expiry;

            const adminBackButton = document.getElementById("admin-back-button") as HTMLAnchorElement;
            adminBackButton.href = window.location.href.replace("my/account", "");

            let messageCard = document.getElementById("card-message");
            if (details.accounts_admin) {
                adminBackButton.classList.remove("unfocused");
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
                messageCard.innerHTML = messageCard.innerHTML.replace(new RegExp("{username}", "g"), details.username);
                // setBestRowSpan(messageCard, false);
                // contactCard.querySelector(".content").classList.add("h-100");
            } else if (!statusCard.classList.contains("unfocused")) {
                // setBestRowSpan(passwordCard, true);
            }

            if (window.referralsEnabled) {
                if (details.has_referrals) {
                    _get("/my/referral", null, (req: XMLHttpRequest) => {
                        if (req.readyState != 4 || req.status != 200) return; 
                        const referral: MyReferral = req.response as MyReferral;
                        referralCard.update(referral);
                        setCardOrder(messageCard);
                    });
                } else {
                    referralCard.hide();
                    setCardOrder(messageCard);
                }
            } else {
                setCardOrder(messageCard);
            }
        }
    });
});

const setCardOrder = (messageCard: HTMLElement) => {
    const cards = document.getElementById("user-cardlist");
    const children = Array.from(cards.children);
    const idxs = [...Array(cards.childElementCount).keys()]
    // The message card is the first element and should always be so, so remove it from the list.
    const hasMessageCard = !(typeof(messageCard) == "undefined" || messageCard == null);
    if (hasMessageCard) idxs.shift();
    const perms = generatePermutations(idxs);
    let minHeight = 999999;
    let minHeightPerm: [number[], number[]];
    for (let perm of perms) {
        let leftHeight = 0;
        for (let idx of perm[0]) {
            leftHeight += (cards.children[idx] as HTMLElement).offsetHeight;
        }
        if (hasMessageCard) leftHeight += (cards.children[0] as HTMLElement).offsetHeight;
        let rightHeight = 0;
        for (let idx of perm[1]) {
            rightHeight += (cards.children[idx] as HTMLElement).offsetHeight;
        }
        let height = Math.max(leftHeight, rightHeight);
        // console.log("got height", leftHeight, rightHeight, height, "for", perm);
        if (height < minHeight) {
            minHeight = height;
            minHeightPerm = perm;
        }
    }

    const gapDiv = () => {
        const g = document.createElement("div");
        g.classList.add("my-4");
        return g;
    };

    let addValue = hasMessageCard ? 1 : 0;
    // if (hasMessageCard) cards.appendChild(children[0]);
    if (hasMessageCard) cards.appendChild(gapDiv());
    for (let side of minHeightPerm) {
        for (let i = 0; i < side.length; i++) {
            // (cards.children[side[i]] as HTMLElement).style.order = (i+addValue).toString();
            children[side[i]].remove();
            cards.appendChild(children[side[i]]);
            cards.appendChild(gapDiv());
        }
        // addValue += side.length;
    }

    console.log("Shortest order:", minHeightPerm);
};

const login = new Login(window.modals.login as Modal, "/my/", "opaque");
login.onLogin = () => {
    console.log("Logged in.");
    document.querySelector(".page-container").classList.remove("unfocused");
    document.dispatchEvent(new CustomEvent("details-reload"));
};

const setBestRowSpan = (el: HTMLElement, setOnParent: boolean) => {
    let largestNonMessageCardHeight = 0;
    const cards = grid.querySelectorAll(".card") as NodeListOf<HTMLElement>;
    for (let i = 0; i < cards.length; i++) {
        if (cards[i].id == el.id) continue;
        if (computeRealHeight(cards[i]) > largestNonMessageCardHeight) {
            largestNonMessageCardHeight = computeRealHeight(cards[i]);
        }
    }

    let rowSpan = Math.ceil(computeRealHeight(el) / largestNonMessageCardHeight);

    if (rowSpan > 0)
        (setOnParent ? el.parentElement : el).style.gridRow = `span ${rowSpan}`;
};

const computeRealHeight = (el: HTMLElement): number => {
    let children = el.children as HTMLCollectionOf<HTMLElement>;
    let total = 0;
    for (let i = 0; i < children.length; i++) {
        // Cope with the contact method card expanding to fill, by counting each contact method individually
        if (el.id == "card-contact" && children[i].classList.contains("content")) {
            // console.log("FOUND CARD_CONTACT, OG:", total + children[i].offsetHeight);
            for (let j = 0; j < children[i].children.length; j++) {
                total += (children[i].children[j] as HTMLElement).offsetHeight;
            }
            // console.log("NEW:", total);
        } else {
            total += children[i].offsetHeight;
        }
    }
    return total;
}

const generatePermutations = (xs: number[]): [number[], number[]][] => {
    const l = xs.length;
    let out: [number[], number[]][] = [];
    for (let i = 0; i < (l << 1); i++) {
        let incl = [];
        let excl = [];
        for (let j = 0; j < l; j++) {
            if (i & (1 << j)) {
                incl.push(xs[j]);
            } else {
                excl.push(xs[j]);
            }
        }
        out.push([incl, excl]);
    }
    return out;
}

login.bindLogout(document.getElementById("logout-button"));

login.login("", "");
