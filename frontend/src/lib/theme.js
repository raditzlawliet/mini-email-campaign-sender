// When app.theme is not set yet, the theme follows the OS colour scheme:
// Traveler Gilded Night when dark, Day when light, switching live when the
// OS setting changes. Once a theme is chosen it is always used as is.
const darkQuery = window.matchMedia("(prefers-color-scheme: dark)");

let selected = "";

export function resolveTheme(theme) {
    if (theme) return theme;
    return darkQuery.matches ? "traveler-gilded-night" : "traveler-gilded-day";
}

export function applyTheme(theme) {
    selected = theme || "";
    document.documentElement.dataset.theme = resolveTheme(selected);
}

darkQuery.addEventListener("change", () => {
    if (!selected) applyTheme(selected);
});
