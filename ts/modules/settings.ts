import { _get, _post, _delete, rmAttr, addAttr } from "../modules/common.js";
import { Focus, Unfocus } from "../modules/admin.js";

interface Profile {
    Admin: boolean;
    LibraryAccess: string;
    FromUser: string;
}

export const populateProfiles = (noTable?: boolean): void => _get("/profiles", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        const profileList = document.getElementById('profileList');
        profileList.textContent = '';
        window.availableProfiles = [this.response["default_profile"]];
        for (let name in this.response["profiles"]) {
            if (name != window.availableProfiles[0]) {
                window.availableProfiles.push(name);
            }
            const reqProfile = this.response["profiles"][name];
            if (!noTable && name != "default_profile") {
                const profile: Profile = {
                    Admin: reqProfile["admin"],
                    LibraryAccess: reqProfile["libraries"],
                    FromUser: reqProfile["fromUser"]
                };
                profileList.innerHTML += `
                <td nowrap="nowrap" class="align-middle"><strong>${name}</strong></td>
                <td nowrap="nowrap" class="align-middle"><input class="${window.bs5 ? "form-check-input" : ""}" type="radio" name="defaultProfile" onclick="setDefaultProfile('${name}')" ${(name == window.availableProfiles[0]) ? "checked" : ""}></td>
                <td nowrap="nowrap" class="align-middle">${profile.FromUser}</td>
                <td nowrap="nowrap" class="align-middle">${profile.Admin ? "Yes" : "No"}</td>
                <td nowrap="nowrap" class="align-middle">${profile.LibraryAccess}</td>
                <td nowrap="nowrap" class="align-middle"><button class="btn btn-outline-danger" id="defaultProfile_${name}" onclick="deleteProfile('${name}')">Delete</button></td>
                `;
            }
        }
    }
});

export const openSettings = (settingsList: HTMLElement, settingsContent: HTMLElement, callback?: () => void): void => _get("/config", null, function (): void {
    if (this.readyState == 4 && this.status == 200) {
        settingsList.textContent = '';
        window.config = this.response;
        for (const i in window.config["order"]) {
            const section: string = window.config["order"][i]
            const sectionCollapse = document.createElement('div') as HTMLDivElement;
            Unfocus(sectionCollapse);
            sectionCollapse.id = section;

            const title: string = window.config[section]["meta"]["name"];
            const description: string = window.config[section]["meta"]["description"];
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
                let entryName: string = window.config[section][entry]["name"];
                let required = false;
                if (window.config[section][entry]["required"]) {
                    entryName += ` <sup class="text-danger">*</sup>`;
                    required = true;
                }
                if (window.config[section][entry]["requires_restart"]) {
                    entryName += ` <sup class="text-danger">R</sup>`;
                }
                if ("description" in window.config[section][entry]) {
                    entryName +=`
                     <a class="text-muted" href="#" data-toggle="tooltip" data-placement="right" title="${window.config[section][entry]['description']}"><i class="fa fa-question-circle-o"></i></a>
                     `;
                }
                const entryValue: boolean | string = window.config[section][entry]["value"];
                const entryType: string = window.config[section][entry]["type"];
                const entryGroup = document.createElement('div');
                if (entryType == "bool") {
                    entryGroup.classList.add("form-check");
                    entryGroup.innerHTML = `
                    <input class="form-check-input" type="checkbox" value="" id="${section}_${entry}" ${(entryValue as boolean) ? 'checked': ''} ${required ? 'required' : ''}>
                    <label class="form-check-label" for="${section}_${entry}">${entryName}</label>
                    `;
                    (entryGroup.querySelector('input[type=checkbox]') as HTMLInputElement).onclick = function (): void {
                        const me = this as HTMLInputElement;
                        for (const y in window.config["order"]) {
                            const sect: string = window.config["order"][y];
                            for (const z in window.config[sect]["order"]) {
                                const ent: string = window.config[sect]["order"][z];
                                if (`${sect}_${window.config[sect][ent]['depends_true']}` == me.id) {
                                    (document.getElementById(`${sect}_${ent}`) as HTMLInputElement).disabled = !(me.checked);
                                } else if (`${sect}_${window.config[sect][ent]['depends_false']}` == me.id) {
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
                    const entryOptions: Array<string> = window.config[section][entry]["options"];
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

export function showSetting(id: string, runBefore?: () => void): void {
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

