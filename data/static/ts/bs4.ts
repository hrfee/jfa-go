var bsVersion = 4;

const send_to_addess_enabled = document.getElementById('send_to_addess_enabled');
if (send_to_addess_enabled) {
    send_to_addess_enabled.classList.remove("form-check-input");
}
const multiUseEnabled = document.getElementById('multiUseEnabled');
if (multiUseEnabled) {
    multiUseEnabled.classList.remove("form-check-input");
}

function createModal(id: string, find?: boolean): any {
    $(`#${id}`).on("shown.bs.modal", (): void => document.body.classList.add("modal-open"));
    return {
        show: function (): any {
            const temp = ($(`#${id}`) as any).modal("show");
            return temp;
        },
        hide: function (): any {
            return ($(`#${id}`) as any).modal("hide");
        }
    };
}

function triggerTooltips(): void {
    const checkboxes = [].slice.call(document.getElementById('settingsContent').querySelectorAll('input[type="checkbox"]'));
    for (const i in checkboxes) {
        checkboxes[i].click();
        checkboxes[i].click();
    }
    const tooltips = [].slice.call(document.querySelectorAll('a[data-toggle="tooltip"]'));
    tooltips.map((el: HTMLAnchorElement): any => {
        return ($(el) as any).tooltip();
    });
}

