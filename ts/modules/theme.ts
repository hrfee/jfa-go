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
