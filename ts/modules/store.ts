import { _get, _post, addLoader, removeLoader } from "./common.js";

interface ConfigSetting {
    setting: string;
    value: any;
}

interface ConfigSection {
    section: string;
    settings: ConfigSetting[];
}

interface ConfigResponse {
    sections: ConfigSection[];
}

export class Store {
    private _el: HTMLElement;
    private _currencySelect: HTMLSelectElement;
    private _monthlyInput: HTMLInputElement;
    private _saveButton: HTMLButtonElement;

    private _paypalEnabled: HTMLInputElement;
    private _paypalMode: HTMLSelectElement;
    private _paypalClientId: HTMLInputElement;
    private _paypalSecret: HTMLInputElement;
    private _paypalPlanId: HTMLInputElement;



    constructor() {
        this._el = document.getElementById("tab-store");
        this._currencySelect = document.getElementById("store-currency") as HTMLSelectElement;
        this._monthlyInput = document.getElementById("store-price-monthly") as HTMLInputElement;

        this._paypalEnabled = document.getElementById("paypal-enabled") as HTMLInputElement;
        this._paypalMode = document.getElementById("paypal-mode") as HTMLSelectElement;
        this._paypalClientId = document.getElementById("paypal-client-id") as HTMLInputElement;
        this._paypalSecret = document.getElementById("paypal-secret") as HTMLInputElement;
        this._paypalPlanId = document.getElementById("paypal-plan-id") as HTMLInputElement;

        this._saveButton = document.getElementById("store-save") as HTMLButtonElement;

        this._saveButton.onclick = this.save;
    }

    load = () => {
        _get("/config", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                console.error("Failed to load config");
                return;
            }
            const data = req.response as ConfigResponse;
            const stripeSection = data.sections.find((s) => s.section === "stripe");
            const paypalSection = data.sections.find((s) => s.section === "paypal");

            if (stripeSection) {
                const currency = stripeSection.settings.find((s) => s.setting === "price_currency");
                const monthly = stripeSection.settings.find((s) => s.setting === "price_monthly");

                if (currency) this._currencySelect.value = currency.value;
                if (monthly) this._monthlyInput.value = (parseInt(monthly.value) / 100).toFixed(2);
            }

            if (paypalSection) {
                const enabled = paypalSection.settings.find((s) => s.setting === "enabled");
                const mode = paypalSection.settings.find((s) => s.setting === "mode");
                const clientId = paypalSection.settings.find((s) => s.setting === "client_id");
                const secret = paypalSection.settings.find((s) => s.setting === "client_secret");
                const planId = paypalSection.settings.find((s) => s.setting === "plan_id_monthly");

                if (enabled) this._paypalEnabled.checked = enabled.value;
                if (mode) this._paypalMode.value = mode.value;
                if (clientId) this._paypalClientId.value = clientId.value;
                if (secret) this._paypalSecret.value = secret.value;
                if (planId) this._paypalPlanId.value = planId.value;
            }
        });
    };

    save = () => {
        const payload = {
            stripe: {
                price_currency: this._currencySelect.value,
                price_monthly: Math.round(parseFloat(this._monthlyInput.value) * 100).toString(),
            },
            paypal: {
                enabled: this._paypalEnabled.checked,
                mode: this._paypalMode.value,
                client_id: this._paypalClientId.value,
                client_secret: this._paypalSecret.value,
                plan_id_monthly: this._paypalPlanId.value,
            }
        };

        addLoader(this._saveButton);
        _post("/config", payload, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._saveButton);
            if (req.status == 200 || req.status == 204) {
                (window as any).notifications.customSuccess("settingsSaved", "Settings saved successfully");
            } else {
                (window as any).notifications.customError("settingsSaved", "Failed to save settings");
            }
        });
    };
}
