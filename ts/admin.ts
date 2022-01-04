import { nightwind } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { Tabs } from "./modules/tabs.js";
import { inviteList, createInvite } from "./modules/invites.js";
import { accountsList } from "./modules/accounts.js";
import { settingsList } from "./modules/settings.js";
import { ProfileEditor } from "./modules/profiles.js";
import { _get, _post, notificationBox, whichAnimationEvent, toggleLoader } from "./modules/common.js";
import { Updater } from "./modules/update.js";

let theme = new nightwind();

const themeButton = document.getElementById('button-theme') as HTMLSpanElement;
const switchThemeIcon = () => {
    const icon = themeButton.childNodes[0] as HTMLElement;
    if (document.documentElement.classList.contains("dark")) {
        icon.classList.add("ri-sun-line");
        icon.classList.remove("ri-moon-line");
        themeButton.classList.add("~warning");
        themeButton.classList.remove("~neutral");
        themeButton.classList.remove("@high");
    } else {
        icon.classList.add("ri-moon-line");
        icon.classList.remove("ri-sun-line");
        themeButton.classList.add("@high");
        themeButton.classList.add("~neutral");
        themeButton.classList.remove("~warning");
    }
};
 themeButton.onclick = () => {
    theme.toggle();
    switchThemeIcon();
 }
switchThemeIcon();


window.lang = new lang(window.langFile as LangFile);
loadLangSelector("admin");
// _get(`/lang/admin/${window.language}.json`, null, (req: XMLHttpRequest) => {
//     if (req.readyState == 4 && req.status == 200) {
//         langLoaded = true;
//         window.lang = new lang(req.response as LangFile); 
//     }
// });

window.animationEvent = whichAnimationEvent();

window.token = "";

window.availableProfiles = window.availableProfiles || [];

// load modals
(() => {
    window.modals = {} as Modals;

    window.modals.login = new Modal(document.getElementById('modal-login'), true);

    window.modals.addUser = new Modal(document.getElementById('modal-add-user'));

    window.modals.about = new Modal(document.getElementById('modal-about'));
    (document.getElementById('setting-about') as HTMLSpanElement).onclick = window.modals.about.toggle;

    window.modals.modifyUser = new Modal(document.getElementById('modal-modify-user'));

    window.modals.deleteUser = new Modal(document.getElementById('modal-delete-user'));

    window.modals.settingsRestart = new Modal(document.getElementById('modal-restart'));

    window.modals.settingsRefresh = new Modal(document.getElementById('modal-refresh'));

    window.modals.ombiProfile = new Modal(document.getElementById('modal-ombi-profile'));
    document.getElementById('form-ombi-defaults').addEventListener('submit', window.modals.ombiProfile.close);

    window.modals.profiles = new Modal(document.getElementById("modal-user-profiles"));

    window.modals.addProfile = new Modal(document.getElementById("modal-add-profile"));

    window.modals.announce = new Modal(document.getElementById("modal-announce"));
    
    window.modals.editor = new Modal(document.getElementById("modal-editor"));

    window.modals.customizeEmails = new Modal(document.getElementById("modal-customize"));

    window.modals.extendExpiry = new Modal(document.getElementById("modal-extend-expiry"));

    window.modals.updateInfo = new Modal(document.getElementById("modal-update"));

    window.modals.matrix = new Modal(document.getElementById("modal-matrix"));

    window.modals.logs = new Modal(document.getElementById("modal-logs"));

    if (window.telegramEnabled) {
        window.modals.telegram = new Modal(document.getElementById("modal-telegram"));
    }

    if (window.discordEnabled) {
        window.modals.discord = new Modal(document.getElementById("modal-discord"));
    }

    if (window.linkResetEnabled) {
        window.modals.sendPWR = new Modal(document.getElementById("modal-send-pwr"));
    }
})();

var inviteCreator = new createInvite();

var accounts = new accountsList();

window.invites = new inviteList();

var settings = new settingsList();

var profiles = new ProfileEditor();

window.notifications = new notificationBox(document.getElementById('notification-box') as HTMLDivElement, 5);

/*const modifySettingsSource = function () {
    const profile = document.getElementById('radio-use-profile') as HTMLInputElement;
    const user = document.getElementById('radio-use-user') as HTMLInputElement;
    const profileSelect = document.getElementById('modify-user-profiles') as HTMLDivElement;
    const userSelect = document.getElementById('modify-user-users') as HTMLDivElement;
    (user.nextElementSibling as HTMLSpanElement).classList.toggle('@low');
    (user.nextElementSibling as HTMLSpanElement).classList.toggle('@high');
    (profile.nextElementSibling as HTMLSpanElement).classList.toggle('@low');
    (profile.nextElementSibling as HTMLSpanElement).classList.toggle('@high');
    profileSelect.classList.toggle('unfocused');
    userSelect.classList.toggle('unfocused');
}*/

// load tabs
window.tabs = new Tabs();
window.tabs.addTab("invites", null, window.invites.reload);
window.tabs.addTab("accounts", null, accounts.reload);
window.tabs.addTab("settings", null, settings.reload);

for (let tab of ["invites", "accounts", "settings"]) {
    if (window.location.pathname == window.URLBase + "/" + tab) {
        window.tabs.switch(tab, true);
    }
}

if ((window.URLBase + "/").includes(window.location.pathname)) {
    window.tabs.switch("invites", true);
}

document.addEventListener("tab-change", (event: CustomEvent) => {
    const urlParams = new URLSearchParams(window.location.search);
    const lang = urlParams.get('lang');
    let tab = window.URLBase + "/" + event.detail;
    if (tab == window.URLBase + "/invites") {
        if (window.location.pathname == window.URLBase + "/") {
            tab = window.URLBase + "/";
        } else if (window.URLBase) { tab = window.URLBase; }
        else { tab = "../"; }
    }
    if (lang) {
        tab += "?lang=" + lang
    }
    window.history.pushState(event.detail, "Admin - jfa-go", tab);
});

window.onpopstate = (event: PopStateEvent) => {
    console.log(event.state);
    window.tabs.switch(event.state);
}

function login(username: string, password: string, run?: (state?: number) => void) {
    const req = new XMLHttpRequest();
    req.responseType = 'json';
    let url = window.URLBase;
    const refresh = (username == "" && password == "");
    if (refresh) {
        url += "/token/refresh";
    } else {
        url += "/token/login";
    }
    req.open("GET", url, true);
    if (!refresh) {
        req.setRequestHeader("Authorization", "Basic " + btoa(username + ":" + password));
    }
    req.onreadystatechange = function (): void {
        if (this.readyState == 4) {
            if (this.status != 200) {
                let errorMsg = window.lang.notif("errorConnection");
                if (this.response) {
                    errorMsg = this.response["error"];
                }
                if (!errorMsg) {
                    errorMsg = window.lang.notif("errorUnknown");
                }
                if (!refresh) {
                    window.notifications.customError("loginError", errorMsg);
                } else {
                    window.modals.login.show();
                }
            } else {
                const data = this.response;
                window.token = data["token"];
                window.updater = new Updater(); // mmm, a race condition
                window.modals.login.close();
                setInterval(() => { window.invites.reload(); accounts.reload(); }, 30*1000);
                const currentTab = window.tabs.current;
                switch (currentTab) {
                    case "invites":
                        window.invites.reload();
                        break;
                    case "accounts":
                        accounts.reload();
                        break;
                    case "settings":
                        settings.reload();
                        break;
                }
                document.getElementById("logout-button").classList.remove("unfocused");
            }
            if (run) { run(+this.status); }
        }
    };
    req.send();
}

(document.getElementById('form-login') as HTMLFormElement).onsubmit = (event: SubmitEvent) => {
    event.preventDefault();
    const button = (event.target as HTMLElement).querySelector(".submit") as HTMLSpanElement;
    const username = (document.getElementById("login-user") as HTMLInputElement).value;
    const password = (document.getElementById("login-password") as HTMLInputElement).value;
    if (!username || !password) {
        window.notifications.customError("loginError", window.lang.notif("errorLoginBlank"));
        return;
    }
    toggleLoader(button);
    login(username, password, () => toggleLoader(button));
};

login("", "");

(document.getElementById('logout-button') as HTMLButtonElement).onclick = () => _post("/logout", null, (req: XMLHttpRequest): boolean => {
    if (req.readyState == 4 && req.status == 200) {
        window.token = "";
        location.reload();
        return false;
    }
});
