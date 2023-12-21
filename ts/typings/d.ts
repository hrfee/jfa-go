declare interface Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement
    show: () => void;
    close: (event?: Event) => void;
    toggle: () => void;
    onopen: (f: () => void) => void;
    onclose: (f: () => void) => void;
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
    telegramEnabled: boolean;
    discordEnabled: boolean;
    matrixEnabled: boolean;
    ombiEnabled: boolean;
    usernameEnabled: boolean;
    linkResetEnabled: boolean;
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
    updater: updater;
    jellyfinLogin: boolean;
    jfAdminOnly: boolean;
    jfAllowAll: boolean;
    referralsEnabled: boolean;
    loginAppearance: string; 
}

declare interface Update {
	version: string; 
    commit: string;    
	date: number;
    description: string;
    changelog: string;
    link: string;
    download_link?: string;
    can_update: boolean;
}

declare interface updater extends Update {
    checkForUpdates: (run?: (req: XMLHttpRequest) => void) => void;
    updateAvailable: boolean;
    update: Update;
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
    switch: (tabID: string, noRun?: boolean, keepURL?: boolean) => void;
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
    ombiProfile?: Modal;
    profiles: Modal;
    addProfile: Modal;
    announce: Modal;
    editor: Modal;
    customizeEmails: Modal;
    extendExpiry: Modal;
    updateInfo: Modal;
    telegram: Modal;
    discord: Modal;
    matrix: Modal;
    sendPWR?: Modal;
    pwr?: Modal;
    logs: Modal;
    email?: Modal;
    enableReferralsUser?: Modal;
    enableReferralsProfile?: Modal;
    backedUp?: Modal;
    backups?: Modal;
}

interface Invite {
    code?: string;
    expiresIn?: string;
    remainingUses?: string;
    send_to?: string;
    usedBy?: { [name: string]: number };
    created?: number;
    notifyExpiry?: boolean;
    notifyCreation?: boolean;
    profile?: string;
    label?: string;
    user_label?: string;
    userExpiry?: boolean;
    userExpiryTime?: string;
}

interface inviteList {
    empty: boolean;
    invites: { [code: string]: Invite }
    add: (invite: Invite) => void;
    reload: (callback?: () => void) => void;
    isInviteURL: () => boolean;
    loadInviteURL: () => void;
}

// Finally added to typescript, dont need this anymore.
// declare interface SubmitEvent extends Event {
//     submitter: HTMLInputElement;
// }

declare var config: Object;
declare var modifiedConfig: Object;
