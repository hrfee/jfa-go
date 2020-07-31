// Used for theme change animation
function whichTransitionEvent() {
    let t;
    let el = document.createElement('fakeelement');
    let transitions = {
        'transition': 'transitionend',
        'OTransition': 'oTransitionEnd',
        'MozTransition': 'transitionend',
        'WebkitTransition': 'webkitTransitionEnd'
    };

    for (t in transitions) {
        if (el.style[t] !== undefined) {
            return transitions[t];
        }
    }
}
var transitionEndEvent = whichTransitionEvent();

// Toggles between light and dark themes
function toggleCSS() {
    let cssEl = document.querySelectorAll('link[rel="stylesheet"][type="text/css"]')[0];
    let href = "bs" + bsVersion;
    if (cssEl.href.includes(href + "-jf")) {
        href += ".css";
    } else {
        href += "-jf.css";
    }
    cssEl.href = href
    document.cookie = "css=" + href;
}

// Toggles between light and dark themes, but runs animation if necessary (dependent on window width for performance) 
var buttonWidth = 0;
function toggleCSSAnim(el) {
    let switchToColor = window.getComputedStyle(document.body, null).backgroundColor;
    let maxWidth = 1500;
    if (window.innerWidth < maxWidth) {
        // Calculate minimum radius to cover whole screen
        let radius = Math.sqrt(Math.pow(window.innerWidth, 2) + Math.pow(window.innerHeight, 2));
        let currentRadius = el.getBoundingClientRect().width / 2;
        let scale = radius / currentRadius;
        buttonWidth = window.getComputedStyle(el, null).width;
        document.body.classList.remove('smooth-transition');
        el.style.transform = `scale(${scale})`;
        el.style.color = switchToColor;
        el.addEventListener(transitionEndEvent, function() {
            if (this.style.transform.length != 0) {
                toggleCSS();
                this.style.removeProperty('transform');
                document.body.classList.add('smooth-transition');
            };
        }, false);
    } else {
        toggleCSS();
        el.style.color = switchToColor;
    }
}

var buttonColor = "custom";
// Predefined colors for 'theme' button
if (cssFile.includes("jf")) {
    buttonColor = "rgb(255,255,255)";
} else if (cssFile == ('bs' + bsVersion + '.css')) {
    buttonColor = "rgb(16,16,16)";
}

if (buttonColor != "custom") {
    let fakeButton = document.createElement('i');
    fakeButton.classList.add('fa', 'fa-circle', 'circle');
    fakeButton.style = `color: ${buttonColor}; margin-left: 0.4rem;`;
    fakeButton.id = "fakeButton";
    let switchButton = document.createElement('button');
    switchButton.classList.add('btn', 'btn-secondary');
    switchButton.textContent = "Theme";
    switchButton.onclick = function() {
        let fakeButton = document.getElementById('fakeButton');
        toggleCSSAnim(fakeButton);
    };
    let group = document.getElementById('headerButtons');
    switchButton.appendChild(fakeButton);
    group.appendChild(switchButton);
}


var loginModal = createModal('login');
var settingsModal = createModal('settingsMenu');
var userDefaultsModal = createModal('userDefaults');
var usersModal = createModal('users');
var restartModal = createModal('restartModal');
var refreshModal = createModal('refreshModal');

// Parsed invite: [<code>, <expires in _>, <1: Empty invite (no delete/link), 0: Actual invite>, <email address>, <remaining uses>, [<used-by>], <date created>, <notify on expiry>, <notify on creation>]
function parseInvite(invite, empty = false) {
    if (empty) {
        return ["None", "", 1];
    }
    let i = [invite["code"], "", 0, invite["email"]];
    let time = ""
    for (m of ["days", "hours", "minutes"]) {
        if (invite[m] != 0) {
            time += `${invite[m]}${m[0]} `;
        }
    }
    i[1] = `Expires in ${time.slice(0, -1)}`;
    if ('remaining-uses' in invite) {
        i[4] = invite['remaining-uses'];
    } 
    if (invite['no-limit']) {
        i[4] = '∞';
    }
    if ('used-by' in invite) {
        i[5] = invite['used-by'];
    } else {
        i[5] = [];
    }
    if ('created' in invite) {
        i[6] = invite['created'];
    }
    if ('notify-expiry' in invite) {
        i[7] = invite['notify-expiry'];
    }
    if ('notify-creation' in invite) {
        i[8] = invite['notify-creation'];
    }
    return i;
}

function addItem(parsedInvite) {
    let links = document.getElementById('invites');
    let itemContainer = document.createElement('div');
    itemContainer.id = parsedInvite[0];
    let listItem = document.createElement('div');
    // listItem.id = parsedInvite[0];
    listItem.classList.add('list-group-item', 'd-flex', 'justify-content-between', 'd-inline-block');

    let code = document.createElement('div');
    code.classList.add('d-flex', 'align-items-center', 'font-monospace');
    let codeLink = document.createElement('a');
    codeLink.setAttribute('style', 'margin-right: 0.5rem;');
    codeLink.textContent = parsedInvite[0].replace(/-/g, '-');

    code.appendChild(codeLink);

    listItem.appendChild(code);

    let listRight = document.createElement('div');
    let listText = document.createElement('span');
    listText.id = parsedInvite[0] + '_expiry';
    listText.setAttribute('style', 'margin-right: 1rem;');
    listText.textContent = parsedInvite[1];

    listRight.appendChild(listText);

    if (parsedInvite[2] == 0) {
        let inviteCode = window.location.href.split('#')[0] + 'invite/' + parsedInvite[0];
        //
        codeLink.href = inviteCode;
        let copyButton = document.createElement('i');
        copyButton.onclick = function() { toClipboard(inviteCode); };
        copyButton.classList.add('fa', 'fa-clipboard', 'icon-button');
        
        code.appendChild(copyButton);

        if (parsedInvite[3] !== undefined) {
            let sentTo = document.createElement('span');
            sentTo.classList.add('text-muted');
            sentTo.setAttribute('style', 'margin-left: 0.4rem; font-style: italic, font-size: 0.75rem;');
            if (!parsedInvite[3].includes('Failed to send to')) {
                sentTo.textContent = "Sent to ";
            }
            sentTo.textContent += parsedInvite[3];

            code.appendChild(sentTo);
        }

        let deleteButton = document.createElement('button');
        deleteButton.onclick = function() { deleteInvite(parsedInvite[0]); };
        deleteButton.classList.add('btn', 'btn-outline-danger');
        deleteButton.textContent = "Delete";
        
        listRight.appendChild(deleteButton);
        let dropButton = document.createElement('i');
        dropButton.classList.add('fa', 'fa-angle-down', 'collapsed', 'icon-button', 'not-rotated');
        dropButton.setAttribute('data-toggle', 'collapse');
        dropButton.setAttribute('aria-expanded', 'false');
        dropButton.setAttribute('data-target', '#' + CSS.escape(parsedInvite[0]) + '_collapse');
        dropButton.onclick = function() {
            if (this.classList.contains('rotated')) {
                this.classList.remove('rotated');
                this.classList.add('not-rotated');
            } else {
                this.classList.remove('not-rotated');
                this.classList.add('rotated');
            }
        };
        dropButton.setAttribute('style', 'margin-left: 1rem;');
        listRight.appendChild(dropButton);
    }
    
    listItem.appendChild(listRight);
    itemContainer.appendChild(listItem);
    if (parsedInvite[2] == 0) {
        let itemDropdown = document.createElement('div');
        itemDropdown.id = parsedInvite[0] + '_collapse';
        itemDropdown.classList.add('collapse');

        let dropdownContent = document.createElement('div');
        dropdownContent.classList.add('container', 'row', 'align-items-start', 'card-body');
        
        let dropdownLeft = document.createElement('div');
        dropdownLeft.classList.add('col');
        
        let leftList = document.createElement('ul');
        leftList.classList.add('list-group', 'list-group-flush');
        
        if (typeof(parsedInvite[6]) != 'undefined') {
            let createdDate = document.createElement('li');
            createdDate.classList.add('list-group-item', 'py-1');
            createdDate.textContent = `Created: ${parsedInvite[6]}`;
            leftList.appendChild(createdDate);
        }

        let remainingUses = document.createElement('li');
        remainingUses.classList.add('list-group-item', 'py-1');
        remainingUses.id = parsedInvite[0] + '_remainingUses';
        remainingUses.textContent = `Remaining uses: ${parsedInvite[4]}`;
        leftList.appendChild(remainingUses);

        dropdownLeft.appendChild(leftList);
        dropdownContent.appendChild(dropdownLeft);
        
        if (notifications_enabled) {
            let dropdownMiddle = document.createElement('div');
            dropdownMiddle.id = parsedInvite[0] + '_notifyButtons';
            dropdownMiddle.classList.add('col');

            let middleList = document.createElement('ul');
            middleList.classList.add('list-group', 'list-group-flush');
            middleList.textContent = 'Notify on:';

            let notifyExpiry = document.createElement('li');
            notifyExpiry.classList.add('list-group-item', 'py-1', 'form-check');
            notifyExpiry.innerHTML = `
            <input class="form-check-input" type="checkbox" value="" id="${parsedInvite[0]}_notifyExpiry">
            <label class="form-check-label" for="${parsedInvite[0]}_notifyExpiry">Expiry</label>
            `;
            if (typeof(parsedInvite[7]) == 'boolean') {
                notifyExpiry.getElementsByTagName('input')[0].checked = parsedInvite[7];
            }

            notifyExpiry.getElementsByTagName('input')[0].onclick = function() {
                let req = new XMLHttpRequest();
                var thisEl = this;
                let send = {};
                let code = thisEl.id.replace('_notifyExpiry', '');
                send[code] = {};
                send[code]['notify-expiry'] = thisEl.checked;
                req.open("POST", "/setNotify", true);
                req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
                req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
                req.onreadystatechange = function() {
                    if (this.readyState == 4 && this.status != 200) {
                        thisEl.checked = !thisEl.checked;
                    }
                };
                req.send(JSON.stringify(send));
            };
            middleList.appendChild(notifyExpiry);

            let notifyCreation = document.createElement('li');
            notifyCreation.classList.add('list-group-item', 'py-1', 'form-check');
            notifyCreation.innerHTML = `
            <input class="form-check-input" type="checkbox" value="" id="${parsedInvite[0]}_notifyCreation">
            <label class="form-check-label" for="${parsedInvite[0]}_notifyCreation">User creation</label>
            `;
            if (typeof(parsedInvite[8]) == 'boolean') {
                notifyCreation.getElementsByTagName('input')[0].checked = parsedInvite[8];
            }
            notifyCreation.getElementsByTagName('input')[0].onclick = function() {
                let req = new XMLHttpRequest();
                var thisEl = this;
                let send = {};
                let code = thisEl.id.replace('_notifyCreation', '');
                send[code] = {};
                send[code]['notify-creation'] = thisEl.checked;
                req.open("POST", "/setNotify", true);
                req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
                req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
                req.onreadystatechange = function() {
                    if (this.readyState == 4 && this.status != 200) {
                        thisEl.checked = !thisEl.checked;
                    }
                };
                req.send(JSON.stringify(send));
            };
            middleList.appendChild(notifyCreation);

            dropdownMiddle.appendChild(middleList);
            dropdownContent.appendChild(dropdownMiddle);
        }


        let dropdownRight = document.createElement('div');
        dropdownRight.id = parsedInvite[0] + '_usersCreated';
        dropdownRight.classList.add('col');
        if (parsedInvite[5].length != 0) {
            let userList = document.createElement('ul');
            userList.classList.add('list-group', 'list-group-flush');
            userList.innerHTML = '<li class="list-group-item py-1">Users created:</li>';
            for (let user of parsedInvite[5]) {
                let li = document.createElement('li');
                li.classList.add('list-group-item', 'py-1', 'disabled');
                let username = document.createElement('div');
                username.classList.add('d-flex', 'float-left');
                username.textContent = user[0];
                li.appendChild(username);
                let date = document.createElement('div');
                date.classList.add('d-flex', 'float-right');
                date.textContent = user[1];
                li.appendChild(date);
                userList.appendChild(li);
            }
            dropdownRight.appendChild(userList);
        }
        dropdownContent.appendChild(dropdownRight);

        itemDropdown.appendChild(dropdownContent);

        itemContainer.appendChild(itemDropdown);
    }
    links.appendChild(itemContainer);
}

function updateInvite(parsedInvite) {
    let expiry = document.getElementById(parsedInvite[0] + '_expiry');
    expiry.textContent = parsedInvite[1];

    let remainingUses = document.getElementById(parsedInvite[0] + '_remainingUses');
    if (remainingUses) {
        remainingUses.textContent = `Remaining uses: ${parsedInvite[4]}`;
    }

    if (parsedInvite[5].length != 0) {
        let usersCreated = document.getElementById(parsedInvite[0] + '_usersCreated'); 
        let dropdownRight = document.createElement('div');
        dropdownRight.id = parsedInvite[0] + '_usersCreated';
        dropdownRight.classList.add('col');
        let userList = document.createElement('ul');
        userList.classList.add('list-group', 'list-group-flush');
        userList.innerHTML = '<li class="list-group-item py-1">Users created:</li>';
        for (let user of parsedInvite[5]) {
            let li = document.createElement('li');
            li.classList.add('list-group-item', 'py-1', 'disabled');
            let username = document.createElement('div');
            username.classList.add('d-flex', 'float-left');
            username.textContent = user[0];
            li.appendChild(username);
            let date = document.createElement('div');
            date.classList.add('d-flex', 'float-right');
            date.textContent = user[1];
            li.appendChild(date);
            userList.appendChild(li);
        }
        dropdownRight.appendChild(userList);
        usersCreated.replaceWith(dropdownRight);
    }
    

}

// delete from list on page
function removeInvite(code) {
    let item = document.getElementById(code);
    item.parentNode.removeChild(item);
}

function generateInvites(empty = false) {
    if (empty === false) {
        let req = new XMLHttpRequest();
        req.open("GET", "/getInvites", true);
        req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
        req.responseType = 'json';
        req.onreadystatechange = function() {
            if (this.readyState == 4) {
                var data = this.response;
                if (data['invites'] == null || data['invites'].length == 0) {
                    document.getElementById('invites').textContent = '';
                    addItem(parseInvite([], true));
                } else {
                    for (let invite of data['invites']) {
                        let match = false;
                        let items = document.getElementById('invites').children;
                        for (let item of items) {
                            if (item.id == invite['code']) {
                                match = true;
                                updateInvite(parseInvite(invite));
                            }
                        }
                        if (match == false) {
                            addItem(parseInvite(invite));
                        }
                    }
                    let items = document.getElementById('invites').children;
                    for (let item of items) {
                        var exists = false;
                        for (let invite of data['invites']) {
                            if (item.id == invite['code']) {
                                exists = true;
                            }
                        }
                        if (exists == false) {
                            removeInvite(item.id);
                        }
                    }
                }
            }
        };
        req.send();
    } else if (empty === true) {
        document.getElementById('invites').textContent = '';
        addItem(parseInvite([], true));
    }
}

// actually delete invite
function deleteInvite(code) {
    let send = JSON.stringify({ "code": code });
    let req = new XMLHttpRequest();
    req.open("POST", "/deleteInvite", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            generateInvites();
        }
    };
    req.send(send);
}

// Add numbers to select element
function addOptions(length, selectElement) {
    for (let v = 0; v <= length; v++) {
        let opt = document.createElement('option');
        opt.textContent = v;
        opt.value = v;
        selectElement.appendChild(opt);
    }
}

function toClipboard(str) {
    const el = document.createElement('textarea');
    el.value = str;
    el.setAttribute('readOnly', '');
    el.style.position = 'absolute';
    el.style.left = '-9999px';
    document.body.appendChild(el);
    const selected = document.getSelection().rangeCount > 0 ? document.getSelection().getRangeAt(0) : false;
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
    if (selected) {
        document.getSelection().removeAllRanges();
        document.getSelection().addRange(selected);
    }
}

function fixCheckboxes() {
    let send_to_address = [document.getElementById('send_to_address'), document.getElementById('send_to_address_enabled')]
    if (send_to_address[0] != null) {
        send_to_address[0].disabled = !send_to_address[1].checked;
    }
    let multiUseEnabled = document.getElementById('multiUseEnabled');
    let multiUseCount = document.getElementById('multiUseCount');
    let noUseLimit = document.getElementById('noUseLimit');
    multiUseCount.disabled = !multiUseEnabled.checked;
    noUseLimit.checked = false;
    noUseLimit.disabled = !multiUseEnabled.checked;
}

fixCheckboxes();

document.getElementById('inviteForm').onsubmit = function() {
    let button = document.getElementById('generateSubmit');
    button.disabled = true;
    button.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    send_object = serializeForm('inviteForm');
    if (!send_object['multiple-uses'] || send_object['no-limit']) {
        delete send_object['remaining-uses'];
    }
    if (document.getElementById('send_to_address') != null) {
        if (send_object['send_to_address_enabled']) {
            send_object['email'] = send_object['send_to_address'];
            delete send_object['send_to_address'];
            delete send_object['send_to_address_enabled'];
        }
    }
    let send = JSON.stringify(send_object);
    let req = new XMLHttpRequest();
    req.open("POST", "/generateInvite", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            button.textContent = 'Generate';
            button.disabled = false;
            generateInvites();
        }
    };
    req.send(send);
    return false;
};

document.getElementById('loginForm').onsubmit = function() {
    window.token = "";
    let details = serializeForm('loginForm');
    let errorArea = document.getElementById('loginErrorArea');
    errorArea.textContent = '';
    let button = document.getElementById('loginSubmit');
    button.disabled = true;
    button.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    let req = new XMLHttpRequest();
    req.responseType = 'json';
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 401) {
                button.disabled = false;
                button.textContent = 'Login';
                let wrongPassword = document.createElement('div');
                wrongPassword.classList.add('alert', 'alert-danger');
                wrongPassword.setAttribute('role', 'alert');
                wrongPassword.textContent = "Incorrect username or password.";
                errorArea.appendChild(wrongPassword);
            } else {
                const data = this.response;
                window.token = data['token'];
                generateInvites();
                const interval = setInterval(function() { generateInvites(); }, 60 * 1000);
                let day = document.getElementById('days');
                addOptions(30, day);
                day.selected = "0";
                let hour = document.getElementById('hours');
                addOptions(24, hour);
                hour.selected = "0";
                let minutes = document.getElementById('minutes');
                addOptions(59, minutes);
                minutes.selected = "30";
                loginModal.hide();
            }
        }
    };
    req.open("GET", "/getToken", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(details['username'] + ":" + details['password']));
    req.send();
    return false;
};

document.getElementById('openDefaultsWizard').onclick = function() {
    this.disabled = true
    this.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    let req = new XMLHttpRequest();
    req.responseType = 'json';
    req.open("GET", "/getUsers", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                let users = req.response['users'];
                let radioList = document.getElementById('defaultUserRadios');
                radioList.textContent = '';
                if (document.getElementById('setDefaultUser')) {
                    document.getElementById('setDefaultUser').remove();
                }
                let first = true;
                for (user of users) {
                    let radio = document.createElement('div');
                    radio.classList.add('radio');
                    let checked = 'checked';
                    if (first) {
                        first = false;
                    } else {
                        checked = '';
                    };
                    radio.innerHTML =
                        `<label><input type="radio" name="defaultRadios" id="default_${user['name']}" style="margin-right: 1rem;" ${checked}>${user['name']}</label>`;
                    radioList.appendChild(radio);
                }
                let button = document.getElementById('openDefaultsWizard');
                button.disabled = false;
                button.innerHTML = 'Set new account defaults';
                let submitButton = document.getElementById('storeDefaults');
                submitButton.disabled = false;
                submitButton.textContent = 'Submit';
                if (submitButton.classList.contains('btn-success')) {
                    submitButton.classList.remove('btn-success');
                    submitButton.classList.add('btn-primary');
                } else if (submitButton.classList.contains('btn-danger')) {
                    submitButton.classList.remove('btn-danger');
                    submitButton.classList.add('btn-primary');
                }
                settingsModal.hide();
                userDefaultsModal.show();
            }
        }
    };
    req.send();
};

document.getElementById('storeDefaults').onclick = function () {
    this.disabled = true;
    this.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    let button = document.getElementById('storeDefaults');
    let radios = document.getElementsByName('defaultRadios');
    for (let radio of radios) {
        if (radio.checked) {
            let data = {
                'username': radio.id.slice(8),
                'homescreen': false};
            if (document.getElementById('storeDefaultHomescreen').checked) {
                data['homescreen'] = true;
            }
            let req = new XMLHttpRequest();
            req.open("POST", "/setDefaults", true);
            req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
            req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
            req.onreadystatechange = function() {
                if (this.readyState == 4) {
                    if (this.status == 200 || this.status == 204) {
                        button.textContent = "Success";
                        if (button.classList.contains('btn-danger')) {
                            button.classList.remove('btn-danger');
                        } else if (button.classList.contains('btn-primary')) {
                            button.classList.remove('btn-primary');
                        }
                        button.classList.add('btn-success');
                        button.disabled = false;
                        setTimeout(function() { userDefaultsModal.hide(); }, 1000);
                    } else {
                        button.textContent = "Failed";
                        button.classList.remove('btn-primary');
                        button.classList.add('btn-danger');
                        setTimeout(function() {
                            let button = document.getElementById('storeDefaults');
                            button.textContent = "Submit";
                            button.classList.remove('btn-danger');
                            button.classList.add('btn-primary');
                            button.disabled = false;
                        }, 1000);
                    }
                }
            };
            req.send(JSON.stringify(data));
        }
    }
};

document.getElementById('openUsers').onclick = function () {
    this.disabled = true;
    this.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    let req = new XMLHttpRequest();
    req.open("GET", "/getUsers", true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                let list = document.getElementById('userList');
                list.textContent = '';
                if (document.getElementById('saveUsers')) {
                    document.getElementById('saveUsers').remove();
                }
                let users = req.response['users'];
                for (let user of users) {
                    let entry = document.createElement('div');
                    entry.classList.add('form-group', 'list-group-item', 'py-1');
                    entry.id = 'user_' + user['name'];
                    let label = document.createElement('label');
                    label.classList.add('d-inline-block');
                    label.setAttribute('for', 'address_' + user['email']);
                    label.textContent = user['name'];
                    entry.appendChild(label);
                    let address = document.createElement('input');
                    address.setAttribute('type', 'email');
                    address.readOnly = true;
                    address.classList.add('form-control-plaintext', 'text-muted', 'd-inline-block', 'addressText');
                    address.id = 'address_' + user['email'];
                    if (typeof(user['email']) != 'undefined') {
                        address.value = user['email'];
                        address.setAttribute('style', 'width: auto; margin-left: 2%;');
                    }
                    let editButton = document.createElement('i');
                    editButton.classList.add('fa', 'fa-edit', 'd-inline-block', 'icon-button');
                    editButton.setAttribute('style', 'margin-left: 2%;');
                    editButton.onclick = function() {
                        this.classList.remove('fa', 'fa-edit');
                        let addressElement = this.parentNode.getElementsByClassName('form-control-plaintext')[0];
                        addressElement.classList.remove('form-control-plaintext', 'text-muted');
                        addressElement.classList.add('form-control');
                        addressElement.readOnly = false;
                        if (addressElement.value == '') {
                            addressElement.placeholder = 'Email Address';
                            address.setAttribute('style', 'width: auto; margin-left: 2%;');
                        }
                        if (document.getElementById('saveUsers') == null) {
                            let footer = document.getElementById('userFooter')
                            let saveUsers = document.createElement('input');
                            saveUsers.classList.add('btn', 'btn-primary');
                            saveUsers.setAttribute('type', 'button');
                            saveUsers.value = 'Save Changes';
                            saveUsers.id = 'saveUsers';
                            saveUsers.onclick = function() {
                                let send = {}
                                let entries = document.getElementById('userList').children;
                                for (let entry of entries) {
                                    if (typeof(entry.getElementsByTagName('input')[0]) != 'undefined') {
                                        const name = entry.id.replace(/user_/g, '');
                                        const address = entry.getElementsByTagName('input')[0].value;
                                        send[name] = address;
                                    }
                                }
                                send = JSON.stringify(send);
                                let req = new XMLHttpRequest();
                                req.open("POST", "/modifyUsers", true);
                                req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
                                req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
                                req.onreadystatechange = function() {
                                    if (this.readyState == 4) {
                                        if (this.status == 200 || this.status == 204) {
                                            usersModal.hide();
                                        }
                                    }
                                };
                                req.send(send);
                            };
                            footer.appendChild(saveUsers);
                        }
                    };
                    entry.appendChild(editButton);
                    entry.appendChild(address);
                    list.appendChild(entry);
                };
                let button = document.getElementById('openUsers');
                button.disabled = false;
                button.innerHTML = 'Users <i class="fa fa-user"></i>';
                settingsModal.hide();
                usersModal.show();
            }
        }
    };
    req.send();
};

generateInvites(empty = true);
loginModal.show();

var config = {};
var modifiedConfig = {};

document.getElementById('openSettings').onclick = function () {
    let req = new XMLHttpRequest();
    req.open("GET", "/getConfig", true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            let settingsList = document.getElementById('settingsList');
            settingsList.textContent = '';
            config = this.response;
            for (let section of Object.keys(config)) {
                let sectionCollapse = document.createElement('div');
                sectionCollapse.classList.add('collapse');
                sectionCollapse.id = section;
                
                let sectionTitle = config[section]['meta']['name'];
                let sectionDescription = config[section]['meta']['description'];
                let entryListID = section + '_entryList';
                let sectionFooter = section + '_footer';

                let innerCollapse = `
                <div class="card card-body">
                    <small class="text-muted">${sectionDescription}</small>
                    <div class="${entryListID}">
                    </div>
                </div>
                `;

                sectionCollapse.innerHTML = innerCollapse;
                
                for (var entry of Object.keys(config[section])) {
                    if (entry != 'meta') {
                        let entryName = config[section][entry]['name'];
                        let required = false;
                        if (config[section][entry]['required']) {
                            entryName += ' <sup class="text-danger">*</sup>';
                            required = true;
                        }
                        if (config[section][entry]['requires_restart']) {
                            entryName += ' <sup class="text-danger">R</sup>';
                        }
                        if (config[section][entry].hasOwnProperty('description')) {
                            let tooltip = `
                            <a class="text-muted" href="#" data-toggle="tooltip" data-placement="right" title="${config[section][entry]['description']}"><i class="fa fa-question-circle-o"></i></a>
                            `;
                            entryName += ' ';
                            entryName += tooltip;
                        };
                        let entryValue = config[section][entry]['value'];
                        let entryType = config[section][entry]['type'];
                        let entryGroup = document.createElement('div');
                        if (entryType == 'bool') {
                            entryGroup.classList.add('form-check');
                            if (entryValue.toString() == 'true') {
                                var checked = true;
                            } else {
                                var checked = false;
                            }
                            entryGroup.innerHTML = `
                            <input class="form-check-input" type="checkbox" value="" id="${section}_${entry}">
                            <label class="form-check-label" for="${section}_${entry}">${entryName}</label>
                            `;
                            entryGroup.getElementsByClassName('form-check-input')[0].required = required;
                            entryGroup.getElementsByClassName('form-check-input')[0].checked = checked;
                            entryGroup.getElementsByClassName('form-check-input')[0].onclick = function() {
                                var state = this.checked;
                                for (var sect of Object.keys(config)) {
                                    for (var ent of Object.keys(config[sect])) {
                                        if ((sect + '_' + config[sect][ent]['depends_true']) == this.id) {
                                            document.getElementById(sect + '_' + ent).disabled = !state;
                                        } else if ((sect + '_' + config[sect][ent]['depends_false']) == this.id) {
                                            document.getElementById(sect + '_' + ent).disabled = state;
                                        }
                                    }
                                }
                            };
                        } else if ((entryType == 'text') || (entryType == 'email') || (entryType == 'password') || (entryType == 'number')) {
                            entryGroup.classList.add('form-group');
                            entryGroup.innerHTML = `
                            <label for="${section}_${entry}">${entryName}</label>
                            <input type="${entryType}" class="form-control" id="${section}_${entry}" aria-describedby="${entry}" value="${entryValue}">
                            `;
                            entryGroup.getElementsByClassName('form-control')[0].required = required;
                        } else if (entryType == 'select') {
                            entryGroup.classList.add('form-group');
                            let entryOptions = config[section][entry]['options'];
                            let innerGroup = `
                            <label for="${section}_${entry}">${entryName}</label>
                            <select class="form-control" id="${section}_${entry}">
                            `;
                            for (let entryOption of entryOptions) {
                                if (entryOption == entryValue) {
                                    var selected = 'selected';
                                } else {
                                    var selected = '';
                                }
                                innerGroup += `
                                <option value="${entryOption}" ${selected}>${entryOption}</option>
                                `;
                            }
                            innerGroup += '</select>';
                            entryGroup.innerHTML = innerGroup;
                            entryGroup.getElementsByClassName('form-control')[0].required = required;
                            
                        }
                        sectionCollapse.getElementsByClassName(entryListID)[0].appendChild(entryGroup);
                    }
                }
                let sectionButton = document.createElement('button');
                sectionButton.setAttribute('type', 'button');
                sectionButton.classList.add('list-group-item', 'list-group-item-action');
                sectionButton.appendChild(document.createTextNode(sectionTitle));
                sectionButton.id = section + '_button';
                sectionButton.setAttribute('data-toggle', 'collapse');
                sectionButton.setAttribute('data-target', '#' + section);
                settingsList.appendChild(sectionButton);
                settingsList.appendChild(sectionCollapse);
            }
        }
    };
    req.send();
    settingsModal.show();
}

triggerTooltips();

function sendConfig(modalId, restart = false) {
    let modal = document.getElementById(modalId);
    modifiedConfig['restart-program'] = false;
    if (restart) {
        modifiedConfig['restart-program'] = true;
    }
    let send = JSON.stringify(modifiedConfig);
    let req = new XMLHttpRequest();
    req.open("POST", "/modifyConfig", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(window.token + ":"));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200 || this.status == 204) {
                createModal(modalId, true).hide();
                if (modalId != 'settingsMenu') {
                    settingsModal.hide();
                }
            } else if (restart) {
                refreshModal.show();
            }
        }
    };
    req.send(send);
}

document.getElementById('settingsSave').onclick = function() {
    modifiedConfig = {};
    var restart_setting_changed = false;
    var settings_changed = false;
    
    for (let section of Object.keys(config)) {
        for (let entry of Object.keys(config[section])) {
            if (entry != 'meta') {
                let entryID = section + '_' + entry;
                let el = document.getElementById(entryID);
                if (el.type == 'checkbox') {
                    var value = el.checked.toString();
                } else {
                    var value = el.value.toString();
                }
                if (value != config[section][entry]['value'].toString()) {
                    if (!modifiedConfig.hasOwnProperty(section)) {
                        modifiedConfig[section] = {};
                    }
                    modifiedConfig[section][entry] = value;
                    settings_changed = true;
                    if (config[section][entry]['requires_restart']) {
                        restart_setting_changed = true;
                    }
                }
            }
        }
    }
    if (restart_setting_changed) {
        document.getElementById('applyRestarts').onclick = function(){ sendConfig('restartModal'); };
        document.getElementById('applyAndRestart').onclick = function(){ sendConfig('restartModal', restart=true); };
        settingsModal.hide();
        restartModal.show();
    } else if (settings_changed) {
        sendConfig('settingsMenu');
    } else {
        settingsModal.hide();
    }
}

