import { rmAttr, addAttr, _post, _get, _delete, createEl } from "../modules/common.js";

export const Focus = (el: HTMLElement): void => rmAttr(el, 'unfocused');
export const Unfocus = (el: HTMLElement): void => addAttr(el, 'unfocused');

export function storeDefaults(users: string | Array<string>): void {
    const button = document.getElementById('storeDefaults') as HTMLButtonElement;
    button.disabled = true;
    button.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
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
    _post("/users/settings", data, function (): void {
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
                    window.Modals.userDefaults.hide();
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
