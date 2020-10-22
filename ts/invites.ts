import { serializeForm, rmAttr, addAttr, _get, _post, _delete } from "./modules/common.js";
import { generateInvites, checkDuration } from "./modules/invites.js";

interface aWindow extends Window {
    setProfile(el: HTMLElement): void;
}

declare var window: aWindow;

function fixCheckboxes(): void {
    const send_to_address: Array<HTMLInputElement> = [document.getElementById('send_to_address') as HTMLInputElement, document.getElementById('send_to_address_enabled') as HTMLInputElement];
    if (send_to_address[0] != null) {
        send_to_address[0].disabled = !send_to_address[1].checked;
    }
    const multiUseEnabled = document.getElementById('multiUseEnabled') as HTMLInputElement;
    const multiUseCount = document.getElementById('multiUseCount') as HTMLInputElement;
    const noUseLimit = document.getElementById('noUseLimit') as HTMLInputElement;
    multiUseCount.disabled = !multiUseEnabled.checked;
    noUseLimit.checked = false;
    noUseLimit.disabled = !multiUseEnabled.checked;
}

fixCheckboxes();

(document.getElementById('inviteForm') as HTMLFormElement).onsubmit = function (): boolean {
    const button = document.getElementById('generateSubmit') as HTMLButtonElement;
    button.disabled = true;
    button.innerHTML = `
    <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>
    Loading...`;
    let send = serializeForm('inviteForm');
    send["remaining-uses"] = +send["remaining-uses"];
    if (!send['multiple-uses'] || send['no-limit']) {
        delete send['remaining-uses'];
    }
    if (send["profile"] == "NoProfile") {
        send["profile"] = "";
    }
    const sendToAddress: any = document.getElementById('send_to_address');
    const sendToAddressEnabled: any = document.getElementById('send_to_address_enabled');
    if (sendToAddress && sendToAddressEnabled) {
        send['email'] = send['send_to_address'];
        delete send['send_to_address'];
        delete send['send_to_address_enabled'];
    }
    _post("/invites", send, function (): void {
        if (this.readyState == 4) {
            button.textContent = 'Generate';
            button.disabled = false;
            generateInvites();
        }
    });
    return false;
};

window.BS.triggerTooltips();

window.setProfile= (select: HTMLSelectElement): void => {
    if (!select.value) {
        return;
    }
    let val = select.value;
    if (select.value == "NoProfile") {
        val = ""
    }
    const invite = select.id.replace("profile_", "");
    const send = {
        "invite": invite,
        "profile": val
    };
    _post("/invites/profile", send, function (): void {
        if (this.readyState == 4 && this.status != 200) {
            generateInvites();
        }
    });
}

const nE: Array<string> = ["days", "hours", "minutes"];
for (const i in nE) {
    document.getElementById(nE[i]).addEventListener("change", checkDuration);
}
