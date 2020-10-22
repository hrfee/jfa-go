import { _post, _get, _delete, rmAttr, addAttr } from "./modules/common.js";
import { generateInvites } from "./modules/invites.js";
import { populateRadios } from "./modules/accounts.js";
import { Focus, Unfocus } from "./modules/admin.js";
import { showSetting, populateProfiles } from "./modules/settings.js";

interface aWindow extends Window {
    setDefaultProfile(name: string): void;
    deleteProfile(name: string): void;
    createProfile(): void;
    showSetting(id: string, runBefore?: () => void): void;
    config: Object;
    modifiedConfig: Object;
}

declare var window: aWindow;

window.config = {};
window.modifiedConfig = {};

window.showSetting = showSetting;

function sendConfig(restart?: boolean): void {
    window.modifiedConfig["restart-program"] = restart;
    _post("/config", window.modifiedConfig, function (): void {
        if (this.readyState == 4) {
            const save = document.getElementById("settingsSave") as HTMLButtonElement
            if (this.status == 200 || this.status == 204) {
                save.textContent = "Success";
                addAttr(save, "btn-success");
                rmAttr(save, "btn-primary");
                setTimeout((): void => {
                    save.textContent = "Save";
                    addAttr(save, "btn-primary");
                    rmAttr(save, "btn-success");
                }, 1000);
            } else {
                save.textContent = "Save";
            }
            if (restart) {
                window.Modals.refresh.show();
            }
        }
    });
}

(document.getElementById('openAbout') as HTMLButtonElement).onclick = (): void => {
    window.Modals.about.show();
};

(document.getElementById('profiles_button') as HTMLButtonElement).onclick = (): void => showSetting("profiles", populateProfiles);

window.setDefaultProfile = (name: string): void => _post("/profiles/default", { "name": name }, function (): void {
    if (this.readyState == 4) {
        if (this.status != 200) {
            (document.getElementById(`defaultProfile_${window.availableProfiles[0]}`) as HTMLInputElement).checked = true;
            (document.getElementById(`defaultProfile_${name}`) as HTMLInputElement).checked = false;
        } else {
            generateInvites();
        }
    }
});

window.deleteProfile = (name: string): void => _delete("/profiles", { "name": name }, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        populateProfiles();
    }
});

const createProfile = (): void => _get("/users", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        window.jfUsers = this.response["users"];
        populateRadios();
        const submitButton = document.getElementById('storeDefaults') as HTMLButtonElement;
        submitButton.disabled = false;
        submitButton.textContent = 'Create';
        addAttr(submitButton, "btn-primary");
        rmAttr(submitButton, "btn-danger");
        rmAttr(submitButton, "btn-success");
        document.getElementById('defaultsTitle').textContent = `Create Profile`;
        document.getElementById('userDefaultsDescription').textContent = `
        Create an account and configure it to your liking, then choose it from below to store the settings as a profile. Profiles can be specified per invite, so that any new user on that invite will have the settings applied.`;
        document.getElementById('storeHomescreenLabel').textContent = `Store homescreen layout`;
        (document.getElementById('defaultsSource') as HTMLSelectElement).value = 'fromUser';
        document.getElementById('defaultsSourceSection').classList.add('unfocused');
        (document.getElementById('storeDefaults') as HTMLButtonElement).onclick = storeProfile;
        Focus(document.getElementById('newProfileBox'));
        (document.getElementById('newProfileName') as HTMLInputElement).value = '';
        Focus(document.getElementById('defaultUserRadiosBox'));
        window.Modals.userDefaults.show();
    }
});

window.createProfile = createProfile;

function storeProfile(): void {
    this.disabled = true;
    this.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    const button = document.getElementById('storeDefaults') as HTMLButtonElement;
    const radio = document.querySelector('input[name=defaultRadios]:checked') as HTMLInputElement
    const name = (document.getElementById('newProfileName') as HTMLInputElement).value;
    let id = radio.id.replace("default_", "");
    let data = {
        "name": name,
        "id": id,
        "homescreen": false
    }
    if ((document.getElementById('storeDefaultHomescreen') as HTMLInputElement).checked) {
        data["homescreen"] = true;
    }
    _post("/profiles", data, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200 || this.status == 204) {
                button.textContent = "Success";
                addAttr(button, "btn-success");
                rmAttr(button, "btn-danger");
                rmAttr(button, "btn-primary");
                button.disabled = false;
                setTimeout((): void => {
                    button.textContent = "Create";
                    addAttr(button, "btn-primary");
                    rmAttr(button, "btn-success");
                    button.disabled = false;
                    window.Modals.userDefaults.hide();

                }, 1000);
                populateProfiles();
                generateInvites();
            } else {
                if ("error" in this.response) {
                    button.textContent = this.response["error"];
                } else if (("policy" in this.response) || ("homescreen" in this.response)) {
                    button.textContent = "Failed (check console)";
                } else {
                    button.textContent = "Failed";
                }
                addAttr(button, "btn-danger");
                rmAttr(button, "btn-primary");
                setTimeout((): void => {
                    button.textContent = "Create";
                    addAttr(button, "btn-primary");
                    rmAttr(button, "btn-danger");
                    button.disabled = false;
                }, 1000);
            }
        }
    });
}

// (document.getElementById('openSettings') as HTMLButtonElement).onclick = (): void => openSettings(document.getElementById('settingsList'), document.getElementById('settingsList'), (): void => settingsModal.show());

(document.getElementById('settingsSave') as HTMLButtonElement).onclick = function (): void {
    window.modifiedConfig = {};
    const save = this as HTMLButtonElement;
    let restartSettingsChanged = false;
    let settingsChanged = false;
    for (const i in window.config["order"]) {
        const section = window.config["order"][i];
        for (const x in window.config[section]["order"]) {
            const entry = window.config[section]["order"][x];
            if (entry == "meta") {
                continue;
            }
            let val: string;
            const entryID = `${section}_${entry}`;
            const el = document.getElementById(entryID) as HTMLInputElement;
            if (el.type == "checkbox") {
                val = el.checked.toString();
            } else {
                val = el.value.toString();
            }
            if (val != window.config[section][entry]["value"].toString()) {
                if (!(section in window.modifiedConfig)) {
                    window.modifiedConfig[section] = {};
                }
                window.modifiedConfig[section][entry] = val;
                settingsChanged = true;
                if (window.config[section][entry]["requires_restart"]) {
                    restartSettingsChanged = true;
                }
            }
        }
    }
    const spinnerHTML = ` 
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
    Loading...`;
    if (restartSettingsChanged) {
        save.innerHTML = spinnerHTML;
        (document.getElementById('applyRestarts') as HTMLButtonElement).onclick = (): void => sendConfig();
        const restartButton = document.getElementById('applyAndRestart') as HTMLButtonElement;
        if (restartButton) {
            restartButton.onclick = (): void => sendConfig(true);
        }
        window.Modals.restart.show();
    } else if (settingsChanged) {
        save.innerHTML = spinnerHTML;
        sendConfig();
    }
};

(document.getElementById('restartModalCancel') as HTMLButtonElement).onclick = (): void => {
    document.getElementById('settingsSave').textContent = "Save";
};
