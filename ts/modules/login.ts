import { Modal } from "../modules/modal.js";
import { toggleLoader, _post } from "../modules/common.js";

export class Login {
    private _modal: Modal;
    private _form: HTMLFormElement;
    private _url: string;
    private _endpoint: string;
    private _onLogin: (username: string, password: string) => void;
    private _logoutButton: HTMLElement = null;
    private _wall: HTMLElement;
    private _hasOpacityWall: boolean = false;

    constructor(modal: Modal, endpoint: string, appearance: string) {
        this._endpoint = endpoint;
        this._url = window.URLBase + endpoint;
        if (this._url[this._url.length-1] != '/') this._url += "/";

        this._modal = modal;
        if (appearance == "opaque") {
            this._hasOpacityWall = true;
            this._wall = document.createElement("div");
            this._wall.classList.add("wall");
            this._modal.asElement().parentElement.appendChild(this._wall);
        }
        this._form = this._modal.asElement().querySelector(".form-login") as HTMLFormElement;
        this._form.onsubmit = (event: SubmitEvent) => {
            event.preventDefault();
            const button = (event.target as HTMLElement).querySelector(".submit") as HTMLSpanElement;
            const username = (document.getElementById("login-user") as HTMLInputElement).value;
            const password = (document.getElementById("login-password") as HTMLInputElement).value;
            if (!username || !password) {
                window.notifications.customError("loginError", window.lang.notif("errorLoginBlank"));
                return;
            }
            toggleLoader(button);
            this.login(username, password, () => toggleLoader(button));
        };
    }

    bindLogout = (button: HTMLElement) => {
        this._logoutButton = button;
        this._logoutButton.classList.add("unfocused");
        const logoutFunc = (url: string, tryAgain: boolean) => {
            _post(url + "logout", null, (req: XMLHttpRequest): boolean => {
                if (req.readyState == 4 && req.status == 200) {
                    window.token = "";
                    location.reload();
                    return false;
                }
            }, false, (req: XMLHttpRequest) => {
                if (req.readyState == 4 && req.status == 404 && tryAgain) {
                    console.log("trying without URL Base...");
                    logoutFunc(this._endpoint, false);
                }
            });
        };
        this._logoutButton.onclick = () => logoutFunc(this._url, true);
    };

    get onLogin() { return this._onLogin; }
    set onLogin(f: (username: string, password: string) => void) { this._onLogin = f; }

    login = (username: string, password: string, run?: (state?: number) => void) => {
        const req = new XMLHttpRequest();
        req.responseType = 'json';
        const refresh = (username == "" && password == "");
        req.open("GET", this._url + (refresh ? "token/refresh" : "token/login"), true);
        if (!refresh) {
            req.setRequestHeader("Authorization", "Basic " + btoa(username + ":" + password));
        }
        req.onreadystatechange = ((req: XMLHttpRequest, _: Event): any => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    let errorMsg = window.lang.notif("errorConnection");
                    if (req.response) {
                        errorMsg = req.response["error"];
                        const langErrorMsg = window.lang.strings(errorMsg);
                        if (langErrorMsg) {
                            errorMsg = langErrorMsg;
                        }
                    }
                    if (!errorMsg) {
                        errorMsg = window.lang.notif("errorUnknown");
                    }
                    if (!refresh) {
                        window.notifications.customError("loginError", errorMsg);
                    } else {
                        this._modal.show();
                    }
                } else {
                    const data = req.response;
                    window.token = data["token"];
                    if (this._onLogin) {
                        this._onLogin(username, password);
                    }
                    if (this._hasOpacityWall) this._wall.remove();
                    this._modal.close();
                    if (this._logoutButton != null)
                        this._logoutButton.classList.remove("unfocused");
                }
                if (run) { run(+req.status); }
            }
        }).bind(this, req);
        req.send();
    };
}

