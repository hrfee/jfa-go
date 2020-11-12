import { serializeForm, rmAttr, addAttr, _get, _post, _delete } from "./modules/common.js";
import { Focus, Unfocus } from "./modules/admin.js";
import { toggleCSS } from "./modules/animation.js";
import { populateUsers, checkCheckboxes } from "./modules/accounts.js";
import { generateInvites, addOptions, checkDuration } from "./modules/invites.js";
import { showSetting, openSettings } from "./modules/settings.js";
import { BS4 } from "./modules/bs4.js";
import { BS5 } from "./modules/bs5.js";
import "./accounts.js";
import "./settings.js";

interface aWindow extends Window {
    toClipboard(str: string): void;
}

declare var window: aWindow;

interface TabSwitcher {
    els: Array<HTMLDivElement>;
    tabButtons: Array<HTMLAnchorElement>;
    focus: (el: number) => void;
    invites: () => void;
    accounts: () => void;
    settings: () => void;
}

const tabs: TabSwitcher = {
    els: [document.getElementById('invitesTab') as HTMLDivElement, document.getElementById('accountsTab') as HTMLDivElement, document.getElementById('settingsTab') as HTMLDivElement],
    tabButtons: [document.getElementById('invitesTabButton') as HTMLAnchorElement, document.getElementById('accountsTabButton') as HTMLAnchorElement, document.getElementById('settingsTabButton') as HTMLAnchorElement],
    focus: (el: number): void => {
        for (let i = 0; i < tabs.els.length; i++) {
            if (i == el) {
                Focus(tabs.els[i]);
                addAttr(tabs.tabButtons[i], "active");
            } else {
                Unfocus(tabs.els[i]);
                rmAttr(tabs.tabButtons[i], "active");
            }
        }
    },
    invites: (): void => tabs.focus(0),
    accounts: (): void => {
        populateUsers();
        (document.getElementById('selectAll') as HTMLInputElement).checked = false;
        checkCheckboxes();
        tabs.focus(1);
    },
    settings: (): void => openSettings(document.getElementById('settingsSections'), document.getElementById('settingsContent'), (): void => {
        window.BS.triggerTooltips();
        showSetting("ui");
        tabs.focus(2);
    })
};

window.bsVersion = window.bs5 ? 5 : 4

if (window.bs5) {
    window.BS = new BS5;
} else {
    window.BS = new BS4;
    window.BS.Compat();
}

window.Modals = {} as BSModals;

window.Modals.login = window.BS.newModal('login');
window.Modals.userDefaults = window.BS.newModal('userDefaults');
window.Modals.users = window.BS.newModal('users');
window.Modals.restart = window.BS.newModal('restartModal');
window.Modals.refresh = window.BS.newModal('refreshModal');
window.Modals.about = window.BS.newModal('aboutModal');
window.Modals.delete = window.BS.newModal('deleteModal');
window.Modals.newUser = window.BS.newModal('newUserModal');

tabs.tabButtons[0].onclick = tabs.invites;
tabs.tabButtons[1].onclick = tabs.accounts;
tabs.tabButtons[2].onclick = tabs.settings;

tabs.invites();

// Predefined colors for the theme button.
var buttonColor: string = "custom";
if (window.cssFile.includes("jf")) {
    buttonColor = "rgb(255,255,255)";
} else if (window.cssFile == ("bs" + window.bsVersion + ".css")) {
    buttonColor = "rgb(16,16,16)";
}

if (buttonColor != "custom") {
    const switchButton = document.createElement('button') as HTMLButtonElement;
    switchButton.classList.add('btn', 'btn-secondary');
    switchButton.innerHTML = `
    Theme
    <i class="fa fa-circle circle" style="color: ${buttonColor}; margin-left: 0.4rem;" id="fakeButton"></i>
    `;
    switchButton.onclick = (): void => toggleCSS(document.getElementById('fakeButton'));
    document.getElementById('headerButtons').appendChild(switchButton);
}

var availableProfiles: Array<string>;

window["token"] = "";

window.toClipboard = (str: string): void => {
    const el = document.createElement('textarea') as HTMLTextAreaElement;
    el.value = str;
    el.readOnly = true;
    el.style.position = "absolute";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    const selected = document.getSelection().rangeCount > 0 ? document.getSelection().getRangeAt(0) : false;
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
    if (selected) {
        document.getSelection().removeAllRanges();
        document.getSelection().addRange(selected);
    }
}

function login(username: string, password: string, modal: boolean, button?: HTMLButtonElement, run?: (arg0: number) => void): void {
    const req = new XMLHttpRequest();
    req.responseType = 'json';
    let url = "/token/login";
    const refresh = (username == "" && password == "");
    if (refresh) {
        url = "/token/refresh";
    }
    req.open("GET", url, true);
    if (!refresh) {
        req.setRequestHeader("Authorization", "Basic " + btoa(username + ":" + password));
    }
    req.onreadystatechange = function (): void {
        if (this.readyState == 4) {
            if (this.status != 200) {
                let errorMsg = this.response["error"];
                if (!errorMsg) {
                    errorMsg = "Unknown error";
                }
                if (modal) {
                    button.disabled = false;
                    button.textContent = errorMsg;
                    addAttr(button, "btn-danger");
                    rmAttr(button, "btn-primary");
                    setTimeout((): void => {
                        addAttr(button, "btn-primary");
                        rmAttr(button, "btn-danger");
                        button.textContent = "Login";
                    }, 4000);
                } else {
                    window.Modals.login.show();
                }
            } else {
                const data = this.response;
                window.token = data["token"];
                generateInvites();
                setInterval((): void => generateInvites(), 60 * 1000);
                addOptions(30, document.getElementById('days') as HTMLSelectElement);
                addOptions(24, document.getElementById('hours') as HTMLSelectElement);
                const minutes = document.getElementById('minutes') as HTMLSelectElement;
                addOptions(59, minutes);
                minutes.value = "30";
                checkDuration();
                if (modal) {
                    window.Modals.login.hide();
                }
                Focus(document.getElementById('logoutButton'));
            }
            if (run) {
                run(+this.status);
            }
        }
    };
    req.send();
}

(document.getElementById('loginForm') as HTMLFormElement).onsubmit = function (): boolean {
    window.token = "";
    const details = serializeForm('loginForm');
    const button = document.getElementById('loginSubmit') as HTMLButtonElement;
    addAttr(button, "btn-primary");
    rmAttr(button, "btn-danger");
    button.disabled = true;
    button.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
    Loading...`;
    login(details["username"], details["password"], true, button);
    return false;
};

generateInvites(true);

login("", "", false, null, (status: number): void => {
    if (!(status == 200 || status == 204)) {
        window.Modals.login.show();
    }
});

(document.getElementById('logoutButton') as HTMLButtonElement).onclick = function (): void {
    _post("/logout", null, function (): boolean {
        if (this.readyState == 4 && this.status == 200) {
            window.token = "";
            location.reload();
            return false;
        }
    });
};


