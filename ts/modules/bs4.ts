declare var $: any;

class Modal implements BSModal {
    el: HTMLDivElement;
    modal: any;

    constructor(id: string, find?: boolean) {
        this.el = document.getElementById(id) as HTMLDivElement;
        this.modal = $(this.el) as any;
        this.modal.on("shown.b.modal", (): void => document.body.classList.add('modal-open'));
    };

    show(): void { this.modal.modal("show"); };
    hide(): void { this.modal.modal("hide"); };
}

export class BS4 implements Bootstrap {
    triggerTooltips: tooltipTrigger = function (): void {
        const checkboxes = [].slice.call(document.getElementById('settingsContent').querySelectorAll('input[type="checkbox"]'));
        for (const i in checkboxes) {
            checkboxes[i].click();
            checkboxes[i].click();
        }
        const tooltips = [].slice.call(document.querySelectorAll('a[data-toggle="tooltip"]'));
        tooltips.map((el: HTMLAnchorElement): any => {
            return ($(el) as any).tooltip();
        });
    };

    Compat(): void {
        console.log('Fixing BS4 Compatability');
        const send_to_address_enabled = document.getElementById('send_to_address_enabled');
        if (send_to_address_enabled) {
            send_to_address_enabled.classList.remove("form-check-input");
        }
        const multiUseEnabled = document.getElementById('multiUseEnabled');
        if (multiUseEnabled) {
            multiUseEnabled.classList.remove("form-check-input");
        }
    }

    newModal: ModalConstructor = function (id: string, find?: boolean): BSModal {
        return new Modal(id, find);
    };
}
