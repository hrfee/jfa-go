import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { _get, _post, notificationBox, whichAnimationEvent } from "./modules/common.js";
import { Login } from "./modules/login.js";

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

const login = new Login(window.modals.login as Modal, "/my/");
login.onLogin = () => {
    console.log("Logged in.");
    document.getElementById("card-user").textContent = "Logged In!";
    _get("/my/hello", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            const card = document.getElementById("card-user");
            card.textContent = card.textContent + " got response " + req.response["response"];
        }
    });
};

login.login("", "");
