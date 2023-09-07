import { Modal } from "../modules/modal.js";
import { _get, _post, toggleLoader, addLoader, removeLoader } from "../modules/common.js";

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
}

declare var window: formWindow;

export interface ServiceConfiguration {
    modal: Modal;
    pin: string;
    inviteURL?: string;
    pinURL: string;
    verifiedURL: string;
    invalidCodeError: string;
    accountLinkedError: string;
    successError: string;
    successFunc: (modalClosed: boolean) => void;
};

export interface DiscordInvite {
    invite: string;
    icon: string;
}

export class ServiceLinker {
    protected _conf: ServiceConfiguration;
    protected _pinAcquired = false;
    protected _modalClosed = false;
    protected _waiting: HTMLSpanElement;
    protected _verified = false;
    protected _name: string;
    protected _pin: string;

    get verified(): boolean { return this._verified; }

    constructor(conf: ServiceConfiguration) {
        this._conf = conf;
        this._conf.modal.onclose = () => {
            this._modalClosed = true;
            toggleLoader(this._waiting);
        };
    }

    protected _checkVerified = () => {
        if (this._modalClosed) return;
        if (!this._pinAcquired) {
            setTimeout(this._checkVerified, 1500);
            return;
        }
        _get(this._conf.verifiedURL + this._pin, null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status == 401) {
                this._conf.modal.close();
                window.notifications.customError("invalidCodeError", this._conf.invalidCodeError);
            } else if (req.status == 400) {
                this._conf.modal.close();
                window.notifications.customError("accountLinkedError", this._conf.accountLinkedError);
            } else if (req.status == 200) {
                if (req.response["success"] as boolean) {
                    this._verified = true;
                    this._waiting.classList.add("~positive");
                    this._waiting.classList.remove("~info");
                    window.notifications.customPositive(this._name + "Verified", "", this._conf.successError); 
                    if (this._conf.successFunc) {
                        this._conf.successFunc(false);
                    }
                    setTimeout(() => {
                        this._conf.modal.close();
                        if (this._conf.successFunc) {
                            this._conf.successFunc(true);
                        }
                    }, 2000);

                } else if (!this._modalClosed) {
                    setTimeout(this._checkVerified, 1500);
                }
            }
        });
    };

    onclick() {
        toggleLoader(this._waiting);

        this._pinAcquired = false;
        this._pin = "";
        if (this._conf.pin) {
            this._pinAcquired = true;
            this._pin = this._conf.pin;
            this._conf.modal.modal.querySelector(".pin").textContent = this._pin;
        } else if (this._conf.pinURL) {
            _get(this._conf.pinURL, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4 && req.status == 200) {
                    this._pin = req.response["pin"];
                    this._conf.modal.modal.querySelector(".pin").textContent = this._pin;
                    this._pinAcquired = true;
                }
            });
        }

        this._modalClosed = false;
        this._conf.modal.show();

        this._checkVerified();
    }
}

export class Discord extends ServiceLinker {

    constructor(conf: ServiceConfiguration) {
        super(conf);
        this._name = "discord";
        this._waiting = document.getElementById("discord-waiting") as HTMLSpanElement;
    }

    private _getInviteURL = () => _get(this._conf.inviteURL, null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        const inv = req.response as DiscordInvite;
        const link = document.getElementById("discord-invite") as HTMLSpanElement;
        (link.parentElement as HTMLAnchorElement).href = inv.invite;
        (link.parentElement as HTMLAnchorElement).target = "_blank";
        let innerHTML = ``;
        if (inv.icon != "") {
            innerHTML += `<span class="img-circle lg mr-4"><img class="img-circle" src="${inv.icon}" width="64" height="64"></span>${window.discordServerName}`;
        } else {
            innerHTML += `
            <span class="shield mr-4 bg-discord"><i class="ri-discord-fill ri-xl text-white"></i></span>${window.discordServerName}
            `;
        }
        link.innerHTML = innerHTML;
    });

    onclick() {
        if (this._conf.inviteURL != "") {
            this._getInviteURL();
        } else {
            (document.getElementById("discord-invite") as HTMLSpanElement).parentElement.remove();
        }

        super.onclick();
    }
}

export class Telegram extends ServiceLinker {
    constructor(conf: ServiceConfiguration) {
        super(conf);
        this._name = "telegram";
        this._waiting = document.getElementById("telegram-waiting") as HTMLSpanElement;
    }
};

export interface MatrixConfiguration {
    modal: Modal;
    sendMessageURL: string;
    verifiedURL: string;
    invalidCodeError: string;
    accountLinkedError: string;
    unknownError: string;
    successError: string;
    successFunc: () => void;
}

export class Matrix {
    private _conf: MatrixConfiguration;
    private _verified = false;
    private _name: string = "matrix";
    private _userID: string = "";
    private _pin: string = "";
    private _input: HTMLInputElement;
    private _submit: HTMLSpanElement;

    get verified(): boolean { return this._verified; }
    get pin(): string { return this._pin; }

    constructor(conf: MatrixConfiguration) {
        this._conf = conf;
        this._input = document.getElementById("matrix-userid") as HTMLInputElement;
        this._submit = document.getElementById("matrix-send") as HTMLSpanElement;
        this._submit.onclick = () => { this._onclick(); };
    }

    private _onclick = () => {
        addLoader(this._submit);
        if (this._userID == "") {
            this._sendMessage();
        } else {
            this._verifyCode();
        }
    };

    show = () => {
        this._input.value = "";
        this._conf.modal.show();
    }

    private _sendMessage = () => _post(this._conf.sendMessageURL, { "user_id": this._input.value }, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        removeLoader(this._submit);
        if (req.status == 400 && req.response["error"] == "errorAccountLinked") {
            this._conf.modal.close();
            window.notifications.customError("accountLinkedError", this._conf.accountLinkedError);
            return;
        } else if (req.status != 200) {
            this._conf.modal.close();
            window.notifications.customError("unknownError", this._conf.unknownError);
            return;
        }
        this._userID = this._input.value;
        this._submit.classList.add("~positive");
        this._submit.classList.remove("~info");
        setTimeout(() => {
            this._submit.classList.add("~info");
            this._submit.classList.remove("~positive");
        }, 2000);
        this._input.placeholder = "PIN";
        this._input.value = "";
    });

    private _verifyCode = () => _get(this._conf.verifiedURL + this._userID + "/" + this._input.value, null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        removeLoader(this._submit);
        const valid = req.response["success"] as boolean;
        if (valid) {
            this._conf.modal.close();
            window.notifications.customPositive(this._name + "Verified", "", this._conf.successError); 
            this._verified = true;
            this._pin = this._input.value;
            if (this._conf.successFunc) {
                this._conf.successFunc();
            }
        } else {
            window.notifications.customError("invalidCodeError", this._conf.invalidCodeError);
            this._submit.classList.add("~critical");
            this._submit.classList.remove("~info");
            setTimeout(() => {
                this._submit.classList.add("~info");
                this._submit.classList.remove("~critical");
            }, 800);
        }
    });
}

