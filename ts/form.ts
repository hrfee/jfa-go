import { Modal } from "./modules/modal.js";
import { notificationBox, whichAnimationEvent } from "./modules/common.js";
import { _get, _post, toggleLoader, addLoader, removeLoader, toDateString } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";
import { Validator, ValidatorConf, ValidatorRespDTO } from "./modules/validator.js";
import { Discord, Telegram, Matrix, ServiceConfiguration, MatrixConfiguration } from "./modules/account-linking.js";

interface formWindow extends Window {
    invalidPassword: string;
    successModal: Modal;
    telegramModal: Modal;
    discordModal: Modal;
    matrixModal: Modal;
    confirmationModal: Modal;
    redirectToJellyfin: boolean;
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
    matrixRequired: boolean;
    matrixUserID: string;
    userExpiryEnabled: boolean;
    userExpiryMonths: number;
    userExpiryDays: number;
    userExpiryHours: number;
    userExpiryMinutes: number;
    userExpiryMessage: string;
    emailRequired: boolean;
    captcha: boolean;
    reCAPTCHA: boolean;
    reCAPTCHASiteKey: string;
    userPageEnabled: boolean;
}

loadLangSelector("form");

window.notifications = new notificationBox(document.getElementById("notification-box") as HTMLDivElement);

window.animationEvent = whichAnimationEvent();

window.successModal = new Modal(document.getElementById("modal-success"), true);


var telegramVerified = false;
if (window.telegramEnabled) {
    window.telegramModal = new Modal(document.getElementById("modal-telegram"), window.telegramRequired);
    const telegramButton = document.getElementById("link-telegram") as HTMLSpanElement;

    const telegramConf: ServiceConfiguration = {
        modal: window.telegramModal as Modal,
        pin: window.telegramPIN,
        pinURL: "",
        verifiedURL: "/invite/" + window.code + "/telegram/verified/",
        invalidCodeError: window.messages["errorInvalidPIN"],
        accountLinkedError: window.messages["errorAccountLinked"],
        successError: window.messages["verified"],
        successFunc: (modalClosed: boolean) => {
            if (modalClosed) return;
            telegramVerified = true;
            telegramButton.classList.add("unfocused");
            document.getElementById("contact-via").classList.remove("unfocused");
            document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
            const radio = document.getElementById("contact-via-telegram") as HTMLInputElement;
            radio.parentElement.classList.remove("unfocused");
            radio.checked = true;
            validator.validate();
        }
    };

    const telegram = new Telegram(telegramConf);

    telegramButton.onclick = () => { telegram.onclick(); };
}

var discordVerified = false;
if (window.discordEnabled) {
    window.discordModal = new Modal(document.getElementById("modal-discord"), window.discordRequired);
    const discordButton = document.getElementById("link-discord") as HTMLSpanElement;
    
    const discordConf: ServiceConfiguration = {
        modal: window.discordModal as Modal,
        pin: window.discordPIN,
        inviteURL: window.discordInviteLink ? ("/invite/" + window.code + "/discord/invite") : "",
        pinURL: "",
        verifiedURL: "/invite/" + window.code + "/discord/verified/",
        invalidCodeError: window.messages["errorInvalidPIN"],
        accountLinkedError: window.messages["errorAccountLinked"],
        successError: window.messages["verified"],
        successFunc: (modalClosed: boolean) => {
            if (modalClosed) return;
            discordVerified = true;
            discordButton.classList.add("unfocused");
            document.getElementById("contact-via").classList.remove("unfocused");
            document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
            const radio = document.getElementById("contact-via-discord") as HTMLInputElement;
            radio.parentElement.classList.remove("unfocused")
            radio.checked = true;
            validator.validate();
        }
    };

    const discord = new Discord(discordConf);

    discordButton.onclick = () => { discord.onclick(); };
}

var matrixVerified = false;
var matrixPIN = "";
if (window.matrixEnabled) {
    window.matrixModal = new Modal(document.getElementById("modal-matrix"), window.matrixRequired);
    const matrixButton = document.getElementById("link-matrix") as HTMLSpanElement;
    
    const matrixConf: MatrixConfiguration = {
        modal: window.matrixModal as Modal,
        sendMessageURL: "/invite/" + window.code + "/matrix/user",
        verifiedURL: "/invite/" + window.code + "/matrix/verified/",
        invalidCodeError: window.messages["errorInvalidPIN"],
        accountLinkedError: window.messages["errorAccountLinked"],
        unknownError: window.messages["errorUnknown"],
        successError: window.messages["verified"],
        successFunc: () => {
            matrixVerified = true;
            matrixPIN = matrix.pin;
            matrixButton.classList.add("unfocused");
            document.getElementById("contact-via").classList.remove("unfocused");
            document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
            const radio = document.getElementById("contact-via-matrix") as HTMLInputElement;
            radio.parentElement.classList.remove("unfocused");
            radio.checked = true;
            validator.validate();
        }
    };

    const matrix = new Matrix(matrixConf);

    matrixButton.onclick = () => { matrix.show(); };
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

const form = document.getElementById("form-create") as HTMLFormElement;
const submitInput = form.querySelector("input[type=submit]") as HTMLInputElement;
const submitSpan = form.querySelector("span.submit") as HTMLSpanElement;
const submitText = submitSpan.textContent;
let usernameField = document.getElementById("create-username") as HTMLInputElement;
const emailField = document.getElementById("create-email") as HTMLInputElement;
if (!window.usernameEnabled) { usernameField.parentElement.remove(); usernameField = emailField; }
const passwordField = document.getElementById("create-password") as HTMLInputElement;
const rePasswordField = document.getElementById("create-reenter-password") as HTMLInputElement;

let captchaVerified = false;
let captchaID = "";
let captchaInput = document.getElementById("captcha-input") as HTMLInputElement;
const captchaCheckbox = document.getElementById("captcha-success") as HTMLSpanElement;
let prevCaptcha = "";

let baseValidator = (oncomplete: (valid: boolean) => void): void => {
    if (window.captcha && !window.reCAPTCHA && (captchaInput.value != prevCaptcha)) {
        prevCaptcha = captchaInput.value;
        _post("/captcha/verify/" + window.code + "/" + captchaID + "/" + captchaInput.value, null, (req: XMLHttpRequest) => {
            if (req.readyState == 4) {
                if (req.status == 204) {
                    captchaCheckbox.innerHTML = `<i class="ri-check-line"></i>`;
                    captchaCheckbox.classList.add("~positive");
                    captchaCheckbox.classList.remove("~critical");
                    captchaVerified = true;
                } else {
                    captchaCheckbox.innerHTML = `<i class="ri-close-line"></i>`;
                    captchaCheckbox.classList.add("~critical");
                    captchaCheckbox.classList.remove("~positive");
                    captchaVerified = false;
                }
                _baseValidator(oncomplete, captchaVerified);
            }
        });
    } else {
        _baseValidator(oncomplete, captchaVerified);
    }
}

function _baseValidator(oncomplete: (valid: boolean) => void, captchaValid: boolean): void {
    if (window.emailRequired) {
        if (!emailField.value.includes("@")) {
            oncomplete(false);
            return;
        }
    }
    if (window.discordEnabled && window.discordRequired && !discordVerified) {
        oncomplete(false);
        return;
    }
    if (window.telegramEnabled && window.telegramRequired && !telegramVerified) {
        oncomplete(false);
        return;
    }
    if (window.matrixEnabled && window.matrixRequired && !matrixVerified) {
        oncomplete(false);
        return;
    }
    if (window.captcha && !window.reCAPTCHA && !captchaValid) {
        oncomplete(false);
        return;
    }
    oncomplete(true);
}

interface GreCAPTCHA {
    render: (container: HTMLDivElement, parameters: {
        sitekey?: string,
        theme?: string,
        size?: string,
        tabindex?: number,
        "callback"?: () => void,
        "expired-callback"?: () => void,
        "error-callback"?: () => void
    }) => void;
    getResponse: (opt_widget_id?: HTMLDivElement) => string;
}

declare var grecaptcha: GreCAPTCHA

let validatorConf: ValidatorConf = {
    passwordField: passwordField,
    rePasswordField: rePasswordField,
    submitInput: submitInput,
    submitButton: submitSpan,
    validatorFunc: baseValidator
};

let validator = new Validator(validatorConf);
var requirements = validator.requirements;

if (window.emailRequired) {
    emailField.addEventListener("keyup", validator.validate)
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
    matrix_pin?: string;
    matrix_contact?: boolean;
    captcha_id?: string;
    captcha_text?: string;
}

const genCaptcha = () => {
    _get("/captcha/gen/"+window.code, null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200) {
                captchaID = req.response["id"];
                document.getElementById("captcha-img").innerHTML = `
                <img class="w-100" src="${window.location.toString().substring(0, window.location.toString().lastIndexOf("/invite"))}/captcha/img/${window.code}/${captchaID}"></img>
                `;
                captchaInput.value = "";
            }
        }
    });
};

if (window.captcha && !window.reCAPTCHA) {
    genCaptcha();
    (document.getElementById("captcha-regen") as HTMLSpanElement).onclick = genCaptcha;
    captchaInput.onkeyup = validator.validate;
}

const create = (event: SubmitEvent) => {
    event.preventDefault();
    if (window.captcha && !window.reCAPTCHA && !captchaVerified) {
        
    }
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
    if (matrixVerified) {
        send.matrix_pin = matrixPIN;
        const radio = document.getElementById("contact-via-matrix") as HTMLInputElement;
        if (radio.checked) {
            send.matrix_contact = true;
        }
    }
    if (window.captcha) {
        if (window.reCAPTCHA) {
            send.captcha_text = grecaptcha.getResponse();
        } else {
            send.captcha_id = captchaID;
            send.captcha_text = captchaInput.value;
        }
    }
    _post("/newUser", send, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            let vals = req.response as ValidatorRespDTO;
            let valid = true;
            for (let type in vals) {
                if (requirements[type]) requirements[type].valid = vals[type];
                if (!vals[type]) valid = false;
            }
            if (req.status == 200 && valid) {
                if (window.redirectToJellyfin == true) {
                    const url = ((document.getElementById("modal-success") as HTMLDivElement).querySelector("a.submit") as HTMLAnchorElement).href;
                    window.location.href = url;
                } else {
                    if (window.userPageEnabled) {
                        const userPageNoticeArea = document.getElementById("modal-success-user-page-area");
                        userPageNoticeArea.textContent = userPageNoticeArea.textContent.replace("{myAccount}", userPageNoticeArea.getAttribute("my-account-term"));
                    }
                    window.successModal.show();
                }
            } else {
                submitSpan.classList.add("~critical");
                submitSpan.classList.remove("~urge");
                if (req.response["error"] as string) {
                    submitSpan.textContent = window.messages[req.response["error"]];
                } else {
                    submitSpan.textContent = window.messages["errorPassword"];
                }
                setTimeout(() => {
                    submitSpan.classList.add("~urge");
                    submitSpan.classList.remove("~critical");
                    submitSpan.textContent = submitText;
                }, 1000);
            }
        }
    }, true, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            toggleLoader(submitSpan);
            if (req.status == 401 || req.status == 400) {
                if (req.response["error"] as string) {
                    if (req.response["error"] == "confirmEmail") {
                        window.confirmationModal.show();
                        return;
                    }
                    if (req.response["error"] in window.messages) {
                        submitSpan.textContent = window.messages[req.response["error"]];
                    } else {
                        submitSpan.textContent = req.response["error"];
                    }
                    setTimeout(() => { submitSpan.textContent = submitText; }, 1000);
                }
            }
        }
    });
};

validator.validate();

form.onsubmit = create;
