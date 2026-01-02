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

    constructor() {
        this._el = document.getElementById("tab-store");
        this._currencySelect = document.getElementById("store-currency") as HTMLSelectElement;
        this._monthlyInput = document.getElementById("store-price-monthly") as HTMLInputElement;
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
            if (!stripeSection) return;

            const currency = stripeSection.settings.find((s) => s.setting === "price_currency");
            const monthly = stripeSection.settings.find((s) => s.setting === "price_monthly");

            if (currency) this._currencySelect.value = currency.value;
            if (monthly) this._monthlyInput.value = (parseInt(monthly.value) / 100).toFixed(2);
        });
    };

    save = () => {
        const payload = {
            stripe: {
                price_currency: this._currencySelect.value,
                price_monthly: Math.round(parseFloat(this._monthlyInput.value) * 100).toString(),
            },
        };

        addLoader(this._saveButton);
        _post("/config", payload, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._saveButton);
            if (req.status == 200 || req.status == 204) {
                // Success feedback handled by notification box in common.ts if used, but here custom
                // Assuming windows.notifications is available globally as seen in admin.ts
                (window as any).notifications.customSuccess("settingsSaved", "Settings saved successfully");
            } else {
                (window as any).notifications.customError("settingsSaved", "Failed to save settings");
            }
        });
    };
}
