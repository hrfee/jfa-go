export function toggleTheme() {
    document.documentElement.classList.toggle('dark');
    document.documentElement.classList.toggle('light');
    localStorage.setItem('theme', document.documentElement.classList.contains('dark') ? "dark" : "light");
}

export function loadTheme() {
    const theme = localStorage.getItem("theme");
    if (theme == "dark") {
        document.documentElement.classList.add('dark');
        document.documentElement.classList.remove('light');
    } else if (theme == "light") {
        document.documentElement.classList.add('light');
        document.documentElement.classList.remove('dark');
    } else if (window.matchMedia('(prefers-color-scheme: dark)').media !== 'not all') {
        document.documentElement.classList.add('dark');
        document.documentElement.classList.remove('light');
    }
}

export class nightwind {
    beforeTransition = () => {
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
    constructor() {
        const theme = localStorage.getItem("theme");
        if (theme == "dark") {
            this.enable(true);
        } else if (theme == "light") {
            this.enable(false);
        } else if (window.matchMedia('(prefers-color-scheme: dark)').media !== 'not all') {
            this.enable(true);
        }
    }

    toggle = () => {
        this.beforeTransition();
        if (!document.documentElement.classList.contains('dark')) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
        localStorage.setItem('theme', document.documentElement.classList.contains('dark') ? "dark" : "light");
    };

    enable = (dark: boolean) => {
        const mode = dark ? "dark" : "light";
        const opposite = dark ? "light" : "dark";
        
        localStorage.setItem('theme', dark ? "dark" : "light");

        this.beforeTransition();

        if (document.documentElement.classList.contains(opposite)) {
            document.documentElement.classList.remove(opposite);
        }
        document.documentElement.classList.add(mode);
    };
 }
