var config: Object = {};
var modifiedConfig: Object = {};

function sendConfig(restart?: boolean): void {
    modifiedConfig["restart-program"] = restart;
    _post("/modifyConfig", modifiedConfig, function (): void {
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
                refreshModal.show();
            }
        }
    });
}

(document.getElementById('openAbout') as HTMLButtonElement).onclick = (): void => {
    aboutModal.show();
};

const openSettings = (settingsList: HTMLElement, settingsContent: HTMLElement, callback?: () => void): void => _get("/getConfig", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        settingsList.textContent = '';
        config = this.response;
        for (const i in config["order"]) {
            const section: string = config["order"][i]
            const sectionCollapse = document.createElement('div') as HTMLDivElement;
            Unfocus(sectionCollapse);
            sectionCollapse.id = section;

            const title: string = config[section]["meta"]["name"];
            const description: string = config[section]["meta"]["description"];
            const entryListID: string = `${section}_entryList`;
            // const footerID: string = `${section}_footer`;

            sectionCollapse.innerHTML = `
            <div class="card card-body">
                <small class="text-muted">${description}</small>
                <div class="${entryListID}">
                </div>
            </div>
            `;

            for (const x in config[section]["order"]) {
                const entry: string = config[section]["order"][x];
                if (entry == "meta") {
                    continue;
                }
                let entryName: string = config[section][entry]["name"];
                let required = false;
                if (config[section][entry]["required"]) {
                    entryName += ` <sup class="text-danger">*</sup>`;
                    required = true;
                }
                if (config[section][entry]["requires_restart"]) {
                    entryName += ` <sup class="text-danger">R</sup>`;
                }
                if ("description" in config[section][entry]) {
                    entryName +=`
                     <a class="text-muted" href="#" data-toggle="tooltip" data-placement="right" title="${config[section][entry]['description']}"><i class="fa fa-question-circle-o"></i></a>
                     `;
                }
                const entryValue: boolean | string = config[section][entry]["value"];
                const entryType: string = config[section][entry]["type"];
                const entryGroup = document.createElement('div');
                if (entryType == "bool") {
                    entryGroup.classList.add("form-check");
                    entryGroup.innerHTML = `
                    <input class="form-check-input" type="checkbox" value="" id="${section}_${entry}" ${(entryValue as boolean) ? 'checked': ''} ${required ? 'required' : ''}>
                    <label class="form-check-label" for="${section}_${entry}">${entryName}</label>
                    `;
                    (entryGroup.querySelector('input[type=checkbox]') as HTMLInputElement).onclick = function (): void {
                        const me = this as HTMLInputElement;
                        for (const y in config["order"]) {
                            const sect: string = config["order"][y];
                            for (const z in config[sect]["order"]) {
                                const ent: string = config[sect]["order"][z];
                                if (`${sect}_${config[sect][ent]['depends_true']}` == me.id) {
                                    (document.getElementById(`${sect}_${ent}`) as HTMLInputElement).disabled = !(me.checked);
                                } else if (`${sect}_${config[sect][ent]['depends_false']}` == me.id) {
                                    (document.getElementById(`${sect}_${ent}`) as HTMLInputElement).disabled = me.checked;
                                }
                            }
                        }
                    };
                } else if ((entryType == 'text') || (entryType == 'email') || (entryType == 'password') || (entryType == 'number')) {
                    entryGroup.classList.add("form-group");
                    entryGroup.innerHTML = `
                    <label for="${section}_${entry}">${entryName}</label>
                    <input type="${entryType}" class="form-control" id="${section}_${entry}" aria-describedby="${entry}" value="${entryValue}" ${required ? 'required' : ''}>
                    `;
                } else if (entryType == 'select') {
                    entryGroup.classList.add("form-group");
                    const entryOptions: Array<string> = config[section][entry]["options"];
                    let innerGroup = `
                    <label for="${section}_${entry}">${entryName}</label>
                    <select class="form-control" id="${section}_${entry}" ${required ? 'required' : ''}>
                    `;
                    for (const z in entryOptions) {
                        const entryOption = entryOptions[z];
                        let selected: boolean = (entryOption == entryValue);
                        innerGroup += `
                        <option value="${entryOption}" ${selected ? 'selected' : ''}>${entryOption}</option>
                        `;
                    }
                    innerGroup += `</select>`;
                    entryGroup.innerHTML = innerGroup;
                }
                sectionCollapse.getElementsByClassName(entryListID)[0].appendChild(entryGroup);
            }
        
            settingsList.innerHTML += `
            <button type="button" class="list-group-item list-group-item-action" id="${section}_button" onclick="showSetting('${section}')">${title}</button>
            `;
            settingsContent.appendChild(sectionCollapse);
        }
        if (callback) {
            callback();
        }
    }
});

interface Profile {
    Admin: boolean;
    LibraryAccess: string;
    FromUser: string;
}

(document.getElementById('profiles_button') as HTMLButtonElement).onclick = (): void => showSetting("profiles", populateProfiles);

const populateProfiles = (noTable?: boolean): void => _get("/getProfiles", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        const profileList = document.getElementById('profileList');
        profileList.textContent = '';
        availableProfiles = [];
        for (let name in this.response) {
            availableProfiles.push(name);
            if (!noTable) {
                const profile: Profile = {
                    Admin: this.response[name]["admin"],
                    LibraryAccess: this.response[name]["libraries"],
                    FromUser: this.response[name]["fromUser"]
                };
                profileList.innerHTML += `
                <td nowrap="nowrap" class="align-middle"><strong>${name}</strong></td>
                <td nowrap="nowrap" class="align-middle">${profile.FromUser}</td>
                <td nowrap="nowrap" class="align-middle">${profile.Admin ? "Yes" : "No"}</td>
                <td nowrap="nowrap" class="align-middle">${profile.LibraryAccess}</td>
                <td nowrap="nowrap" class="align-middle"><button class="btn btn-outline-danger" onclick="deleteProfile('${name}')">Delete</button></td>
                `;
            }
        }
    }
});

const deleteProfile = (name: string): void => _post("/deleteProfile", { "name": name }, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        populateProfiles();
    }
});

const createProfile = (): void => _get("/getUsers", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        jfUsers = this.response["users"];
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
        Focus(document.getElementById('defaultUserRadios'));
        userDefaultsModal.show();
    }
});

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
    _post("/createProfile", data, function (): void {
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
                    populateProfiles();
                    userDefaultsModal.hide();

                }, 1000);
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

function showSetting(id: string, runBefore?: () => void): void {
    const els = document.getElementById('settingsLeft').querySelectorAll("button[type=button]:not(.static)") as NodeListOf<HTMLButtonElement>;
    for (let i = 0; i < els.length; i++) {
        const el = els[i];
        if (el.id != `${id}_button`) {
            rmAttr(el, "active");
        }
        const sectEl = document.getElementById(el.id.replace("_button", ""));
        if (sectEl.id != id) {
            Unfocus(sectEl);
        }
    }
    addAttr(document.getElementById(`${id}_button`), "active");
    const section = document.getElementById(id);
    if (runBefore) {
        runBefore();
    }
    Focus(section);
    if (screen.width <= 1100) {
        // ugly
        setTimeout((): void => section.scrollIntoView(<ScrollIntoViewOptions>{ block: "center", behavior: "smooth" }), 200);
    }
}

// (document.getElementById('openSettings') as HTMLButtonElement).onclick = (): void => openSettings(document.getElementById('settingsList'), document.getElementById('settingsList'), (): void => settingsModal.show());

(document.getElementById('settingsSave') as HTMLButtonElement).onclick = function (): void {
    modifiedConfig = {};
    const save = this as HTMLButtonElement;
    let restartSettingsChanged = false;
    let settingsChanged = false;
    for (const i in config["order"]) {
        const section = config["order"][i];
        for (const x in config[section]["order"]) {
            const entry = config[section]["order"][x];
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
            if (val != config[section][entry]["value"].toString()) {
                if (!(section in modifiedConfig)) {
                    modifiedConfig[section] = {};
                }
                modifiedConfig[section][entry] = val;
                settingsChanged = true;
                if (config[section][entry]["requires_restart"]) {
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
        restartModal.show();
    } else if (settingsChanged) {
        save.innerHTML = spinnerHTML;
        sendConfig();
    }
};

(document.getElementById('restartModalCancel') as HTMLButtonElement).onclick = (): void => {
    document.getElementById('settingsSave').textContent = "Save";
};
