document.getElementById('selectAll').onclick = function() {
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    for (check of checkboxes) {
        check.checked = this.checked;
    }
    checkCheckboxes();
};

function checkCheckboxes() {
    const defaultsButton = document.getElementById('accountsTabSetDefaults');
    const deleteButton = document.getElementById('accountsTabDelete');
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    let checked = 0;
    for (check of checkboxes) {
        if (check.checked) {
            checked++;
        }
    }
    if (checked == 0) {
        defaultsButton.classList.add('unfocused');
        deleteButton.classList.add('unfocused');
    } else {
        if (defaultsButton.classList.contains('unfocused')) {
            defaultsButton.classList.remove('unfocused');
        }
        if (deleteButton.classList.contains('unfocused')) {
            deleteButton.classList.remove('unfocused');
        }
        if (checked == 1) {
            deleteButton.textContent = 'Delete User';
        } else {
            deleteButton.textContent = 'Delete Users';
        }
    }
}

document.getElementById('deleteModalNotify').onclick = function() {
    const textbox = document.getElementById('deleteModalReasonBox');
    if (this.checked && textbox.classList.contains('unfocused')) {
        textbox.classList.remove('unfocused');
    } else if (!this.checked) {
        textbox.classList.add('unfocused');
    }
};

document.getElementById('accountsTabDelete').onclick = function() {
    const deleteButton = this;
    let selected = [];
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    for (check of checkboxes) {
        if (check.checked) {
            selected.push(check.id.replace('select_', ''));
        }
    }
    let title = " user";
    let msg = "Notify user";
    if (selected.length > 1) {
        title += "s";
        msg += "s";
    }
    title = "Delete " + selected.length + title;
    msg += " of account deletion";
    document.getElementById('deleteModalTitle').textContent = title;
    document.getElementById('deleteModalNotify').checked = false;
    document.getElementById('deleteModalNotifyLabel').textContent = msg;
    document.getElementById('deleteModalReason').value = '';
    document.getElementById('deleteModalReasonBox').classList.add('unfocused');
    document.getElementById('deleteModalSend').textContent = 'Delete';
    
    document.getElementById('deleteModalSend').onclick = function() {
        const button = this;
        const send = {
            'users': selected,
            'notify': document.getElementById('deleteModalNotify').checked,
            'reason': document.getElementById('deleteModalReason').value
        };
        let req = new XMLHttpRequest();
        req.open("POST", "/deleteUser", true);
        req.responseType = 'json';
        req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
        req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
        req.onreadystatechange = function() {
            if (this.readyState == 4) {
                if (this.status == 500) {
                    if ("error" in req.response) {
                        button.textContent = 'Failed';
                    } else {
                        button.textContent = 'Partial fail (check console)';
                        console.log(req.response);
                    }
                    setTimeout(function() {
                        deleteModal.hide();
                        deleteButton.classList.add('unfocused');
                    }, 4000);
                } else {
                    deleteButton.classList.add('unfocused');
                    deleteModal.hide();
                }
                populateUsers();
                checkCheckboxes();
            }
        };
        req.send(JSON.stringify(send));
    };
    deleteModal.show();
}

var jfUsers = [];

function validEmail(email) {
    const re = /\S+@\S+\.\S+/;
    return re.test(email);
}

function changeEmail(icon, id) {
    const iconContent = icon.outerHTML;
    icon.setAttribute("class", "");
    const entry = icon.nextElementSibling;
    const ogEmail = entry.value;
    entry.readOnly = false;
    entry.classList.remove('form-control-plaintext');
    entry.classList.add('form-control');
    if (entry.value == "") {
        entry.placeholder = 'Address';
    }
    const tick = document.createElement('i');
    tick.classList.add("fa", "fa-check", "d-inline-block", "icon-button", "text-success");
    tick.setAttribute('style', 'margin-left: 0.5rem; margin-right: 0.5rem;');
    tick.onclick = function() {
        const newEmail = entry.value;
        if (!validEmail(newEmail) || newEmail == ogEmail) {
            return
        }
        cross.remove();
        this.outerHTML = `
        <div class="spinner-border spinner-border-sm" role="status" style="width: 1rem; height: 1rem; margin-left: 0.5rem;">
            <span class="sr-only">Saving...</span>
        </div>`;
        //this.remove();
        let send = {};
        send[id] = newEmail;
        let req = new XMLHttpRequest();
        req.open("POST", "/modifyEmails", true);
        req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
        req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
        req.onreadystatechange = function() {
            if (this.readyState == 4) {
                if (this.status == 200 || this.status == 204) {
                    entry.nextElementSibling.remove();
                } else {
                    entry.value = ogEmail;
                }
            }
        };
        req.send(JSON.stringify(send));
        icon.outerHTML = iconContent;
        entry.readOnly = true;
        entry.classList.remove('form-control');
        entry.classList.add('form-control-plaintext');
        entry.placeholder = '';
    };
    const cross = document.createElement('i');
    cross.classList.add("fa", "fa-close", "d-inline-block", "icon-button", "text-danger");
    cross.onclick = function() {
        tick.remove();
        this.remove();
        icon.outerHTML = iconContent;
        entry.readOnly = true;
        entry.classList.remove('form-control');
        entry.classList.add('form-control-plaintext');
        entry.placeholder = '';
        entry.value = ogEmail;
    };
    icon.parentNode.appendChild(tick);
    icon.parentNode.appendChild(cross);
}

function populateUsers() {
    const acList = document.getElementById('accountsList');
    acList.innerHTML = `
    <div class="d-flex align-items-center">
        <strong>Getting Users...</strong>
        <div class="spinner-border ml-auto" role="status" aria-hidden="true"></div>
    </div>`;
    acList.parentNode.querySelector('thead').classList.add('unfocused');
    const accountsList = document.createElement('tbody');
    accountsList.id = 'accountsList';
    const generateEmail = function(id, name, email) {
        let entry = document.createElement('div');
        // entry.classList.add('py-1');
        entry.id = 'email_' + id;
        let emailValue = email;
        if (email === undefined) {
            emailValue = "";
        }
        entry.innerHTML = `
        <i class="fa fa-edit d-inline-block icon-button" style="margin-right: 2%;" onclick="changeEmail(this, '${id}')"></i>
        <input type="email" class="form-control-plaintext form-control-sm text-muted d-inline-block addressText" id="address_${id}" style="width: auto;" value="${emailValue}" readonly>
        `;
        return entry.outerHTML
    };
    const template = function(id, username, email, lastActive, admin) {
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

    let req = new XMLHttpRequest();
    req.responseType = 'json';
    req.open("GET", "/getUsers", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                jfUsers = req.response['users'];
                for (user of jfUsers) {
                    let tr = document.createElement('tr');
                    tr.innerHTML = template(user['id'], user['name'], user['email'], user['last_active'], user['admin']);
                    accountsList.appendChild(tr);
                }
                const header = acList.parentNode.querySelector('thead');
                if (header.classList.contains('unfocused')) {
                    header.classList.remove('unfocused');
                }
                acList.replaceWith(accountsList);
            }
        }
    };
    req.send();
}

document.getElementById('selectAll').checked = false;

document.getElementById('accountsTabSetDefaults').onclick = function() {
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    let userIDs = [];
    for (check of checkboxes) {
        if (check.checked) {
            userIDs.push(check.id.replace('select_', ''));
        }
    }
    if (userIDs.length == 0) {
        return;
    }
    populateRadios();
    let userstring = 'user';
    if (userIDs.length > 1) {
        userstring += 's';
    }
    document.getElementById('defaultsTitle').textContent = `Apply settings to ${userIDs.length} ${userstring}`;
    document.getElementById('userDefaultsDescription').textContent = `
    Create an account and configure it to your liking, then choose it from below to apply to your selected users.`;
    document.getElementById('storeHomescreenLabel').textContent = `Apply homescreen layout`;
    if (document.getElementById('defaultsSourceSection').classList.contains('unfocused')) {
        document.getElementById('defaultsSourceSection').classList.remove('unfocused');
    }
    document.getElementById('defaultsSource').value = 'userTemplate';
    document.getElementById('defaultUserRadios').classList.add('unfocused');
    document.getElementById('storeDefaults').onclick = function() {
        storeDefaults(userIDs);
    };
    userDefaultsModal.show();
};

document.getElementById('defaultsSource').addEventListener('change', function() {
    const radios = document.getElementById('defaultUserRadios');
    if (this.value == 'userTemplate') {
        radios.classList.add('unfocused');
    } else if (radios.classList.contains('unfocused')) {
        radios.classList.remove('unfocused');
    }
})

document.getElementById('newUserCreate').onclick = function() {
    const ogText = this.textContent;
    this.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Creating...`;
    const email = document.getElementById('newUserEmail').value;
    var username = email;
    if (document.getElementById('newUserName') != null) {
        username = document.getElementById('newUserName').value;
    }
    const password = document.getElementById('newUserPassword').value;
    if (!validEmail(email) && email != "") {
        return;
    }
    const send = {
        'username': username,
        'password': password,
        'email': email
    }
    let req = new XMLHttpRequest()
    req.open("POST", "/newUserAdmin", true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    const button = this;
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            button.textContent = ogText;
            if (this.status == 200) {
                if (button.classList.contains('btn-primary')) {
                    button.classList.remove('btn-primary');
                }
                button.classList.add('btn-success');
                button.textContent = 'Success';
                setTimeout(function() {
                    if (button.classList.contains('btn-success')) {
                        button.classList.remove('btn-success');
                    }
                    button.classList.add('btn-primary');
                    button.textContent = ogText;
                    newUserModal.hide();
                }, 1000);
            } else {
                if (button.classList.contains('btn-primary')) {
                    button.classList.remove('btn-primary');
                }
                button.classList.add('btn-danger');
                if ("error" in req.response) {
                    button.textContent = req.response["error"];
                } else {
                    button.textContent = 'Failed';
                }
                setTimeout(function() {
                    if (button.classList.contains('btn-danger')) {
                        button.classList.remove('btn-danger');
                    }
                    button.classList.add('btn-primary');
                    button.textContent = ogText;
                }, 2000);
            }
        }
    };
    req.send(JSON.stringify(send));
}

document.getElementById('accountsTabAddUser').onclick = function() {
    document.getElementById('newUserEmail').value = '';
    document.getElementById('newUserPassword').value = '';
    if (document.getElementById('newUserName') != null) {
        document.getElementById('newUserName').value = '';
    }
    newUserModal.show();
};
