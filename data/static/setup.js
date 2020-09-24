document.getElementById('page-1').scrollIntoView({
    behavior: 'auto',
    block: 'center',
    inline: 'center' });

function checkAuthRadio() {
    if (document.getElementById('manualAuthRadio').checked) {
        document.getElementById('adminOnlyArea').style.display = 'none';
        document.getElementById('manualAuthArea').style.display = '';
    } else {
        document.getElementById('manualAuthArea').style.display = 'none';
        document.getElementById('adminOnlyArea').style.display = '';
    };
};
var authRadios = ['manualAuthRadio', 'jfAuthRadio'];
for (var i = 0; i < authRadios.length; i++) {
    document.getElementById(authRadios[i]).addEventListener('change', function() {
        checkAuthRadio();
    });
};

function checkEmailRadio() {
    document.getElementById('emailNextButton').href = '#page-5';
    document.getElementById('valBackButton').href = '#page-7';
    if (document.getElementById('emailSMTPRadio').checked) {
        document.getElementById('emailCommonArea').style.display = '';
        document.getElementById('emailSMTPArea').style.display = '';
        document.getElementById('emailMailgunArea').style.display = 'none';
        document.getElementById('notificationsEnabled').checked = true;
    } else if (document.getElementById('emailMailgunRadio').checked) {
        document.getElementById('emailCommonArea').style.display = '';
        document.getElementById('emailSMTPArea').style.display = 'none';
        document.getElementById('emailMailgunArea').style.display = '';
        document.getElementById('notificationsEnabled').checked = true;
    } else if (document.getElementById('emailDisabledRadio').checked) {
        document.getElementById('emailCommonArea').style.display = 'none';
        document.getElementById('emailSMTPArea').style.display = 'none';
        document.getElementById('emailMailgunArea').style.display = 'none';
        document.getElementById('emailNextButton').href = '#page-8';
        document.getElementById('valBackButton').href = '#page-4';
        document.getElementById('notificationsEnabled').checked = false;
    };
};
var emailRadios = ['emailDisabledRadio', 'emailSMTPRadio', 'emailMailgunRadio'];
for (var i = 0; i < emailRadios.length; i++) {
    document.getElementById(emailRadios[i]).addEventListener('change', function() {
        checkEmailRadio();
    });
};

function checkSSL() {
    var label = document.getElementById('emailSSL_TLSLabel');
    if (document.getElementById('emailSSL_TLS').checked) {
        label.textContent = 'Use SSL/TLS';
    } else {
        label.textContent = 'Use STARTTLS';
    };
};
document.getElementById('emailSSL_TLS').addEventListener('change', function() {
    checkSSL();
});

function checkPwrEnabled() {
    if (document.getElementById('pwrEnabled').checked) {
        document.getElementById('pwrArea').style.display = '';
    } else {
        document.getElementById('pwrArea').style.display = 'none';
    };
};
var pwrEnabled = document.getElementById('pwrEnabled');
pwrEnabled.addEventListener('change', function() {
    checkPwrEnabled();
});

function checkInvEnabled() {
    if (document.getElementById('invEnabled').checked) {
        document.getElementById('invArea').style.display = '';
    } else {
        document.getElementById('invArea').style.display = 'none';
    };
};
document.getElementById('invEnabled').addEventListener('change', function() {
    checkInvEnabled();
});

function checkValEnabled() {
    if (document.getElementById('valEnabled').checked) {
        document.getElementById('valArea').style.display = '';
    } else {
        document.getElementById('valArea').style.display = 'none';
    };
};
document.getElementById('valEnabled').addEventListener('change', function() {
    checkValEnabled();
});
checkValEnabled();
checkInvEnabled();
checkSSL();
checkAuthRadio();
checkEmailRadio();
checkPwrEnabled();

var jfValid = false
document.getElementById('jfTestButton').onclick = function() {
    var testButton = document.getElementById('jfTestButton');
    var nextButton = document.getElementById('jfNextButton');
    var jfData = {};
    jfData['jfHost'] = document.getElementById('jfHost').value;
    jfData['jfUser'] = document.getElementById('jfUser').value;
    jfData['jfPassword'] = document.getElementById('jfPassword').value;
    let valid = true;
    for (val in jfData) {
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

document.getElementById('submitButton').onclick = function() {
    var submitButton = document.getElementById('submitButton');
    submitButton.disabled = true;
    submitButton.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Submitting...';
    var config = {};
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
    if (document.getElementById('jfAuthRadio').checked) {
        config['ui']['jellyfin_login'] = 'true';
        if (document.getElementById('jfAuthAdminOnly').checked) {
            config['ui']['admin_only'] = 'true';
        } else {
            config['ui']['admin_only'] = 'false'
        };
    } else {
        config['ui']['username'] = document.getElementById('manualAuthUsername').value;
        config['ui']['password'] = document.getElementById('manualAuthPassword').value;
        config['ui']['email'] = document.getElementById('manualAuthEmail').value;
    };
    // Page 3: Connect to jellyfin
    config['jellyfin']['server'] = document.getElementById('jfHost').value;
    let publicAddress = document.getElementById('jfPublicHost').value;
    if (publicAddress != "") {
        config['jellyfin']['public_server'] = publicAddress;
    }
    config['jellyfin']['username'] = document.getElementById('jfUser').value;
    config['jellyfin']['password'] = document.getElementById('jfPassword').value;
    // Page 4: Email (Page 5, 6, 7 are only used if this is enabled)
    if (document.getElementById('emailDisabledRadio').checked) {
        config['password_resets']['enabled'] = 'false';
        config['invite_emails']['enabled'] = 'false';
        config['notifications']['enabled'] = 'false';
    } else {
        if (document.getElementById('emailSMTPRadio').checked) {
            if (document.getElementById('emailSSL_TLS').checked) {
                config['smtp']['encryption'] = 'ssl_tls';
            } else {
                config['smtp']['encryption'] = 'starttls';
            };
            config['email']['method'] = 'smtp';
            config['smtp']['server'] = document.getElementById('emailSMTPServer').value;
            config['smtp']['port'] = document.getElementById('emailSMTPPort').value;
            config['smtp']['password'] = document.getElementById('emailSMTPPassword').value;
            config['email']['address'] = document.getElementById('emailSMTPAddress').value;
        } else {
            config['email']['method'] = 'mailgun';
            config['mailgun']['api_url'] = document.getElementById('emailMailgunURL').value;
            config['mailgun']['api_key'] = document.getElementById('emailMailgunKey').value;
            config['email']['address'] = document.getElementById('emailMailgunAddress').value;
        };
        config['notifications']['enabled'] = document.getElementById('notificationsEnabled').checked.toString();
        // Page 5: Email formatting
        config['email']['from'] = document.getElementById('emailSender').value;
        config['email']['date_format'] = document.getElementById('emailDateFormat').value;
        if (document.getElementById('email24hTimeRadio').checked) {
            config['email']['use_24h'] = 'true';
        } else {
            config['email']['use_24h'] = 'false';
        };
        config['email']['message'] = document.getElementById('emailMessage').value;
        // Page 6: Password Resets
        if (document.getElementById('pwrEnabled').checked) {
            config['password_resets']['enabled'] = 'true';
            config['password_resets']['watch_directory'] = document.getElementById('pwrJfPath').value;
            config['password_resets']['subject'] = document.getElementById('pwrSubject').value;
        } else {
            config['password_resets']['enabled'] = 'false';
        };
        // Page 7: Invite Emails
        if (document.getElementById('invEnabled').checked) {
            config['invite_emails']['enabled'] = 'true';
            config['invite_emails']['url_base'] = document.getElementById('invURLBase').value;
            config['invite_emails']['subject'] = document.getElementById('invSubject').value;
        } else {
            config['invite_emails']['enabled'] = 'false';
        };
    };
    // Page 8: Password Validation
    if (document.getElementById('valEnabled').checked) {
        config['password_validation']['enabled'] = 'true';
        config['password_validation']['min_length'] = document.getElementById('valLength').value;
        config['password_validation']['upper'] = document.getElementById('valUpper').value;
        config['password_validation']['lower'] = document.getElementById('valLower').value;
        config['password_validation']['number'] = document.getElementById('valNumber').value;
        config['password_validation']['special'] = document.getElementById('valSpecial').value;
    } else {
        config['password_validation']['enabled'] = 'false';
    };
    // Page 9: Messages
    config['ui']['contact_message'] = document.getElementById('msgContact').value;
    config['ui']['help_message'] = document.getElementById('msgHelp').value;
    config['ui']['success_message'] = document.getElementById('msgSuccess').value;
    // Send it
    config["restart-program"] = true;
    var req = new XMLHttpRequest();
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
