import { Modal } from "../modules/modal.js";
import { _get, _post, toggleLoader } from "../modules/common.js";

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

export interface DiscordConfiguration {
    modal: Modal;
    pin: string;
    inviteURL: string;
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

export class Discord {
    private _conf: DiscordConfiguration;
    private _pinAcquired = false;
    private _modalClosed = false;
    private _waiting = document.getElementById("discord-waiting") as HTMLSpanElement;
    private _verified = false;

    get verified(): boolean { return this._verified; }

    constructor(conf: DiscordConfiguration) {
        this._conf = conf;
        this._conf.modal.onclose = () => {
            this._modalClosed = true;
            toggleLoader(this._waiting);
        };
    }

    private _getInviteURL = () => _get(this._conf.inviteURL, null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        const inv = req.response as DiscordInvite;
        const link = document.getElementById("discord-invite") as HTMLAnchorElement;
        link.href = inv.invite;
        link.target = "_blank";
        if (inv.icon != "") {
            link.innerHTML = `<span class="img-circle lg mr-4"><img class="img-circle" src="${inv.icon}" width="64" height="64"></span>${window.discordServerName}`;
        } else {
            link.innerHTML = `
            <span class="shield mr-4 bg-discord"><i class="ri-discord-fill ri-xl text-white"></i></span>${window.discordServerName}
            `;
        }
    });

    private _checkVerified = () => {
        if (this._modalClosed) return;
        if (!this._pinAcquired) {
            setTimeout(this._checkVerified, 1500);
            return;
        }
        _get(this._conf.verifiedURL + this._conf.pin, null, (req: XMLHttpRequest) => {
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
                    window.notifications.customPositive("discordVerified", "", this._conf.successError); 
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

    onclick = () => {
        if (this._conf.inviteURL != "") {
            this._getInviteURL();
        }
        
        toggleLoader(this._waiting);

        this._pinAcquired = false;
        if (this._conf.pin) {
            this._pinAcquired = true;
            this._conf.modal.modal.querySelector(".pin").textContent = this._conf.pin;
        } else {
            _get(this._conf.pinURL, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4 && req.status == 200) {
                    this._conf.pin = req.response["pin"];
                    this._conf.modal.modal.querySelector(".pin").textContent = this._conf.pin;
                    this._pinAcquired = true;
                }
            });
        }

        this._modalClosed = false;
        this._conf.modal.show();

        this._checkVerified();
    }
}
