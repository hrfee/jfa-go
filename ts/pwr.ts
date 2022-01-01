import { Modal } from "./modules/modal.js";
import { initValidator } from "./modules/validator.js";
import { _post, addLoader, removeLoader } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";

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
}

loadLangSelector("pwr");

declare var window: formWindow;

const form = document.getElementById("form-create") as HTMLFormElement;
const submitButton = form.querySelector("input[type=submit]") as HTMLInputElement;
const submitSpan = form.querySelector("span.submit") as HTMLSpanElement;
const passwordField = document.getElementById("create-password") as HTMLInputElement;
const rePasswordField = document.getElementById("create-reenter-password") as HTMLInputElement;

window.successModal = new Modal(document.getElementById("modal-success"), true);

var requirements = initValidator(passwordField, rePasswordField, submitButton, submitSpan)

interface sendDTO {
    pin: string;
    password: string;
}

form.onsubmit = (event: Event) => {
    event.preventDefault();
    addLoader(submitSpan);
    const params = new URLSearchParams(window.location.search);
    let send: sendDTO = {
        pin: params.get("pin"),
        password: passwordField.value
    };
    _post("/reset", send, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            removeLoader(submitSpan);
            if (req.status == 400) {
                for (let type in req.response) {
                    if (requirements[type]) { requirements[type].valid = req.response[type] as boolean; }
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
