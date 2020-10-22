import { _get, _post, _delete } from "../modules/common.js";

interface aWindow extends Window {
    setNotify(el: HTMLElement): void;
    deleteInvite(code: string): void;
}

declare var window: aWindow;

const emptyInvite = (): Invite => { return { code: "None", empty: true } as Invite; }

function genUsedBy(usedBy: Array<Array<string>>): string {
    let uB = "";
    if (usedBy && usedBy.length != 0) {
        uB = `
        <ul class="list-group list-group-flush">
            <li class="list-group-item py-1">Users created:</li>
        `;
        for (const i in usedBy) {
            uB += `
            <li class="list-group-item py-1 disabled">
                <div class="d-flex float-left">${usedBy[i][0]}</div>
                <div class="d-flex float-right">${usedBy[i][1]}</div>
            </li>
            `;
        }
        uB += `</ul>`
    }
    return uB;
}

function addItem(invite: Invite): void {
    const links = document.getElementById('invites');
    const container = document.createElement('div') as HTMLDivElement;
    container.id = invite.code;
    const item = document.createElement('div') as HTMLDivElement;
    item.classList.add('list-group-item', 'd-flex', 'justify-content-between', 'd-inline-block');
    let link  = "";
    let innerHTML = `<a>None</a>`;
    if (invite.empty) {
        item.innerHTML = `
        <div class="d-flex align-items-center font-monospace" style="width: 40%;">
            ${innerHTML}
        </div>
        `;
        container.appendChild(item);
        links.appendChild(container);
        return;
    }
    link = window.location.href.split('#')[0] + "invite/" + invite.code;
    innerHTML = `
    <div class="d-flex align-items-center font-monospace" style="width: 40%;">
        <a class="invite-link" href="${link}">${invite.code.replace(/-/g, '-')}</a>
        <i class="fa fa-clipboard icon-button" onclick="window.toClipboard('${link}')" style="margin-right: 0.5rem; margin-left: 0.5rem;"></i>
    `;
    if (invite.email) {
        let email = invite.email;
        if (!invite.email.includes("Failed to send to")) {
            email = `Sent to ${email}`;
        }
        innerHTML += `
        <span class="text-muted" style="margin-left: 0.4rem; font-style: italic; font-size: 0.8rem;">${email}</span>
        `;
    }
    innerHTML +=  `
    </div>
    <div style="text-align: right;">
        <span id="${invite.code}_expiry" style="margin-right: 1rem;">${invite.expiresIn}</span>
        <div style="display: inline-block;">
            <button class="btn btn-outline-danger" onclick="deleteInvite('${invite.code}')">Delete</button>
            <i class="fa fa-angle-down collapsed icon-button not-rotated" style="padding: 1rem; margin: -1rem -1rem -1rem 0;" data-toggle="collapse" aria-expanded="false" data-target="#${CSS.escape(invite.code)}_collapse" onclick="window.rotateButton(this)"></i>
        </div>
    </div>
    `;

    item.innerHTML = innerHTML;
    container.appendChild(item);
    
    let profiles = `
    <label class="input-group-text" for="profile_${CSS.escape(invite.code)}">Profile: </label>
    <select class="form-select" id="profile_${CSS.escape(invite.code)}" onchange="window.setProfile(this)">
        <option value="NoProfile" selected>No Profile</option>
    `;
    for (const i in window.availableProfiles) {
        let selected = "";
        if (window.availableProfiles[i] == invite.profile) {
            selected = "selected";
        }
        profiles += `<option value="${window.availableProfiles[i]}" ${selected}>${window.availableProfiles[i]}</option>`;
    }
    profiles += `</select>`;

    let dateCreated: string;
    if (invite.created) {
        dateCreated = `<li class="list-group-item py-1">Created: ${invite.created}</li>`;
    }

    let middle: string;
    if (window.notifications_enabled) {
        middle = `
        <div class="col" id="${CSS.escape(invite.code)}_notifyButtons">
            <ul class="list-group list-group-flush">
                Notify on:
                <li class="list-group-item py-1 form-check">
                    <input class="form-check-input" type="checkbox" value="" id="${CSS.escape(invite.code)}_notifyExpiry" onclick="setNotify(this)" ${invite.notifyExpiry ? "checked" : ""}>
                    <label class="form-check-label" for="${CSS.escape(invite.code)}_notifyExpiry">Expiry</label>
                </li>
                <li class="list-group-item py-1 form-check">
                    <input class="form-check-input" type="checkbox" value="" id="${CSS.escape(invite.code)}_notifyCreation" onclick="setNotify(this)" ${invite.notifyCreation ? "checked" : ""}>
                    <label class="form-check-label" for="${CSS.escape(invite.code)}_notifyCreation">User creation</label>
                </li>
            </ul>
        </div>  
        `;
    }

    let right: string = genUsedBy(invite.usedBy)

    const dropdown = document.createElement('div') as HTMLDivElement;
    dropdown.id = `${CSS.escape(invite.code)}_collapse`;
    dropdown.classList.add("collapse");
    dropdown.innerHTML = `
    <div class="container row align-items-start card-body">
        <div class="col">
            <ul class="list-group list-group-flush">
                <li class="input-group py-1">
                    ${profiles}
                </li>
                ${dateCreated}
                <li class="list-group-item py-1" id="${CSS.escape(invite.code)}_remainingUses">Remaining uses: ${invite.remainingUses}</li>
            </ul>
        </div>
        ${middle}
        <div class="col" id="${CSS.escape(invite.code)}_usersCreated">
            ${right}
        </div>
    </div>
    `;
    
    container.appendChild(dropdown);
    links.appendChild(container);
}

function parseInvite(invite: Object): Invite {
    let inv: Invite = { code: invite["code"], empty: false, };
    if (invite["email"]) {
        inv.email = invite["email"];
    }
    let time = ""
    const f = ["days", "hours", "minutes"];
    for (const i in f) {
        if (invite[f[i]] != 0) {
            time += `${invite[f[i]]}${f[i][0]} `;
        }
    }
    inv.expiresIn = `Expires in ${time.slice(0, -1)}`;
    if (invite["no-limit"]) {
        inv.remainingUses = "∞";
    } else if ("remaining-uses" in invite) {
        inv.remainingUses = invite["remaining-uses"];
    }
    if ("used-by" in invite) {
        inv.usedBy = invite["used-by"];
    }
    if ("created" in invite) {
        inv.created = invite["created"];
    }
    if ("notify-expiry" in invite) {
        inv.notifyExpiry = invite["notify-expiry"];
    }
    if ("notify-creation" in invite) {
        inv.notifyCreation = invite["notify-creation"];
    }
    if ("profile" in invite) {
        inv.profile = invite["profile"];
    }
    return inv;
}

window.setNotify = (el: HTMLElement): void => {
    let send = {};
    let code: string;
    let notifyType: string;
    if (el.id.includes("Expiry")) {
        code = el.id.replace("_notifyExpiry", "");
        notifyType = "notify-expiry";
    } else if (el.id.includes("Creation")) {
        code = el.id.replace("_notifyCreation", "");
        notifyType = "notify-creation";
    }
    send[code] = {};
    send[code][notifyType] = (el as HTMLInputElement).checked;
    _post("/invites/notify", send, function (): void {
        if (this.readyState == 4 && this.status != 200) {
            (el as HTMLInputElement).checked = !(el as HTMLInputElement).checked;
        }
    });
}

function updateInvite(invite: Invite): void {
    document.getElementById(invite.code + "_expiry").textContent = invite.expiresIn;
    const remainingUses: any = document.getElementById(CSS.escape(invite.code) + "_remainingUses");
    if (remainingUses) {
        remainingUses.textContent = `Remaining uses: ${invite.remainingUses}`;
    }
    document.getElementById(CSS.escape(invite.code) + "_usersCreated").innerHTML = genUsedBy(invite.usedBy);
}

// delete invite from DOM
const hideInvite = (code: string): void => document.getElementById(CSS.escape(code)).remove();

// delete invite from jfa-go
window.deleteInvite = (code: string): void => _delete("/invites", { "code": code }, function (): void {
    if (this.readyState == 4) {
        generateInvites();
    }
});

export function generateInvites(empty?: boolean): void {
    if (empty) {
        document.getElementById('invites').textContent = '';
        addItem(emptyInvite());
        return;
    }
    _get("/invites", null, function (): void {
        if (this.readyState == 4) {
            let data = this.response;
            window.availableProfiles = data['profiles'];
            const Profiles = document.getElementById('inviteProfile') as HTMLSelectElement;
            let innerHTML = "";
            for (let i = 0; i < window.availableProfiles.length; i++) {
                const profile = window.availableProfiles[i];
                innerHTML += `
                <option value="${profile}" ${(i == 0) ? "selected" : ""}>${profile}</option>
                `;
            }
            innerHTML += `
            <option value="NoProfile" ${(window.availableProfiles.length == 0) ? "selected" : ""}>No Profile</option>
            `;
            Profiles.innerHTML = innerHTML;
            if (data['invites'] == null || data['invites'].length == 0) {
                document.getElementById('invites').textContent = '';
                addItem(emptyInvite());
                return;
            }
            let items = document.getElementById('invites').children;
            for (const i in data['invites']) {
                let match = false;
                const inv = parseInvite(data['invites'][i]);
                for (const x in items) {
                    if (items[x].id == inv.code) {
                        match = true;
                        updateInvite(inv);
                        break;
                    }
                }
                if (!match) {
                    addItem(inv);
                }
            }
            // second pass to check for expired invites
            items = document.getElementById('invites').children;
            for (let i = 0; i < items.length; i++) {
                let exists = false;
                for (const x in data['invites']) {
                    if (items[i].id == data['invites'][x]['code']) {
                        exists = true;
                        break;
                    }
                }
                if (!exists) {
                    hideInvite(items[i].id);
                }
            }
        }
    });
}

export const addOptions = (length: number, el: HTMLSelectElement): void => {
    for (let v = 0; v <= length; v++) {
        const opt = document.createElement('option');
        opt.textContent = ""+v;
        opt.value = ""+v;
        el.appendChild(opt);
    }
    el.value = "0";
};

export function checkDuration(): void {
    const boxVals: Array<number> = [+(document.getElementById("days") as HTMLSelectElement).value, +(document.getElementById("hours") as HTMLSelectElement).value, +(document.getElementById("minutes") as HTMLSelectElement).value];
    const submit = document.getElementById('generateSubmit') as HTMLButtonElement;
    if (boxVals.reduce((a: number, b: number): number => a + b) == 0) {
        submit.disabled = true;
    } else {
        submit.disabled = false;
    }
}
