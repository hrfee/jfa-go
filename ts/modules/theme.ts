export function toggleTheme() {
    document.documentElement.classList.toggle('dark-theme');
    document.documentElement.classList.toggle('light-theme');
    localStorage.setItem('theme', document.documentElement.classList.contains('dark-theme') ? "dark" : "light");
}

export function loadTheme() {
    const theme = localStorage.getItem("theme");
    if (theme == "dark") {
        document.documentElement.classList.add('dark-theme');
        document.documentElement.classList.remove('light-theme');
    } else if (theme == "light") {
        document.documentElement.classList.add('light-theme');
        document.documentElement.classList.remove('dark-theme');
    }
}
