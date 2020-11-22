declare interface ModalConstructor {
    (id: string, find?: boolean): BSModal;
}

declare interface BSModal {
    el: HTMLDivElement;
    modal: any;
    show: () => void;
    hide: () => void;
}

declare interface Window {
    getComputedStyle(element: HTMLElement, pseudoElt: HTMLElement): any;
    bsVersion: number;
    bs5: boolean;
    BS: Bootstrap;
    URLBase: string;
    Modals: BSModals;
    cssFile: string;
    availableProfiles: Array<any>;
    jfUsers: Array<Object>;
    notifications_enabled: boolean;
    token: string;
    buttonWidth: number;
}

declare interface tooltipTrigger {
    (): void;
}

declare interface Bootstrap {
    newModal: ModalConstructor;
    triggerTooltips: tooltipTrigger;
    Compat?(): void;
}

declare interface BSModals {
    login: BSModal;
    userDefaults: BSModal;
    users: BSModal;
    restart: BSModal;
    refresh: BSModal;
    about: BSModal;
    delete: BSModal;
    newUser: BSModal;
}

interface Invite {
    code?: string;
    expiresIn?: string;
    empty: boolean;
    remainingUses?: string;
    email?: string;
    usedBy?: Array<Array<string>>;
    created?: string;
    notifyExpiry?: boolean;
    notifyCreation?: boolean;
    profile?: string;
}

declare var config: Object;
declare var modifiedConfig: Object;
