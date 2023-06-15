declare var window: Window;

export class Modal implements Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement;
    openEvent: CustomEvent;
    closeEvent: CustomEvent;
    constructor(modal: HTMLElement, important: boolean = false) {
        this.modal = modal;
        this.openEvent = new CustomEvent("modal-open-" + modal.id);
        this.closeEvent = new CustomEvent("modal-close-" + modal.id);
        const closeButton = this.modal.querySelector('span.modal-close');
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
        this.modal.classList.add('animate-fade-out');
        this.modal.classList.remove("animate-fade-in");
        const modal = this.modal;
        const listenerFunc = () => {
            modal.classList.remove('block');
            modal.classList.remove('animate-fade-out');
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
        this.modal.classList.add('block', 'animate-fade-in');
        document.dispatchEvent(this.openEvent);
    }
    toggle = () => {
        if (this.modal.classList.contains('animate-fade-in')) {
            this.close();
        } else {
            this.show();
        }
    }

    asElement = () => { return this.modal; }
}
