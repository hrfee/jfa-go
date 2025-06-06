import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { Tabs, Tab } from "./modules/tabs.js";
import { inviteList, createInvite } from "./modules/invites.js";
import { accountsList } from "./modules/accounts.js";
import { settingsList } from "./modules/settings.js";
import { activityList } from "./modules/activity.js";
import { ProfileEditor, reloadProfileNames } from "./modules/profiles.js";
import { _get, _post, notificationBox, whichAnimationEvent, bindManualDropdowns } from "./modules/common.js";
import { Updater } from "./modules/update.js";
import { Login } from "./modules/login.js";

declare var window: GlobalWindow;

const theme = new ThemeManager(document.getElementById("button-theme"));

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
    
    window.modals.jellyseerrProfile = new Modal(document.getElementById('modal-jellyseerr-profile'));
    document.getElementById('form-jellyseerr-defaults').addEventListener('submit', window.modals.jellyseerrProfile.close);

    window.modals.profiles = new Modal(document.getElementById("modal-user-profiles"));

    window.modals.addProfile = new Modal(document.getElementById("modal-add-profile"));

    window.modals.announce = new Modal(document.getElementById("modal-announce"));
    
    window.modals.editor = new Modal(document.getElementById("modal-editor"));

    window.modals.customizeEmails = new Modal(document.getElementById("modal-customize"));

    window.modals.extendExpiry = new Modal(document.getElementById("modal-extend-expiry"));

    window.modals.updateInfo = new Modal(document.getElementById("modal-update"));

    window.modals.matrix = new Modal(document.getElementById("modal-matrix"));

    window.modals.logs = new Modal(document.getElementById("modal-logs"));

    window.modals.backedUp = new Modal(document.getElementById("modal-backed-up"));

    window.modals.backups = new Modal(document.getElementById("modal-backups"));

    if (window.telegramEnabled) {
        window.modals.telegram = new Modal(document.getElementById("modal-telegram"));
    }

    if (window.discordEnabled) {
        window.modals.discord = new Modal(document.getElementById("modal-discord"));
    }

    if (window.linkResetEnabled) {
        window.modals.sendPWR = new Modal(document.getElementById("modal-send-pwr"));
    }

    if (window.referralsEnabled) {
        window.modals.enableReferralsUser = new Modal(document.getElementById("modal-enable-referrals-user"));
        window.modals.enableReferralsProfile = new Modal(document.getElementById("modal-enable-referrals-profile"));
    }
})();

var inviteCreator = new createInvite();

var accounts = new accountsList();

var activity = new activityList();

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

// Determine if url references an invite or account
let isInviteURL = window.invites.isInviteURL();
let isAccountURL = accounts.isAccountURL();

// load tabs
const tabs: { id: string, url: string, reloader: () => void, unloader?: () => void }[] = [
    {
        id: "invites",
        url: "",
        reloader: () => window.invites.reload(() => {
            if (isInviteURL) {
                window.invites.loadInviteURL();
                // Don't keep loading the same item on every tab refresh
                isInviteURL = false;
            }
        }),
    },
    {
        id: "accounts",
        url: "accounts",
        reloader: () => accounts.reload(() => {
            if (isAccountURL) {
                accounts.loadAccountURL();
                // Don't keep loading the same item on every tab refresh
                isAccountURL = false;
            }
            accounts.bindPageEvents();
        }),
        unloader: accounts.unbindPageEvents
        
    },
    {
        id: "activity",
        url: "activity",
        reloader: () => {
            activity.reload()
            activity.bindPageEvents();
        },
        unloader: activity.unbindPageEvents
    },
    {
        id: "settings",
        url: "settings",
        reloader: settings.reload
    }
];

const defaultTab = tabs[0];

window.tabs = new Tabs();

for (let tab of tabs) {
    window.tabs.addTab(tab.id, window.pages.Base + window.pages.Admin + "/" + tab.url, null, tab.reloader, tab.unloader || null);
}

let matchedTab = false
for (const tab of tabs) {
    if (window.location.pathname.startsWith(window.pages.Base + window.pages.Current + "/" + tab.url)) {
        window.tabs.switch(tab.url, true);
        matchedTab = true;
    }
}
// Default tab
if (!matchedTab) {
    window.tabs.switch("", true);
}

const login = new Login(window.modals.login as Modal, "/", window.loginAppearance);
login.onLogin = () => {
    console.log("Logged in.");
    window.updater = new Updater();
    // FIXME: Decide whether to autoload activity or not
    reloadProfileNames();
    setInterval(() => { window.invites.reload(); accounts.reloadIfNotInScroll(); }, 30*1000);
    // Triggers pre and post funcs, even though we're already on that page
    window.tabs.switch(window.tabs.current);
}

bindManualDropdowns();

login.bindLogout(document.getElementById("logout-button"));

login.login("", "");
