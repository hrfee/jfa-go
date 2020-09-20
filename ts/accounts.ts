const _post = (url: string, data: Object, onreadystatechange: () => void): void => {
    let req = new XMLHttpRequest();
    req.open("POST", url, true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
};

const _get = (url: string, data: Object, onreadystatechange: () => void): void => {
    let req = new XMLHttpRequest();
    req.open("GET", url, true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
};

const rmAttr = (el: HTMLElement, attr: string): void => {
    if (el.classList.contains(attr)) {
        el.classList.remove(attr);
    }
};
const addAttr = (el: HTMLElement, attr: string): void => el.classList.add(attr);

const Focus = (el: HTMLElement): void => rmAttr(el, 'unfocused');
const Unfocus = (el: HTMLElement): void => addAttr(el, 'unfocused');

const checkCheckboxes = (): void => {
    const defaultsButton = document.getElementById('accountsTabSetDefaults');
    const deleteButton = document.getElementById('accountsTabDelete');
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]:checked');
    let checked = checkboxes.length;
    if (checked == 0) {
        Unfocus(defaultsButton);
        Unfocus(deleteButton);
    } else {
        Focus(defaultsButton);
        Focus(deleteButton);
        if (checked == 1) {
            deleteButton.textContent = 'Delete User';
        } else {
            deleteButton.textContent = 'Delete Users';
        }
    }
}

const validateEmail = (email: string): boolean => {
    const re = /\S+@\S+\.\S+/;
    return re.test(email);
}

const changeEmail = (icon: HTMLElement, id: string): void => {
    const iconContent = icon.outerHTML;
    icon.setAttribute('class', '');
    const entry: HTMLInputElement = icon.nextElementSibling;
    const ogEmail = entry.value;
    entry.readOnly = false;
    entry.classList.remove('form-control-plaintext');
    entry.classList.add('form-control');
    if (ogEmail == "") {
        entry.placeholder = 'Address';
    }
    const tick = document.createElement('i');
    tick.outerHTML = `
    <i class="fa fa-check d-inline-block icon-button text-success" style="margin-left: 0.5rem; margin-right: 0.5rem;"></i>
    `;
    tick.onclick = (): void => {
        const newEmail = entry.value;
        if (!validateEmail(newEmail) || newEmail == ogEmail) {
            return;
        }
        cross.remove();
        tick.outerHTML = `
        <div class="spinner-border spinner-border-sm" role="status" style="width: 1rem; height: 1rem; margin-left: 0.5rem;">
            <span class="sr-only">Saving...</span>
        </div>
        `;
        let send = {};
        send[id] = newEmail;
        _post("/modifyEmails", send, function (): void {
            if (this.readyState == 4) {
                if (this.status == '200' || this.status == '204') {
                    entry.nextElementSibling.remove();
                } else {
                    entry.value = ogEmail;
                }
            }
        });
        icon.outerHTML = iconContent;
        entry.readOnly = true;
        entry.classList.remove('form-control');
        entry.classList.add('form-control-plaintext');
        entry.placeholder = '';
    };
    const cross: HTMLElement = document.createElement('i');
    cross.outerHTML = `
    <i class="fa fa-close d-inline-block icon-button text-danger"></i>
    `;
    cross.onclick = (): void => {
        tick.remove();
        cross.remove();
        icon.outerHTML = iconContent;
        entry.readOnly = true;
        entry.classList.remove('form-control');
        entry.classList.add('form-control-plaintext');
        entry.placeholder = '';
        entry.value = ogEmail;
    };
    icon.parentNode.appendChild(tick);
    icon.parentNode.appendChild(cross);
};

var jfUsers: Array<Object>;

const populateUsers = (): void => {
    const acList = document.getElementById('accountsList');
    acList.innerHTML = `
    <div class="d-flex align-items-center">
        <strong>Getting Users...</strong>
        <div class="spinner-border ml-auto" role="status" aria-hidden="true"></div>
    </div>
    `;
    Unfocus(acList.parentNode.querySelector('thead'));
    const accountsList = document.createElement('tbody');
    accountsList.id = 'accountsList';
    const generateEmail = (id: string, name: string, email: string): string => {
        let entry: HTMLDivElement = document.createElement('div');
        entry.id = 'email_' + id;
        let emailValue: string = email;
        if (emailValue == undefined) {
            emailValue = "";
        }
        entry.innerHTML = `
        <i class="fa fa-edit d-inline-block icon-button" style="margin-right: 2%;" onclick="changeEmail(this, '${id}')"></i>
        <input type="email" class="form-control-plaintext form-control-sm text-muted d-inline-block addressText" id="address_${id}" style="width: auto;" value="${emailValue}" readonly>
        `;
        return entry.outerHTML;
    };
    const template = (id: string, username: string, email: string, lastActive: string, admin: boolean): string => {
        let isAdmin = "No";
        if (admin) {
            isAdmin = "Yes";
        }
        let fci = "form-check-input";
        if (bsVersion != 5) {
            fci = "";
        }
        return `
            <td nowrap="nowrap" class="align-middle" scope="row"><input class="${fci}" type="checkbox" value="" id="select_${id}" onclick="checkCheckboxes();"></td>
            <td nowrap="nowrap" class="align-middle">${username}</td>
            <td nowrap="nowrap" class="align-middle">${generateEmail(id, name, email)}</td>
            <td nowrap="nowrap" class="align-middle">${lastActive}</td>
            <td nowrap="nowrap" class="align-middle">${isAdmin}</td>
        `;
    };

    _get("/getUsers", null, function (): void {
        if (this.readyState == 4 && this.status == '200') {
            jfUsers = this.response['users'];
            for (const user of jfUsers) {
                let tr = document.createElement('tr');
                tr.innerHTML = template(user['id'], user['name'], user['email'], user['last_active'], user['admin']);
                accountsList.appendChild(tr);
            }
            Focus(acList.parentNode.querySelector('thead'));
            acList.replaceWith(accountsList);
        }
    });
}

(<HTMLInputElement>document.getElementById('selectAll')).onclick = function (): void {
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    for (const i in checkboxes) {
        checkboxes[i].checked = (<HTMLInputElement>this).checked;
    }
};

(<HTMLInputElement>document.getElementById('deleteModalNotify')).onclick = function (): void {
    const textbox: HTMLElement = document.getElementById('deleteModalReasonBox');
    if ((<HTMLInputElement>this).checked) {
        Focus(textbox);
    } else {
        Unfocus(textbox);
    }
};

(<HTMLButtonElement>document.getElementById('accountsTabDelete')).onclick =function (): void {
    const deleteButton: HTMLButtonElement = this;
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]:checked');
    let selected: Array<string> = new Array(checkboxes.length);
    for (const i in checkboxes){ 
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
    const dmNotify: HTMLInputElement = document.getElementById('deleteModalNotify')
    dmNotify.checked = false;
    document.getElementById('deleteModalNotifyLabel').textContent = msg;
    const dmReason: HTMLTextAreaElement = document.getElementById('deleteModalReason')
    dmReason.value = '';
    Unfocus(document.getElementById('deleteModalReasonBox'));
    const dmSend: HTMLButtonElement = document.getElementById('deleteModalSend');
    dmSend.textContent = 'Delete';
    dmSend.onclick = function (): void {
        const button: HTMLButtonElement = this;
        const send = {
            'users': selected,
            'notify': dmNotify.checked,
            'reason': dmReason.value
        };
        _post("/deleteUser", send, function (): void {
            if (this.readyState == 4) {
                if (this.status == '500') {
                    if ("error" in req.reponse) {
                        button.textContent = 'Failed';
                    } else {
                        button.textContent = 'Partial fail (check console)';
                        console.log(req.response);
                    }
                    setTimeout((): void => {
                        Unfocus(deleteButton);
                        deleteModal.hide();
                    }, 4000);
                } else {
                    Unfocus(deleteButton);
                    deleteModal.hide()
                }
                populateUsers();
                checkCheckboxes();
            }
        });
    };
    deleteModal.show();
};

(<HTMLInputElement>document.getElementById('selectAll')).checked = false;

(<HTMLButtonElement>document.getElementById('accountsTabSetDefaults')).onclick = function (): void {
    const checkboxes: NodeListOf<HTMLInputElement> = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]:checked');
    let userIDs: Array<string> = new Array(checkboxes.length);
    for (const i in checkboxes){ 
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
    document.getElementById('defaultsTitle').textContent = `Apply settings to ${userIDs.length} ${userString}`;
    document.getElementById('userDefaultsDescription').textContent = `
    Create an account and configure it to your liking, then choose it from below to apply to your selected users.
    `;
    document.getElementById('storeHomescreenLabel').textContent = `Apply homescreen layout`;
    Focus(document.getElementById('defaultsSourceSection'));
    (<HTMLSelectElement>document.getElementById('defaultsSource')).value = 'userTemplate';
    Unfocus(document.getElementById('defaultUserRadios'));
    document.getElementById('storeDefaults').onclick = (): void => storeDefaults(userIDs);
    userDefaultsModal.show();
};

(<HTMLSelectElement>document.getElementById('defaultsSource')).addEventListener('change', function (): void {
    const radios = document.getElementById('defaultUserRadios');
    if (this.value == 'userTemplate') {
        Unfocus(radios);
    } else {
        Focus(radios);
    }
});

(<HTMLButtonElement>document.getElementById('newUserCreate')).onclick = function (): void {
    const ogText = this.textContent;
    this.innerHTML = `
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
    const button: HTMLButtonElement = this;
    _post("/newUserAdmin", send, function (): void {
        if (this.readyState == 4) {
            rmAttr(button, 'btn-primary');
            if (this.status == '200') {
                addAttr(button, 'btn-success');
                button.textContent = 'Success';
                setTimeout((): void => {
                    rmAttr(button, 'btn-success');
                    addAttr('btn-primary');
                    button.textContent = ogText;
                    newUserModal.hide();
                }, 1000);
            } else {
                addAttr(button, 'btn-danger');
                if ("error" in req.response) {
                    button.textContent = req.response["error"];
                } else {
                    button.textContent = 'Failed';
                }
                setTimeout((): void => {
                    rmAttr(button, 'btn-danger');
                    addAttr(button, 'btn-primary');
                    button.textContent = ogText;
                }, 2000);
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
    newUserModal.show();
};
