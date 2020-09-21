var config: Object = {};
var modifiedConfig: Object = {};

function sendConfig(modalID: string, restart?: boolean): void {
    modifiedConfig["restart-program"] = restart;
    _post("/modifyConfig", modifiedConfig, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200 || this.status == 204) {
                createModal(modalID, true).hide();
                if (modalID != "settingsMenu") {
                    settingsModal.hide();
                }
            }
            if (restart) {
                refreshModal.show();
            }
        }
    });
}

(document.getElementById('openDefaultsWizard') as HTMLButtonElement).onclick = function (): void {
    const button = this as HTMLButtonElement;
    button.disabled = true;
    const ogHTML = button.innerHTML;
    button.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
    Loading...`;
    _get("/getUsers", null, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200) {
                jfUsers = this.response['users'];
                populateRadios();
                button.disabled = false;
                button.innerHTML = ogHTML;
                const submitButton = document.getElementById('storeDefaults') as HTMLButtonElement;
                submitButton.disabled = false;
                submitButton.textContent = 'Submit';
                addAttr(submitButton, "btn-primary");
                rmAttr(submitButton, "btn-danger");
                rmAttr(submitButton, "btn-success");
                document.getElementById('defaultsTitle').textContent = `New user defaults`;
                document.getElementById('userDefaultsDescription').textContent = `
                Create an account and configure it to your liking, then choose it from below to store the settings as a template for all new users.`;
                document.getElementById('storeHomescreenLabel').textContent = `Store homescreen layout`;
                (document.getElementById('defaultsSource') as HTMLSelectElement).value = 'fromUser';
                document.getElementById('defaultsSourceSection').classList.add('unfocused');
                (document.getElementById('storeDefaults') as HTMLButtonElement).onclick = (): void => storeDefaults('all');
                Focus(document.getElementById('defaultUserRadios'));
                settingsModal.hide();
                userDefaultsModal.show();
            }
        }
    });
};

(document.getElementById('openAbout') as HTMLButtonElement).onclick = (): void => {
    settingsModal.hide();
    aboutModal.show();
};

(document.getElementById('openSettings') as HTMLButtonElement).onclick = (): void => _get("/getConfig", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        const settingsList = document.getElementById('settingsList');
        settingsList.textContent = '';
        config = this.response;
        for (const i in config["order"]) {
            const section: string = config["order"][i]
            const sectionCollapse = document.createElement('div') as HTMLDivElement;
            addAttr(sectionCollapse, "collapse");
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
            <button type="button" class="list-group-item list-group-item-action" id="${section}_button" data-toggle="collapse" data-target="#${section}">${title}</button>
            `;
            settingsList.appendChild(sectionCollapse);
        }
        settingsModal.show();
    }
});

(document.getElementById('settingsSave') as HTMLButtonElement).onclick = function (): void {
    modifiedConfig = {};
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
            if (val != config[section][entry]["value"]) {
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
    if (restartSettingsChanged) {
        (document.getElementById('applyRestarts') as HTMLButtonElement).onclick = (): void => sendConfig("restartModal");
        const restartButton = document.getElementById('applyAndRestart') as HTMLButtonElement;
        if (restartButton) {
            restartButton.onclick = (): void => sendConfig("restartModal", true);
        }
        settingsModal.hide();
        restartModal.show();
    } else if (settingsChanged) {
        sendConfig("settingsMenu");
    } else {
        settingsModal.hide();
    }
};
