import { _post, addLoader, notificationBox, removeLoader, whichAnimationEvent } from "./modules/common.js";
import { loadLangSelector } from "./modules/lang.js";

declare var window: GlobalWindow;

interface StoreCheckoutResponse {
    response?: string;
    error?: string;
}

const emailInput = document.getElementById("email-input") as HTMLInputElement;
const emailError = document.getElementById("email-error") as HTMLElement;
const checkoutButtons = Array.from(document.querySelectorAll(".store-checkout")) as HTMLButtonElement[];

window.animationEvent = whichAnimationEvent();
window.notifications = new notificationBox(document.getElementById("notification-box") as HTMLDivElement);

loadLangSelector("form");

const setButtonsDisabled = (disabled: boolean) => {
    for (const button of checkoutButtons) {
        button.disabled = disabled;
    }
};

const validateEmail = (): string => {
    const email = emailInput.value.trim();
    if (emailInput.checkValidity()) {
        emailError.classList.add("hidden");
        return email;
    }
    emailError.classList.remove("hidden");
    emailInput.focus();
    emailInput.scrollIntoView({ behavior: "smooth", block: "center" });
    return "";
};

const checkout = (button: HTMLButtonElement) => {
    const email = validateEmail();
    const plan = button.dataset.plan || "";
    if (!email || !plan) {
        return;
    }

    addLoader(button);
    setButtonsDisabled(true);
    _post("/stripe/create-checkout", { email, plan }, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        removeLoader(button);
        setButtonsDisabled(false);

        const data = req.response as StoreCheckoutResponse;
        if (req.status != 200) {
            window.notifications.customError("storeCheckout", data?.error || "Failed to start checkout.");
            return;
        }
        if (!data?.response) {
            window.notifications.customError("storeCheckout", "Stripe did not return a checkout URL.");
            return;
        }
        window.location.href = data.response;
    }, true);
};

for (const button of checkoutButtons) {
    button.onclick = () => checkout(button);
}
