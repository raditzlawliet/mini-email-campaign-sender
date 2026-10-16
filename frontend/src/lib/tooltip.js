// Svelte action: `use:tooltip={text}` shows a themed tooltip on hover and
// keyboard focus, in place of the native `title` bubble (which the OS draws
// and no theme can style). The bubble is mounted on <body> with fixed
// positioning, so overflow and the title bar never clip it. It sits below
// the element when there is no room above, and stays inside the window.
const GAP = 8;
const MARGIN = 4;
const SHOW_DELAY = 300;

let nextId = 0;

export function tooltip(node, text) {
    let bubble = null;
    let timer = null;
    const id = `app-tooltip-${++nextId}`;

    function show() {
        clearTimeout(timer);
        if (!text || bubble) return;
        bubble = document.createElement("div");
        bubble.id = id;
        bubble.className = "app-tooltip";
        bubble.setAttribute("role", "tooltip");
        bubble.textContent = text;
        document.body.appendChild(bubble);
        node.setAttribute("aria-describedby", id);
        place();
    }

    function place() {
        const r = node.getBoundingClientRect();
        const b = bubble.getBoundingClientRect();
        const below = r.top - b.height - GAP < MARGIN;
        const top = below ? r.bottom + GAP : r.top - b.height - GAP;
        const centre = r.left + r.width / 2;
        const left = Math.min(
            Math.max(centre - b.width / 2, MARGIN),
            window.innerWidth - b.width - MARGIN,
        );
        bubble.dataset.placement = below ? "bottom" : "top";
        bubble.style.top = `${Math.round(top)}px`;
        bubble.style.left = `${Math.round(left)}px`;
        // The arrow points at the element even when the bubble is clamped.
        bubble.style.setProperty("--arrow-x", `${Math.round(centre - left)}px`);
    }

    function hide() {
        clearTimeout(timer);
        if (!bubble) return;
        bubble.remove();
        bubble = null;
        node.removeAttribute("aria-describedby");
    }

    function onEnter() {
        clearTimeout(timer);
        timer = setTimeout(show, SHOW_DELAY);
    }

    function onFocus() {
        if (node.matches(":focus-visible")) show();
    }

    function onKey(e) {
        if (e.key === "Escape") hide();
    }

    node.addEventListener("mouseenter", onEnter);
    node.addEventListener("mouseleave", hide);
    node.addEventListener("focus", onFocus);
    node.addEventListener("blur", hide);
    node.addEventListener("click", hide);
    node.addEventListener("keydown", onKey);

    return {
        update(newText) {
            text = newText;
            if (!bubble) return;
            if (!text) return hide();
            bubble.textContent = text;
            place();
        },
        destroy() {
            hide();
            node.removeEventListener("mouseenter", onEnter);
            node.removeEventListener("mouseleave", hide);
            node.removeEventListener("focus", onFocus);
            node.removeEventListener("blur", hide);
            node.removeEventListener("click", hide);
            node.removeEventListener("keydown", onKey);
        },
    };
}
