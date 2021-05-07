declare var window: Window;

export class Modal implements Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement;
    openEvent: CustomEvent;
    closeEvent: CustomEvent;
    constructor(modal: HTMLElement, important: boolean = false) {
        this.modal = modal;
        this.openEvent = new CustomEvent("modal-open-" + modal.id)
        this.closeEvent = new CustomEvent("modal-close-" + modal.id)
        const closeButton = this.modal.querySelector('span.modal-close')
        if (closeButton !== null) {
            this.closeButton = closeButton as HTMLSpanElement;
            this.closeButton.onclick = this.close;
        }
        if (!important) {
            window.addEventListener('click', (event: Event) => {
                if (event.target == this.modal) { this.close(); }
            });
        }
    }
    close = (event?: Event) => {
        if (event) {
            event.preventDefault();
        }
        this.modal.classList.add('modal-hiding');
        const modal = this.modal;
        const listenerFunc = function () {
            modal.classList.remove('modal-shown');
            modal.classList.remove('modal-hiding');
            modal.removeEventListener(window.animationEvent, listenerFunc);
            document.dispatchEvent(this.closeEvent);
        };
        this.modal.addEventListener(window.animationEvent, listenerFunc, false);
    }

    set onopen(f: () => void) {
        document.addEventListener("modal-open-"+this.modal.id, f);
    }
    set onclose(f: () => void) {
        document.addEventListener("modal-close-"+this.modal.id, f);
    }

    show = () => {
        this.modal.classList.add('modal-shown');
        document.dispatchEvent(this.openEvent);
    }
    toggle = () => {
        if (this.modal.classList.contains('modal-shown')) {
            this.close();
        } else {
            this.show();
        }
    }
}
