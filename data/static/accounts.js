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
        console.log(send);
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
        return `
            <td nowrap="nowrap" class="align-middle" scope="row"><input class="form-check-input" type="checkbox" value="" id="select_${id}" onclick="checkCheckboxes();"></td>
            <td nowrap="nowrap" class="align-middle">${username}</td>
            <td nowrap="nowrap" class="align-middle">${generateEmail(id, name, email)}</td>
            <td nowrap="nowrap" class="align-middle">${lastActive}</td>
            <td nowrap="nowrap" class="align-middle">${isAdmin}</td>
            <td nowrap="nowrap" class="align-middle"><i class="fa fa-eye icon-button" id="viewConfig_${id}"></i></td>`;
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
    let radioList = document.getElementById('defaultUserRadios');
    radioList.textContent = '';
    let first = true;
    for (user of jfUsers) {
        let radio = document.createElement('div');
        radio.classList.add('radio');
        let checked = 'checked';
        if (first) {
            first = false;
        } else {
            checked = '';
        }
        radio.innerHTML = `
        <label><input type="radio" name="defaultRadios" id="select_${user['id']}" style="margin-right: 1rem;" ${checked}>${user['name']}</label>`;
        radioList.appendChild(radio);
    }
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




