declare interface Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement
    show: () => void;
    close: (event?: Event, noDispatch?: boolean) => void;
    toggle: () => void;
    onopen: (f: () => void) => void;
    onclose: (f: () => void) => void;
}

interface ArrayConstructor {
    from(arrayLike: any, mapFn?, thisArg?): Array<any>;
}

declare interface PagePaths {
    // The base subfolder the app is being accessed from.
    Base: string;
    // The base subfolder the app is meant to be accessed from ("Reverse proxy subfolder")
    TrueBase: string;
    // The subdirectory this bit of the app is hosted on (e.g. admin is usually on "/", myacc is usually on "/my/account")
    Current: string;
    // Those for other pages
    Admin: string;
    MyAccount: string;
    Form: string;
    // The "External jfa-go URL"
    ExternalURI: string;
}

declare interface GlobalWindow extends Window {
    pages: PagePaths;
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
    jellyseerrEnabled: boolean;
    pwrEnabled: boolean;
    usernameEnabled: boolean;
    linkResetEnabled: boolean;
    token: string;
    buttonWidth: number;
    transitionEvent: string;
    animationEvent: string;
    tabs: Tabs;
    invites: InviteList;
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

declare interface InviteList {
    empty: boolean;
    invites: { [code: string]: Invite }
    add: (invite: Invite) => void;
    reload: (callback?: () => void) => void;
    isInviteURL: () => boolean;
    loadInviteURL: () => void;
}

declare interface Invite {
    code?: string;
    expiresIn?: string;
    remainingUses?: string;
    send_to?: string; // DEPRECATED: use sent_to instead.
    sent_to?: SentToList;
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

declare interface SendFailure {
    address: string;
    reason: "CheckLogs" | "NoUser" | "MultiUser" | "InvalidAddress";
}

declare interface SentToList {
    success: string[];
    failed: SendFailure[];
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
    template: (sect: string, key: string, subs: { [key: string]: string }) => string;
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
    addTab: (tabID: string, url: string, preFunc?: () => void, postFunc?: () => void, unloadFunc?: () => void) => void;
    switch: (tabID: string, noRun?: boolean, keepURL?: boolean) => void;
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
    jellyseerrProfile?: Modal;
    profiles: Modal;
    addProfile: Modal;
    editProfile: Modal;
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
    tasks: Modal;
    email?: Modal;
    enableReferralsUser?: Modal;
    enableReferralsProfile?: Modal;
    backedUp?: Modal;
    backups?: Modal;
}

interface paginatedDTO {
    last_page: boolean;
}

interface PaginatedReqDTO {
    limit: number;
    page: number;
    sortByField: string;
    ascending: boolean;
};

interface DateAttempt {
    year?: number;
    month?: number;
    day?: number;
    hour?: number;
    minute?: number;
    offsetMinutesFromUTC?: number;
}

interface ParsedDate {
    attempt: DateAttempt;
    date: Date;
    text: string;
    invalid?: boolean;
};

declare var config: Object;
declare var modifiedConfig: Object;
