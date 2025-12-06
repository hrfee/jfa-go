export class ThemeManager {

    private _themeButton: HTMLElement = null;
    private _metaTag: HTMLMetaElement;
       
    private _cssLightFiles: HTMLLinkElement[];
    private _cssDarkFiles: HTMLLinkElement[];

    
    private _beforeTransition = () => {
        const doc = document.documentElement;
        const onTransitionDone = () => {
            doc.classList.remove('nightwind');
            doc.removeEventListener('transitionend', onTransitionDone);
        }
        doc.addEventListener('transitionend', onTransitionDone);
        if (!doc.classList.contains('nightwind')) {
            doc.classList.add('nightwind');
        }
    };

    private _updateThemeIcon = () => {
        const icon = this._themeButton.childNodes[0] as HTMLElement;
        if (document.documentElement.classList.contains("dark")) {
            icon.classList.add("ri-sun-line");
            icon.classList.remove("ri-moon-line");
            this._themeButton.classList.add("~warning");
            this._themeButton.classList.remove("~neutral");
            this._themeButton.classList.remove("@high");
        } else {
            icon.classList.add("ri-moon-line");
            icon.classList.remove("ri-sun-line");
            this._themeButton.classList.add("@high");
            this._themeButton.classList.add("~neutral");
            this._themeButton.classList.remove("~warning");
        }
    };

    bindButton = (button: HTMLElement) => {
        this._themeButton = button;
        this._themeButton.onclick = this.toggle;
        this._updateThemeIcon();
    }

    toggle = () => {
        this._toggle();
        if (this._themeButton) {
            this._updateThemeIcon();
        }
    }

    constructor(button?: HTMLElement) {
        this._metaTag = document.querySelector("meta[name=color-scheme]") as HTMLMetaElement;
    
        this._cssLightFiles = Array.from(document.head.querySelectorAll("link[data-theme=light]")) as Array<HTMLLinkElement>;
        this._cssDarkFiles = Array.from(document.head.querySelectorAll("link[data-theme=dark]")) as Array<HTMLLinkElement>;
        this._cssLightFiles.forEach((el) => el.remove());
        this._cssDarkFiles.forEach((el) => el.remove());
        const theme = localStorage.getItem("theme");
        if (theme == "dark") {
            this._enable(true);
        } else if (theme == "light") {
            this._enable(false);
        } else if (window.matchMedia('(prefers-color-scheme: dark)').media !== 'not all') {
            this._enable(true);
        }

        if (button)
            this.bindButton(button);
    }

    private _toggle = () => {
        let metaValue = "light dark";
        this._beforeTransition();
        const dark = !document.documentElement.classList.contains("dark");
        if (dark) {
            document.documentElement.classList.add('dark');
            metaValue = "dark light";
            this._cssLightFiles.forEach((el) => el.remove());
            this._cssDarkFiles.forEach((el) => document.head.appendChild(el));
        } else {
            document.documentElement.classList.remove('dark');
            this._cssDarkFiles.forEach((el) => el.remove());
            this._cssLightFiles.forEach((el) => document.head.appendChild(el));
        }
        localStorage.setItem('theme', document.documentElement.classList.contains('dark') ? "dark" : "light");
        
        // this._metaTag.setAttribute("content", metaValue);
    };

    private _enable = (dark: boolean) => {
        const mode = dark ? "dark" : "light";
        const opposite = dark ? "light" : "dark";
        
        localStorage.setItem('theme', dark ? "dark" : "light");

        this._beforeTransition();

        if (document.documentElement.classList.contains(opposite)) {
            document.documentElement.classList.remove(opposite);
        }
        document.documentElement.classList.add(mode);
       
        if (dark) {
            this._cssLightFiles.forEach((el) => el.remove());
            this._cssDarkFiles.forEach((el) => document.head.appendChild(el));
        } else {
            this._cssDarkFiles.forEach((el) => el.remove());
            this._cssLightFiles.forEach((el) => document.head.appendChild(el));
        }
        // this._metaTag.setAttribute("content", `${mode} ${opposite}`);
    };

    enable = (dark: boolean) => {
        this._enable(dark);
        if (this._themeButton != null) {
            this._updateThemeIcon();
        }
    };
 }
