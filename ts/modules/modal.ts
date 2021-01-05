declare var window: Window;

export class Modal implements Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement;
    constructor(modal: HTMLElement, important: boolean = false) {
        this.modal = modal;
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
        };
        this.modal.addEventListener(window.animationEvent, listenerFunc, false);
    }
    show = () => {
        this.modal.classList.add('modal-shown');
    }
    toggle = () => {
        if (this.modal.classList.contains('modal-shown')) {
            this.close();
        } else {
            this.show();
        }
    }
}
