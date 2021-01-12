declare interface Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement
    show: () => void;
    close: (event?: Event) => void;
    toggle: () => void;
}

interface ArrayConstructor {
    from(arrayLike: any, mapFn?, thisArg?): Array<any>;
}

declare interface Window {
    URLBase: string;
    modals: Modals;
    cssFile: string;
    availableProfiles: string[];
    jfUsers: Array<Object>;
    notificationsEnabled: boolean;
    emailEnabled: boolean;
    ombiEnabled: boolean;
    usernameEnabled: boolean;
    token: string;
    buttonWidth: number;
    transitionEvent: string;
    animationEvent: string;
    tabs: Tabs;
    invites: inviteList;
    notifications: NotificationBox;
    language: string;
    lang: Lang;
    langFile: {};
}

declare interface Lang {
    get: (sect: string, key: string) => string;
    strings: (key: string) => string;
    notif: (key: string) => string;
    var: (sect: string, key: string, ...subs: string[]) => string;
    quantity: (key: string, number: number) => string;
}

declare interface NotificationBox {
    connectionError: () => void;
    customError: (type: string, message: string) => void;
    customPositive: (type: string, bold: string,  message: string) => void;
    customSuccess: (type: string, message: string) => void;
}

declare interface Tabs {
    current: string;
    tabs: Array<Tab>;
    addTab: (tabID: string, preFunc?: () => void, postFunc?: () => void) => void;
    switch: (tabID: string, noRun?: boolean) => void;
}

declare interface Tab {
    tabID: string;
    tabEl: HTMLDivElement;
    buttonEl: HTMLSpanElement;
    preFunc?: () => void;
    postFunc?: () => void;
}


declare interface Modals {
    about: Modal;
    login: Modal;
    addUser: Modal;
    modifyUser: Modal;
    deleteUser: Modal;
    settingsRestart: Modal;
    settingsRefresh: Modal;
    ombiDefaults?: Modal;
    profiles: Modal;
    addProfile: Modal;
}

interface Invite {
    code?: string;
    expiresIn?: string;
    remainingUses?: string;
    email?: string;
    usedBy?: string[][];
    created?: string;
    notifyExpiry?: boolean;
    notifyCreation?: boolean;
    profile?: string;
}

interface inviteList {
    empty: boolean;
    invites: { [code: string]: Invite }
    add: (invite: Invite) => void;
    reload: () => void;
}

declare interface SubmitEvent extends Event {
    submitter: HTMLInputElement;
}

declare var config: Object;
declare var modifiedConfig: Object;
