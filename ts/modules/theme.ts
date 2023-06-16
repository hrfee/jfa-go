export class ThemeManager {

    private _themeButton: HTMLElement = null;
    
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
        const theme = localStorage.getItem("theme");
        if (theme == "dark") {
            this._enable(true);
        } else if (theme == "light") {
            this._enable(false);
        } else if (window.matchMedia('(prefers-color-scheme: dark)').media !== 'not all') {
            this._enable(true);
        }

        if (arguments.length == 1)
            this.bindButton(button);
    }

    private _toggle = () => {
        this._beforeTransition();
        if (!document.documentElement.classList.contains('dark')) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
        localStorage.setItem('theme', document.documentElement.classList.contains('dark') ? "dark" : "light");
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
    };

    enable = (dark: boolean) => {
        this._enable(dark);
        if (this._themeButton != null) {
            this._updateThemeIcon();
        }
    };
 }
