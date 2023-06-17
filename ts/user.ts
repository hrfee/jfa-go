import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { _get, _post, notificationBox, whichAnimationEvent } from "./modules/common.js";
import { Login } from "./modules/login.js";

interface userWindow extends Window {
    jellyfinID: string;
    username: string;
}

declare var window: userWindow;

const theme = new ThemeManager(document.getElementById("button-theme"));

window.lang = new lang(window.langFile as LangFile);

loadLangSelector("user");

window.animationEvent = whichAnimationEvent();

window.token = "";

window.modals = {} as Modals;

(() => {
    window.modals.login = new Modal(document.getElementById("modal-login"), true);
})();

window.notifications = new notificationBox(document.getElementById('notification-box') as HTMLDivElement, 5);

var rootCard = document.getElementById("card-user");

const login = new Login(window.modals.login as Modal, "/my/");
login.onLogin = () => {
    console.log("Logged in.");
    _get("/my/details", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                window.notifications.customError("myDetailsError", req.response["error"]);
                return;
            }
            window.jellyfinID = req.response["id"];
            window.username = req.response["username"];
            rootCard.querySelector(".heading").textContent = window.lang.strings("welcomeUser").replace("{user}", window.username);
        }
    });
};

login.login("", "");
