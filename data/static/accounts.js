document.getElementById('selectAll').onclick = function() {
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    for (check of checkboxes) {
        check.checked = this.checked;
    }
    checkCheckboxes();
};

function checkCheckboxes() {
    const defaultsButton = document.getElementById('accountsTabSetDefaults');
    const checkboxes = document.getElementById('accountsList').querySelectorAll('input[type=checkbox]');
    let checked = false;
    for (check of checkboxes) {
        if (check.checked) {
            checked = true;
            break;
        }
    }
    if (!checked) {
        defaultsButton.classList.add('unfocused');
    } else if (defaultsButton.classList.contains('unfocused')) {
        defaultsButton.classList.remove('unfocused');
    }
}

var jfUsers = [];

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
    const template = function(id, username, email, lastActive, admin) {
        let isAdmin = "No";
        if (admin) {
            isAdmin = "Yes";
        }
        return `
            <td scope="row"><input class="form-check-input" type="checkbox" value="" id="select_${id}" onclick="checkCheckboxes();"></td>
            <td>${username}</td>
            <td>${email}</td>
            <td>${lastActive}</td>
            <td>${isAdmin}</td>
            <td><i class="fa fa-eye icon-button" id="viewConfig_${id}"></i></td>`;
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




