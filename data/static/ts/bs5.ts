declare var bootstrap: any;

var bsVersion = 5;

function createModal(id: string, find?: boolean): any {
    let modal: any;
    if (find) {
        modal = bootstrap.Modal.getInstance(document.getElementById(id));
    } else {
        modal = new bootstrap.Modal(document.getElementById(id));
    }
    document.getElementById(id).addEventListener('shown.bs.modal', (): void => document.body.classList.add("modal-open"));
    return {
        modal: modal,
        show: function (): any {
            const temp = this.modal.show();
            return temp;
        },
        hide: function (): any { return this.modal.hide(); }
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
        return new bootstrap.Tooltip(el);
    });
}

