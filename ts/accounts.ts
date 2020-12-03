import { checkCheckboxes, populateUsers, populateRadios, changeEmail, validateEmail } from "./modules/accounts.js";
import { _post, _get, _delete, rmAttr, addAttr, createEl } from "./modules/common.js";
import { populateProfiles } from "./modules/settings.js";
import { Focus, Unfocus, storeDefaults } from "./modules/admin.js";

interface aWindow extends Window {
    changeEmail(icon: HTMLElement, id: string): void;
}

declare var window: aWindow;

window.changeEmail = changeEmail;

(<HTMLInputElement>document.getElementById('selectAll')).onclick = function (): void {
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    for (let i = 0; i < checkboxes.length; i++) {
        checkboxes[i].checked = (<HTMLInputElement>this).checked;
    }
    checkCheckboxes();
};

(<HTMLInputElement>document.getElementById('deleteModalNotify')).onclick = function (): void {
    const textbox: HTMLElement = document.getElementById('deleteModalReasonBox');
    if ((<HTMLInputElement>this).checked) {
        Focus(textbox);
    } else {
        Unfocus(textbox);
    }
};

(<HTMLButtonElement>document.getElementById('accountsTabDelete')).onclick = function (): void {
    const deleteButton = this as HTMLButtonElement;
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]:checked');
    let selected: Array<string> = new Array(checkboxes.length);
    for (let i = 0; i < checkboxes.length; i++) {
        selected[i] = checkboxes[i].id.replace("select_", "");
    }
    let title = " user";
    let msg = "Notify user";
    if (selected.length > 1) {
        title += "s";
        msg += "s";
    }
    title = `Delete ${selected.length} ${title}`;
    msg += " of account deletion";

    document.getElementById('deleteModalTitle').textContent = title;
    const dmNotify = document.getElementById('deleteModalNotify') as HTMLInputElement;
    dmNotify.checked = false;
    document.getElementById('deleteModalNotifyLabel').textContent = msg;
    const dmReason = document.getElementById('deleteModalReason') as HTMLTextAreaElement;
    dmReason.value = '';
    Unfocus(document.getElementById('deleteModalReasonBox'));
    const dmSend  = document.getElementById('deleteModalSend') as HTMLButtonElement;
    dmSend.textContent = 'Delete';
    dmSend.onclick = function (): void {
        const button = this as HTMLButtonElement;
        const send = {
            'users': selected,
            'notify': dmNotify.checked,
            'reason': dmReason.value
        };
        _delete("/users", send, function (): void {
            if (this.readyState == 4) {
                if (this.status == 500) {
                    if ("error" in this.reponse) {
                        button.textContent = 'Failed';
                    } else {
                        button.textContent = 'Partial fail (check console)';
                        console.log(this.response);
                    }
                    setTimeout((): void => {
                        Unfocus(deleteButton);
                        window.Modals.delete.hide();
                    }, 4000);
                } else {
                    Unfocus(deleteButton);
                    window.Modals.delete.hide()
                }
                populateUsers();
                checkCheckboxes();
            }
        });
    };
    window.Modals.delete.show();
};

(<HTMLInputElement>document.getElementById('selectAll')).checked = false;

(<HTMLButtonElement>document.getElementById('accountsTabSetDefaults')).onclick = function (): void {
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]:checked');
    let userIDs: Array<string> = new Array(checkboxes.length);
    for (let i = 0; i < checkboxes.length; i++){
        userIDs[i] = checkboxes[i].id.replace("select_", "");
    }
    if (userIDs.length == 0) {
        return;
    }
    populateRadios();
    let userString = 'user';
    if (userIDs.length > 1) {
        userString += "s";
    }
    populateProfiles(true);
    const profileSelect = document.getElementById('profileSelect') as HTMLSelectElement;
    profileSelect.textContent = '';
    for (let i = 0; i < window.availableProfiles.length; i++) {
        profileSelect.innerHTML += `
        <option value="${window.availableProfiles[i]}" ${(i == 0) ? "selected" : ""}>${window.availableProfiles[i]}</option>
        `;
    }
    document.getElementById('defaultsTitle').textContent = `Apply settings to ${userIDs.length} ${userString}`;
    document.getElementById('userDefaultsDescription').textContent = `
    Apply settings from an existing profile or source settings from a user.
    `;
    document.getElementById('storeHomescreenLabel').textContent = `Apply homescreen layout`;
    Focus(document.getElementById('defaultsSourceSection'));
    (<HTMLSelectElement>document.getElementById('defaultsSource')).value = 'profile';
    Focus(document.getElementById('profileSelectBox'));
    Unfocus(document.getElementById('defaultUserRadiosBox'));
    Unfocus(document.getElementById('newProfileBox'));
    document.getElementById('storeDefaults').onclick = (): void => storeDefaults(userIDs);
    window.Modals.userDefaults.show();
};

(<HTMLSelectElement>document.getElementById('defaultsSource')).addEventListener('change', function (): void {
    const radios = document.getElementById('defaultUserRadiosBox');
    const profileBox = document.getElementById('profileSelectBox');
    if (this.value == 'profile') {
        Unfocus(radios);
        Focus(profileBox);
    } else {
        Unfocus(profileBox);
        Focus(radios);
    }
});

(<HTMLButtonElement>document.getElementById('newUserCreate')).onclick = function (): void {
    const button = this as HTMLButtonElement;
    const ogText = button.textContent;
    button.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Creating...
    `;
    const email: string = (<HTMLInputElement>document.getElementById('newUserEmail')).value;
    var username: string = email;
    if (document.getElementById('newUserName') != null) {
        username = (<HTMLInputElement>document.getElementById('newUserName')).value;
    }
    const password: string = (<HTMLInputElement>document.getElementById('newUserPassword')).value;
    if (!validateEmail(email) && email != "") {
        return;
    }
    const send = {
        'username': username,
        'password': password,
        'email': email
    };
    _post("/users", send, function (): void {
        if (this.readyState == 4) {
            rmAttr(button, 'btn-primary');
            if (this.status == 200) {
                addAttr(button, 'btn-success');
                button.textContent = 'Success';
                setTimeout((): void => {
                    rmAttr(button, 'btn-success');
                    addAttr(button, 'btn-primary');
                    button.textContent = ogText;
                    window.Modals.newUser.hide();
                }, 1000);
                populateUsers();
            } else {
                addAttr(button, 'btn-danger');
                if ("error" in this.response) {
                    button.textContent = this.response["error"];
                } else {
                    button.textContent = 'Failed';
                }
                setTimeout((): void => {
                    rmAttr(button, 'btn-danger');
                    addAttr(button, 'btn-primary');
                    button.textContent = ogText;
                }, 2000);
                populateUsers();
            }
        }
    });
};

(<HTMLButtonElement>document.getElementById('accountsTabAddUser')).onclick = function (): void {
    (<HTMLInputElement>document.getElementById('newUserEmail')).value = '';
    (<HTMLInputElement>document.getElementById('newUserPassword')).value = '';
    if (document.getElementById('newUserName') != null) {
        (<HTMLInputElement>document.getElementById('newUserName')).value = '';
    }
    window.Modals.newUser.show();
};
