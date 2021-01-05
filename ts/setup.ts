// Lord forgive me for this mess, i'll fix it one day i swear

document.getElementById("page-1").scrollIntoView({
    behavior: "auto",
    block: "center",
    inline: "center"
});

const checkAuthRadio = () => {
    if ((document.getElementById('manualAuthRadio') as HTMLInputElement).checked) {
        document.getElementById('adminOnlyArea').style.display = 'none';
        document.getElementById('manualAuthArea').style.display = '';
    } else {
        document.getElementById('manualAuthArea').style.display = 'none';
        document.getElementById('adminOnlyArea').style.display = '';
    }
};

for (let radio of ['manualAuthRadio', 'jfAuthRadio']) {
    document.getElementById(radio).addEventListener('change', checkAuthRadio);
};

const checkEmailRadio = () => {
    (document.getElementById('emailNextButton') as HTMLAnchorElement).href = '#page-5';
    (document.getElementById('valBackButton') as HTMLAnchorElement).href = '#page-7';
    if ((document.getElementById('emailSMTPRadio') as HTMLInputElement).checked) {
        document.getElementById('emailCommonArea').style.display = '';
        document.getElementById('emailSMTPArea').style.display = '';
        document.getElementById('emailMailgunArea').style.display = 'none';
        (document.getElementById('notificationsEnabled') as HTMLInputElement).checked = true;
    } else if ((document.getElementById('emailMailgunRadio') as HTMLInputElement).checked) {
        document.getElementById('emailCommonArea').style.display = '';
        document.getElementById('emailSMTPArea').style.display = 'none';
        document.getElementById('emailMailgunArea').style.display = '';
        (document.getElementById('notificationsEnabled') as HTMLInputElement).checked = true;
    } else if ((document.getElementById('emailDisabledRadio') as HTMLInputElement).checked) {
        document.getElementById('emailCommonArea').style.display = 'none';
        document.getElementById('emailSMTPArea').style.display = 'none';
        document.getElementById('emailMailgunArea').style.display = 'none';
        (document.getElementById('emailNextButton') as HTMLAnchorElement).href = '#page-8';
        (document.getElementById('valBackButton') as HTMLAnchorElement).href = '#page-4';
        (document.getElementById('notificationsEnabled') as HTMLInputElement).checked = false;
    }
};

for (let radio of ['emailDisabledRadio', 'emailSMTPRadio', 'emailMailgunRadio']) {
    document.getElementById(radio).addEventListener('change', checkEmailRadio);
}

const checkSSL = () => {
    var label = document.getElementById('emailSSL_TLSLabel');
    if ((document.getElementById('emailSSL_TLS') as HTMLInputElement).checked) {
        label.textContent = 'Use SSL/TLS';
    } else {
        label.textContent = 'Use STARTTLS';
    }
};
document.getElementById('emailSSL_TLS').addEventListener('change', checkSSL);

var pwrEnabled = document.getElementById('pwrEnabled') as HTMLInputElement;
const checkPwrEnabled = () => {
    if (pwrEnabled.checked) {
        document.getElementById('pwrArea').style.display = '';
    } else {
        document.getElementById('pwrArea').style.display = 'none';
    }
};
pwrEnabled.addEventListener('change', checkPwrEnabled);

var invEnabled = document.getElementById("invEnabled") as HTMLInputElement;
const checkInvEnabled = () => {
    if (invEnabled.checked) {
        document.getElementById('invArea').style.display = '';
    } else {
        document.getElementById('invArea').style.display = 'none';
    }
};
invEnabled.addEventListener('change', checkInvEnabled);

var valEnabled = document.getElementById("valEnabled") as HTMLInputElement;
const checkValEnabled = () => {
    const valArea = document.getElementById("valArea");
    if (valEnabled.checked) {
        valArea.style.display = '';
    } else {
        valArea.style.display = 'none';
    }
};
valEnabled.addEventListener('change', checkValEnabled);

checkValEnabled();
checkInvEnabled();
checkSSL();
checkAuthRadio();
checkEmailRadio();
checkPwrEnabled();

var jfValid = false
document.getElementById('jfTestButton').onclick = () => {
    let testButton = document.getElementById('jfTestButton') as HTMLInputElement;
    let nextButton = document.getElementById('jfNextButton') as HTMLAnchorElement;
    let jfData = {};
    jfData['jfHost'] = (document.getElementById('jfHost') as HTMLInputElement).value;
    jfData['jfUser'] = (document.getElementById('jfUser') as HTMLInputElement).value;
    jfData['jfPassword'] = (document.getElementById('jfPassword') as HTMLInputElement).value;
    let valid = true;
    for (let val in jfData) {
        if (jfData[val] == "") {
            valid = false;
        }
    }
    if (!valid) {
        if (!testButton.classList.contains('btn-danger')) {
            testButton.classList.add('btn-danger');
            testButton.textContent = 'Fill out fields above.';
            setTimeout(function() {
                if (testButton.classList.contains('btn-danger')) {
                    testButton.classList.remove('btn-danger');
                    testButton.textContent = 'Test';
                }
            }, 2000);
        }
    } else {
        testButton.disabled = true;
        testButton.innerHTML =
            '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
            'Testing...';
        nextButton.classList.add('disabled');
        nextButton.setAttribute('aria-disabled', 'true');
        var req = new XMLHttpRequest();
        req.open("POST", "/jellyfin/test", true);
        req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
        req.responseType = 'json';
        req.onreadystatechange = function() {
            if (this.readyState == 4) {
                testButton.disabled = false;
                testButton.className = '';
                if (this.response['success'] == true) {
                    testButton.classList.add('btn', 'btn-success');
                    testButton.textContent = 'Success';
                    nextButton.classList.remove('disabled');
                    nextButton.setAttribute('aria-disabled', 'false');
                } else {
                    testButton.classList.add('btn', 'btn-danger');
                    testButton.textContent = 'Failed';
                };
            };
        };   
        req.send(JSON.stringify(jfData));
    }
};

document.getElementById('submitButton').onclick = () => {
    const submitButton = document.getElementById('submitButton') as HTMLInputElement;
    submitButton.disabled = true;
    submitButton.innerHTML =`
        <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
        Submitting...
    `;
    let config = {};
    config['jellyfin'] = {};
    config['ui'] = {};
    config['password_validation'] = {};
    config['email'] = {};
    config['password_resets'] = {};
    config['invite_emails'] = {};
    config['mailgun'] = {};
    config['smtp'] = {};
    config['notifications'] = {};
    // Page 2: Auth
    if ((document.getElementById('jfAuthRadio') as HTMLInputElement).checked) {
        config['ui']['jellyfin_login'] = 'true';
        config['ui']['admin_only'] = ""+(document.getElementById("jfAuthAdminOnly") as HTMLInputElement).checked;
    } else {
        config['ui']['username'] = (document.getElementById('manualAuthUsername') as HTMLInputElement).value;
        config['ui']['password'] = (document.getElementById('manualAuthPassword') as HTMLInputElement).value;
        config['ui']['email'] = (document.getElementById('manualAuthEmail') as HTMLInputElement).value;
    };
    // Page 3: Connect to jellyfin
    config['jellyfin']['server'] = (document.getElementById('jfHost') as HTMLInputElement).value;
    let publicAddress = (document.getElementById('jfPublicHost') as HTMLInputElement).value;
    if (publicAddress != "") {
        config['jellyfin']['public_server'] = publicAddress;
    }
    config['jellyfin']['username'] = (document.getElementById('jfUser') as HTMLInputElement).value;
    config['jellyfin']['password'] = (document.getElementById('jfPassword') as HTMLInputElement).value;
    // Page 4: Email (Page 5, 6, 7 are only used if this is enabled)
    if ((document.getElementById('emailDisabledRadio') as HTMLInputElement).checked) {
        config['password_resets']['enabled'] = 'false';
        config['invite_emails']['enabled'] = 'false';
        config['notifications']['enabled'] = 'false';
    } else {
        if ((document.getElementById('emailSMTPRadio') as HTMLInputElement).checked) {
            config['smtp']['encryption'] = (document.getElementById('emailSSL_TLS') as HTMLInputElement).checked ? "ssl_tls" : "starttls";
            config['email']['method'] = 'smtp';
            config['smtp']['server'] = (document.getElementById('emailSMTPServer') as HTMLInputElement).value;
            config['smtp']['port'] = (document.getElementById('emailSMTPPort') as HTMLInputElement).value;
            config['smtp']['password'] = (document.getElementById('emailSMTPPassword') as HTMLInputElement).value;
            config['email']['address'] = (document.getElementById('emailSMTPAddress') as HTMLInputElement).value;
        } else {
            config['email']['method'] = 'mailgun';
            config['mailgun']['api_url'] = (document.getElementById('emailMailgunURL') as HTMLInputElement).value;
            config['mailgun']['api_key'] = (document.getElementById('emailMailgunKey') as HTMLInputElement).value;
            config['email']['address'] = (document.getElementById('emailMailgunAddress') as HTMLInputElement).value;
        };
        config['notifications']['enabled'] = ""+(document.getElementById('notificationsEnabled') as HTMLInputElement).checked;
        // Page 5: Email formatting
        config['email']['from'] = (document.getElementById('emailSender') as HTMLInputElement).value;
        config['email']['date_format'] = (document.getElementById('emailDateFormat') as HTMLInputElement).value;
        config['email']['use_24h'] = ""+(document.getElementById('email24hTimeRadio') as HTMLInputElement).checked;
        config['email']['message'] = (document.getElementById('emailMessage') as HTMLInputElement).value;
        // Page 6: Password Resets
        if (pwrEnabled.checked) {
            config['password_resets']['enabled'] = 'true';
            config['password_resets']['watch_directory'] = (document.getElementById('pwrJfPath') as HTMLInputElement).value;
            config['password_resets']['subject'] = (document.getElementById('pwrSubject') as HTMLInputElement).value;
        } else {
            config['password_resets']['enabled'] = 'false';
        };
        // Page 7: Invite Emails
        if ((document.getElementById('invEnabled') as HTMLInputElement).checked) {
            config['invite_emails']['enabled'] = 'true';
            config['invite_emails']['url_base'] = (document.getElementById('invURLBase') as HTMLInputElement).value;
            config['invite_emails']['subject'] = (document.getElementById('invSubject') as HTMLInputElement).value;
        } else {
            config['invite_emails']['enabled'] = 'false';
        };
    };
    // Page 8: Password Validation
    if ((document.getElementById('valEnabled') as HTMLInputElement).checked) {
        config['password_validation']['enabled'] = 'true';
        config['password_validation']['min_length'] = (document.getElementById('valLength') as HTMLInputElement).value;
        config['password_validation']['upper'] = (document.getElementById('valUpper') as HTMLInputElement).value;
        config['password_validation']['lower'] = (document.getElementById('valLower') as HTMLInputElement).value;
        config['password_validation']['number'] = (document.getElementById('valNumber') as HTMLInputElement).value;
        config['password_validation']['special'] = (document.getElementById('valSpecial') as HTMLInputElement).value;
    } else {
        config['password_validation']['enabled'] = 'false';
    };
    // Page 9: Messages
    config['ui']['contact_message'] = (document.getElementById('msgContact') as HTMLInputElement).value;
    config['ui']['help_message'] = (document.getElementById('msgHelp') as HTMLInputElement).value;
    config['ui']['success_message'] = (document.getElementById('msgSuccess') as HTMLInputElement).value;
    // Send it
    config["restart-program"] = true;
    let req = new XMLHttpRequest();
    req.open("POST", "/config", true);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.responseType = 'json';
    req.onreadystatechange = function() {
        if (this.readyState == 4) {
            submitButton.disabled = false;
            submitButton.className = '';
            submitButton.classList.add('btn', 'btn-success');
            submitButton.textContent = 'Success';
        };
    };
    req.send(JSON.stringify(config));
};

