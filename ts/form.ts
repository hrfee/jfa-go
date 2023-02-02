import { Modal } from "./modules/modal.js";
import { notificationBox, whichAnimationEvent } from "./modules/common.js";
import { _get, _post, toggleLoader, addLoader, removeLoader, toDateString } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";
import { initValidator } from "./modules/validator.js";

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
                } else if (req.status == 400) {
                    window.telegramModal.close();
                    window.notifications.customError("accountLinkedError", window.messages["errorAccountLinked"]);
                } else if (req.status == 200) {
                    if (req.response["success"] as boolean) {
                        telegramVerified = true;
                        waiting.classList.add("~positive");
                        waiting.classList.remove("~info");
                        window.notifications.customPositive("telegramVerified", "", window.messages["verified"]); 
                        setTimeout(window.telegramModal.close, 2000);
                        telegramButton.classList.add("unfocused");
                        document.getElementById("contact-via").classList.remove("unfocused");
                        document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
                        const radio = document.getElementById("contact-via-telegram") as HTMLInputElement;
                        radio.parentElement.classList.remove("unfocused");
                        radio.checked = true;
                        validatorFunc();
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
                link.innerHTML = `<span class="img-circle lg mr-4"><img class="img-circle" src="${inv.icon}" width="64" height="64"></span>${window.discordServerName}`;
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
                } else if (req.status == 400) {
                    window.discordModal.close();
                    window.notifications.customError("accountLinkedError", window.messages["errorAccountLinked"]);
                } else if (req.status == 200) {
                    if (req.response["success"] as boolean) {
                        discordVerified = true;
                        waiting.classList.add("~positive");
                        waiting.classList.remove("~info");
                        window.notifications.customPositive("discordVerified", "", window.messages["verified"]); 
                        setTimeout(window.discordModal.close, 2000);
                        discordButton.classList.add("unfocused");
                        document.getElementById("contact-via").classList.remove("unfocused");
                        document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
                        const radio = document.getElementById("contact-via-discord") as HTMLInputElement;
                        radio.parentElement.classList.remove("unfocused")
                        radio.checked = true;
                        validatorFunc();
                    } else if (!modalClosed) {
                        setTimeout(checkVerified, 1500);
                    }
                }
            }
        });
        checkVerified();
    };
}

var matrixVerified = false;
var matrixPIN = "";
if (window.matrixEnabled) {
    window.matrixModal = new Modal(document.getElementById("modal-matrix"), window.matrixRequired);
    const matrixButton = document.getElementById("link-matrix") as HTMLSpanElement;
    matrixButton.onclick = window.matrixModal.show;
    const submitButton = document.getElementById("matrix-send") as HTMLSpanElement;
    const input = document.getElementById("matrix-userid") as HTMLInputElement;
    let userID = "";
    submitButton.onclick = () => {
        addLoader(submitButton);
        if (userID == "") {
            const send = {
                user_id: input.value
            };
            _post("/invite/" + window.code + "/matrix/user", send, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status == 400 && req.response["error"] == "errorAccountLinked") {
                        window.matrixModal.close();
                        window.notifications.customError("accountLinkedError", window.messages["errorAccountLinked"]);
                    }
                    removeLoader(submitButton);
                    userID = input.value;
                    if (req.status != 200) {
                        window.notifications.customError("errorUnknown", window.messages["errorUnknown"]);
                        window.matrixModal.close();
                        return;
                    }
                    submitButton.classList.add("~positive");
                    submitButton.classList.remove("~info");
                    setTimeout(() => {
                        submitButton.classList.add("~info");
                        submitButton.classList.remove("~positive");
                    }, 2000);
                    input.placeholder = "PIN";
                    input.value = "";
                }
            });
        } else {
            _get("/invite/" + window.code + "/matrix/verified/" + userID + "/" + input.value, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    removeLoader(submitButton)
                    const valid = req.response["success"] as boolean;
                    if (valid) {
                        window.matrixModal.close();
                        window.notifications.customPositive("successVerified", "", window.messages["verified"]);
                        matrixVerified = true;
                        matrixPIN = input.value;
                        matrixButton.classList.add("unfocused");
                        document.getElementById("contact-via").classList.remove("unfocused");
                        document.getElementById("contact-via-email").parentElement.classList.remove("unfocused");
                        const radio = document.getElementById("contact-via-matrix") as HTMLInputElement;
                        radio.parentElement.classList.remove("unfocused");
                        radio.checked = true;
                        validatorFunc();
                    } else {
                        window.notifications.customError("errorInvalidPIN", window.messages["errorInvalidPIN"]);
                        submitButton.classList.add("~critical");
                        submitButton.classList.remove("~info");
                        setTimeout(() => {
                            submitButton.classList.add("~info");
                            submitButton.classList.remove("~critical");
                        }, 800);
                    }
                }
            },);
        }
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

const form = document.getElementById("form-create") as HTMLFormElement;
const submitButton = form.querySelector("input[type=submit]") as HTMLInputElement;
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

function baseValidator(oncomplete: (valid: boolean) => void): void {
    let captchaChecked = false;
    let captchaChange = false;
    if (window.captcha) {
        captchaChange = captchaInput.value != prevCaptcha;
        if (captchaChange) {
            prevCaptcha = captchaInput.value;
            _post("/captcha/verify/" + window.code + "/" + captchaID + "/" + captchaInput.value, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status == 204) {
                        captchaCheckbox.innerHTML = `<i class="ri-check-line"></i>`;
                        captchaCheckbox.classList.add("~positive");
                        captchaCheckbox.classList.remove("~critical");
                        captchaVerified = true;
                        captchaChecked = true;
                    } else {
                        captchaCheckbox.innerHTML = `<i class="ri-close-line"></i>`;
                        captchaCheckbox.classList.add("~critical");
                        captchaCheckbox.classList.remove("~positive");
                        captchaVerified = false;
                        captchaChecked = true;
                        return;
                    }
                }
            });
        }
    }
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
    if (window.captcha) {
        if (!captchaChange) {
            oncomplete(captchaVerified);
            return;
        }
        while (!captchaChecked) {
            continue;
        }
        oncomplete(captchaVerified);
    } else {
        oncomplete(true);
    }
}

let r = initValidator(passwordField, rePasswordField, submitButton, submitSpan, baseValidator);
var requirements = r[0];
var validatorFunc = r[1] as () => void;

if (window.emailRequired) {
    emailField.addEventListener("keyup", validatorFunc)
}

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

if (window.captcha) {
    genCaptcha();
    (document.getElementById("captcha-regen") as HTMLSpanElement).onclick = genCaptcha;
    captchaInput.onkeyup = validatorFunc;
}

const create = (event: SubmitEvent) => {
    event.preventDefault();
    if (window.captcha && !captchaVerified) {
        
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
        send.captcha_id = captchaID;
        send.captcha_text = captchaInput.value;
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
                if (window.redirectToJellyfin == true) {
                    const url = ((document.getElementById("modal-success") as HTMLDivElement).querySelector("a.submit") as HTMLAnchorElement).href;
                    window.location.href = url;
                } else {
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

validatorFunc();

form.onsubmit = create;
