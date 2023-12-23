import { Modal } from "./modules/modal.js";
import { Validator, ValidatorConf } from "./modules/validator.js";
import { _post, addLoader, removeLoader } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";
import { Captcha, GreCAPTCHA } from "./modules/captcha.js";

interface formWindow extends Window {
    invalidPassword: string;
    successModal: Modal;
    telegramModal: Modal;
    discordModal: Modal;
    matrixModal: Modal;
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
    matrixRequired: boolean;
    matrixUserID: string;
    userExpiryEnabled: boolean;
    userExpiryMonths: number;
    userExpiryDays: number;
    userExpiryHours: number;
    userExpiryMinutes: number;
    userExpiryMessage: string;
    captcha: boolean;
    reCAPTCHA: boolean;
    reCAPTCHASiteKey: string;
    pwrPIN: string;
}

loadLangSelector("pwr");

declare var window: formWindow;

const form = document.getElementById("form-create") as HTMLFormElement;
const submitInput = form.querySelector("input[type=submit]") as HTMLInputElement;
const submitSpan = form.querySelector("span.submit") as HTMLSpanElement;
const passwordField = document.getElementById("create-password") as HTMLInputElement;
const rePasswordField = document.getElementById("create-reenter-password") as HTMLInputElement;

window.successModal = new Modal(document.getElementById("modal-success"), true);

function _baseValidator(oncomplete: (valid: boolean) => void, captchaValid: boolean): void {
    if (window.captcha && !window.reCAPTCHA && !captchaValid) {
        oncomplete(false);
        return;
    }
    oncomplete(true);
}

let captcha = new Captcha(window.pwrPIN, window.captcha, window.reCAPTCHA, true);

declare var grecaptcha: GreCAPTCHA;

let baseValidator = captcha.baseValidatorWrapper(_baseValidator);

let validatorConf: ValidatorConf = {
    passwordField: passwordField,
    rePasswordField: rePasswordField,
    submitInput: submitInput,
    submitButton: submitSpan,
    validatorFunc: baseValidator
};

var validator = new Validator(validatorConf);
var requirements = validator.requirements;

interface sendDTO {
    pin: string;
    password: string;
    captcha_text?: string;
}

if (window.captcha && !window.reCAPTCHA) {
    captcha.generate();
    (document.getElementById("captcha-regen") as HTMLSpanElement).onclick = captcha.generate;
    captcha.input.onkeyup = validator.validate;
}

form.onsubmit = (event: Event) => {
    event.preventDefault();
    addLoader(submitSpan);
    const params = new URLSearchParams(window.location.search);
    let send: sendDTO = {
        pin: params.get("pin"),
        password: passwordField.value
    };
    if (window.captcha) {
        if (window.reCAPTCHA) {
            send.captcha_text = grecaptcha.getResponse();
        } else {
            send.captcha_text = captcha.input.value;
        }
    }
    _post("/reset", send, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            removeLoader(submitSpan);
            if (req.status == 400) {
                if (req.response["error"] as string) {
                    const old = submitSpan.textContent;
                    submitSpan.textContent = window.messages[req.response["error"]];
                    submitSpan.classList.add("~critical");
                    submitSpan.classList.remove("~urge");
                    setTimeout(() => {
                        submitSpan.classList.add("~urge");
                        submitSpan.classList.remove("~critical");
                        submitSpan.textContent = old;
                    }, 2000);
                } else {
                    for (let type in req.response) {
                        if (requirements[type]) { requirements[type].valid = req.response[type] as boolean; }
                    }
                }
                return;
            } else if (req.status != 200) {
                const old = submitSpan.textContent;
                submitSpan.textContent = window.messages["errorUnknown"];
                submitSpan.classList.add("~critical");
                submitSpan.classList.remove("~urge");
                setTimeout(() => {
                    submitSpan.classList.add("~urge");
                    submitSpan.classList.remove("~critical");
                    submitSpan.textContent = old;
                }, 2000);
            } else {
                window.successModal.show();
            }
        }
    }, true);
};

validator.validate();
