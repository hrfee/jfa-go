import { toClipboard, notificationBox } from "./modules/common.js";

const pin = document.getElementById("pin") as HTMLSpanElement;

if (pin) {
    // Load this individual string into the DOM, so we don't have to load the whole language file.
    const copy = document.getElementById("copy-notification");
    const copyString = copy.textContent;
    copy.remove();

    window.notifications = new notificationBox(document.getElementById("notification-box") as HTMLDivElement, 5);

    pin.onclick = () => {
        toClipboard(pin.textContent);
        window.notifications.customPositive("copied", "", copyString);
        pin.classList.add("~positive");
        pin.classList.remove("~urge");
        setTimeout(() => {
            pin.classList.add("~urge");
            pin.classList.remove("~positive");
        }, 5000);
    };
}
