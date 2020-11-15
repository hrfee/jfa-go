import { _get, _post, _delete } from "../modules/common.js";
import { Focus, Unfocus } from "../modules/admin.js";

interface aWindow extends Window {
    checkCheckboxes: () => void;
}

declare var window: aWindow;

export const checkCheckboxes = (): void => {
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

window.checkCheckboxes = checkCheckboxes;

export function populateUsers(): void {
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
        if (window.bsVersion != 5) {
            fci = "";
        }
        return `
            <td nowrap="nowrap" class="align-middle" scope="row"><input class="${fci}" type="checkbox" value="" id="select_${id}" onclick="checkCheckboxes();"></td>
            <td nowrap="nowrap" class="align-middle">${username}${admin ? '<span style="margin-left: 1rem;" class="badge rounded-pill bg-info text-dark">Admin</span>' : ''}</td>
            <td nowrap="nowrap" class="align-middle">${generateEmail(id, name, email)}</td>
            <td nowrap="nowrap" class="align-middle">${lastActive}</td>
        `;
    };

    _get("/users", null, function (): void {
        if (this.readyState == 4 && this.status == 200) {
            window.jfUsers = this.response['users'];
            for (const user of window.jfUsers) {
                let tr = document.createElement('tr');
                tr.innerHTML = template(user['id'], user['name'], user['email'], user['last_active'], user['admin']);
                accountsList.appendChild(tr);
            }
            Focus(acList.parentNode.querySelector('thead'));
            acList.replaceWith(accountsList);
        }
    });
}

export function populateRadios(): void {
    const radioList = document.getElementById('defaultUserRadios');
    radioList.textContent = '';
    let first = true;
    for (const i in window.jfUsers) {
        const user = window.jfUsers[i];
        const radio = document.createElement('div');
        radio.classList.add('form-check');
        let checked = '';
        if (first) {
            checked = 'checked';
            first = false;
        }
        radio.innerHTML = `
        <input class="form-check-input" type="radio" name="defaultRadios" id="default_${user['id']}" ${checked}>
        <label class="form-check-label" for="default_${user['id']}">${user['name']}</label>`;
        radioList.appendChild(radio);
    }
}

