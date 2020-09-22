interface Window {
    token: string;
}

// Set in admin.html
var cssFile: string;

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

interface TabSwitcher {
    els: Array<HTMLDivElement>;
    tabButtons: Array<HTMLAnchorElement>;
    focus: (el: number) => void;
    invites: () => void;
    accounts: () => void;
    settings: () => void;
}

const tabs: TabSwitcher = {
    els: [document.getElementById('invitesTab') as HTMLDivElement, document.getElementById('accountsTab') as HTMLDivElement, document.getElementById('settingsTab') as HTMLDivElement],
    tabButtons: [document.getElementById('invitesTabButton') as HTMLAnchorElement, document.getElementById('accountsTabButton') as HTMLAnchorElement, document.getElementById('settingsTabButton') as HTMLAnchorElement],
    focus: (el: number): void => {
        for (let i = 0; i < tabs.els.length; i++) {
            if (i == el) {
                Focus(tabs.els[i]);
                addAttr(tabs.tabButtons[i], "active");
            } else {
                Unfocus(tabs.els[i]);
                rmAttr(tabs.tabButtons[i], "active");
            }
        }
    },
    invites: (): void => tabs.focus(0),
    accounts: (): void => {
        populateUsers();
        (document.getElementById('selectAll') as HTMLInputElement).checked = false;
        checkCheckboxes();
        tabs.focus(1);
    },
    settings: (): void => openSettings(document.getElementById('settingsSections'), document.getElementById('settingsContent'), (): void => {
        triggerTooltips();
        showSetting("ui");
        tabs.focus(2);
    })
};

// for (let i = 0; i < tabs.els.length; i++) {
//     tabs.tabButtons[i].onclick = (): void => tabs.focus(i);
// }
tabs.tabButtons[0].onclick = tabs.invites;
tabs.tabButtons[1].onclick = tabs.accounts;
tabs.tabButtons[2].onclick = tabs.settings;


tabs.invites();

// Predefined colors for the theme button.
var buttonColor: string = "custom";
if (cssFile.includes("jf")) {
    buttonColor = "rgb(255,255,255)";
} else if (cssFile == ("bs" + bsVersion + ".css")) {
    buttonColor = "rgb(16,16,16)";
}

if (buttonColor != "custom") {
    const switchButton = document.createElement('button') as HTMLButtonElement;
    switchButton.classList.add('btn', 'btn-secondary');
    switchButton.innerHTML = `
    Theme
    <i class="fa fa-circle circle" style="color: ${buttonColor}; margin-left: 0.4rem;" id="fakeButton"></i>
    `;
    switchButton.onclick = (): void => toggleCSS(document.getElementById('fakeButton'));
    document.getElementById('headerButtons').appendChild(switchButton);
}

var loginModal = createModal('login');
var userDefaultsModal = createModal('userDefaults');
var usersModal = createModal('users');
var restartModal = createModal('restartModal');
var refreshModal = createModal('refreshModal');
var aboutModal = createModal('aboutModal');
var deleteModal = createModal('deleteModal');
var newUserModal = createModal('newUserModal');

var availableProfiles: Array<string>;

window["token"] = "";

function toClipboard(str: string): void {
    const el = document.createElement('textarea') as HTMLTextAreaElement;
    el.value = str;
    el.readOnly = true;
    el.style.position = "absolute";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    const selected = document.getSelection().rangeCount > 0 ? document.getSelection().getRangeAt(0) : false;
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
    if (selected) {
        document.getSelection().removeAllRanges();
        document.getSelection().addRange(selected);
    }
}

function login(username: string, password: string, modal: boolean, button?: HTMLButtonElement, run?: (arg0: number) => void): void {
    const req = new XMLHttpRequest();
    req.responseType = 'json';
    req.open("GET", "/getToken", true);
    req.setRequestHeader("Authorization", "Basic " + btoa(username + ":" + password));
    req.onreadystatechange = function (): void {
        if (this.readyState == 4) {
            if (this.status != 200) {
                let errorMsg = this.response["error"];
                if (!errorMsg) {
                    errorMsg = "Unknown error";
                }
                if (modal) {
                    button.disabled = false;
                    button.textContent = errorMsg;
                    addAttr(button, "btn-danger");
                    rmAttr(button, "btn-primary");
                    setTimeout((): void => {
                        addAttr(button, "btn-primary");
                        rmAttr(button, "btn-danger");
                        button.textContent = "Login";
                    }, 4000);
                } else {
                    loginModal.show();
                }
            } else {
                const data = this.response;
                window.token = data["token"];
                generateInvites();
                setInterval((): void => generateInvites(), 60 * 1000);
                addOptions(30, document.getElementById('days') as HTMLSelectElement);
                addOptions(24, document.getElementById('hours') as HTMLSelectElement);
                const minutes = document.getElementById('minutes') as HTMLSelectElement;
                addOptions(59, minutes);
                minutes.value = "30";
                checkDuration();
                if (modal) {
                    loginModal.hide();
                }
                Focus(document.getElementById('logoutButton'));
            }
            if (run) {
                run(+this.status);
            }
        }
    };
    req.send();
}

function createEl(html: string): HTMLElement {
    let div = document.createElement('div') as HTMLDivElement;
    div.innerHTML = html;
    return div.firstElementChild as HTMLElement;
}

(document.getElementById('loginForm') as HTMLFormElement).onsubmit = function (): boolean {
    window.token = "";
    const details = serializeForm('loginForm');
    const button = document.getElementById('loginSubmit') as HTMLButtonElement;
    addAttr(button, "btn-primary");
    rmAttr(button, "btn-danger");
    button.disabled = true;
    button.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
    Loading...`;
    login(details["username"], details["password"], true, button);
    return false;
};

function storeDefaults(users: string | Array<string>): void {
    // not sure if this does anything, but w/e
    this.disabled = true;
    this.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    const button = document.getElementById('storeDefaults') as HTMLButtonElement;
    let data = { "homescreen": false };
    if ((document.getElementById('defaultsSource') as HTMLSelectElement).value == 'profile') {
        data["from"] = "profile";
        data["profile"] = (document.getElementById('profileSelect') as HTMLSelectElement).value;
    } else {
        const radio = document.querySelector('input[name=defaultRadios]:checked') as HTMLInputElement
        let id = radio.id.replace("default_", "");
        data["from"] = "user";
        data["id"] = id;
    }
    if (users != "all") {
        data["apply_to"] = users;
    }
    if ((document.getElementById('storeDefaultHomescreen') as HTMLInputElement).checked) {
        data["homescreen"] = true;
    }
    _post("/applySettings", data, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200 || this.status == 204) {
                button.textContent = "Success";
                addAttr(button, "btn-success");
                rmAttr(button, "btn-danger");
                rmAttr(button, "btn-primary");
                button.disabled = false;
                setTimeout((): void => {
                    button.textContent = "Submit";
                    addAttr(button, "btn-primary");
                    rmAttr(button, "btn-success");
                    button.disabled = false;
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
                    button.textContent = "Submit";
                    addAttr(button, "btn-primary");
                    rmAttr(button, "btn-danger");
                    button.disabled = false;
                }, 1000);
            }
        }
    });
} 

generateInvites(true);

login("", "", false, null, (status: number): void => {
    if (!(status == 200 || status == 204)) {
        loginModal.show();
    }
});

(document.getElementById('logoutButton') as HTMLButtonElement).onclick = function (): void {
    _post("/logout", null, function (): boolean {
        if (this.readyState == 4 && this.status == 200) {
            window.token = "";
            location.reload();
            return false;
        }
    });
};


