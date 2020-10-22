import { _get, _post, _delete, rmAttr, addAttr } from "modules/common.js";

const ombiDefaultsModal = window.BS.newModal('ombiDefaults');

(document.getElementById('openOmbiDefaults') as HTMLButtonElement).onclick = function (): void {
    let button = this as HTMLButtonElement;
    button.disabled = true;
    const ogHTML = button.innerHTML;
    button.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    _get("/ombi/users", null, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200) {
                const users = this.response['users'];
                const radioList = document.getElementById('ombiUserRadios');
                radioList.textContent = '';
                let first = true;
                for (const i in users) {
                    const user = users[i];
                    const radio = document.createElement('div') as HTMLDivElement;
                    radio.classList.add('form-check');
                    let checked = '';
                    if (first) {
                        checked = 'checked';
                        first = false;
                    }
                    radio.innerHTML = `
                    <input class="form-check-input" type="radio" name="ombiRadios" id="ombiDefault_${user['id']}" ${checked}>
                    <label class="form-check-label" for="ombiDefault_${user['id']}">${user['name']}</label>
                    `;
                    radioList.appendChild(radio);
                }
                button.disabled = false;
                button.innerHTML = ogHTML;
                const submitButton = document.getElementById('storeOmbiDefaults') as HTMLButtonElement;
                submitButton.disabled = false;
                submitButton.textContent = 'Submit';
                addAttr(submitButton,  "btn-primary");
                rmAttr(submitButton, "btn-success");
                rmAttr(submitButton, "btn-danger");
                ombiDefaultsModal.show();
            }
        }
    });
};

(document.getElementById('storeOmbiDefaults') as HTMLButtonElement).onclick = function (): void {
    let button = this as HTMLButtonElement;
    button.disabled = true;
    button.innerHTML =
        '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" style="margin-right: 0.5rem;"></span>' +
        'Loading...';
    const radio = document.querySelector('input[name=ombiRadios]:checked') as HTMLInputElement;
    const data = {
        "id": radio.id.replace("ombiDefault_", "")
    };
    _post("/ombi/defaults", data, function (): void {
        if (this.readyState == 4) {
            if (this.status == 200 || this.status == 204) {
                button.textContent = "Success";
                addAttr(button, "btn-success");
                rmAttr(button, "btn-danger");
                rmAttr(button, "btn-primary");
                button.disabled = false;
                setTimeout((): void => ombiDefaultsModal.hide(), 1000);
            } else {
                button.textContent = "Failed";
                rmAttr(button, "btn-primary");
                addAttr(button, "btn-danger");
                setTimeout((): void => {
                    button.textContent = "Submit";
                    addAttr(button, "btn-primary");
                    rmAttr(button, "btn-danger");
                    button.disabled = false;
                }, 1000);
            }
        }
    });
};




