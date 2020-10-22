declare var bootstrap: any;

class Modal implements BSModal {
    el: HTMLDivElement;
    modal: any;

    constructor(id: string, find?: boolean) {
        this.el = document.getElementById(id) as HTMLDivElement;
        if (find) {
            this.modal = bootstrap.Modal.getInstance(this.el);
        } else {
            this.modal = new bootstrap.Modal(this.el);
        }
        this.el.addEventListener('shown.bs.modal', (): void => document.body.classList.add("modal-open"));
    };

    show(): void { this.modal.show(); };
    hide(): void { this.modal.hide(); };
}

export class BS5 implements Bootstrap {
    triggerTooltips: tooltipTrigger = function (): void {
        const checkboxes = [].slice.call(document.getElementById('settingsContent').querySelectorAll('input[type="checkbox"]'));
        for (const i in checkboxes) {
            checkboxes[i].click();
            checkboxes[i].click();
        }
        const tooltips = [].slice.call(document.querySelectorAll('a[data-toggle="tooltip"]'));
        tooltips.map((el: HTMLAnchorElement): any => {
            return new bootstrap.Tooltip(el);
        });
    };

    newModal: ModalConstructor = function (id: string, find?: boolean): BSModal {
        return new Modal(id, find);
    };
};
