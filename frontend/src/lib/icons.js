// Pixel icons for the Traveler Gilded themes: 12 x 12 one-bit bitmaps, drawn
// at 12, 24 or 48px so one cell is a whole number of pixels.
//
// DS_ICONS come from the "Traveler Gilded Pixel" design system (its Icon
// component), one row per three hex digits, top to bottom.
const DS_ICONS = {
    "chevron-up": "0000000000600f01f839c70e606000000000",
    "chevron-down": "00000000060670e39c1f80f0060000000000",
    "chevron-left": "0000180380700e01c01c00e0070038018000",
    "chevron-right": "0001801c00e00700380380700e01c0180000",
    "close": "00060670e39c1f80f00f01f839c70e606000",
    "mail": "000000fffc03b0d8f1801801801fff000000",
    "minimize": "0000000000000000000000007fe7fe000000",
    "maximize": "0007fe7fe6066066066066066067fe000000",
    "info": "0f03fc79e79effff9ff9ff9f79e7fe3fc0f0",
    "warning": "0000600f009019819839c3fc79e7fefff000",
};

// Icons the design system does not have, drawn on the same grid and in the
// same manner: one colour, strokes two cells wide, solid shapes with cells
// knocked out for detail. "#" is ink, "." is empty.
const APP_ICONS = {
    play: [
        "............",
        "..##........",
        "..####......",
        "..######....",
        "..########..",
        "..#########.",
        "..#########.",
        "..########..",
        "..######....",
        "..####......",
        "..##........",
        "............",
    ],
    pause: [
        "............",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "..###..###..",
        "............",
    ],
    globe: [
        "....####....",
        "..########..",
        ".###.##.###.",
        ".##..##..##.",
        "##...##...##",
        "############",
        "############",
        "##...##...##",
        ".##..##..##.",
        ".###.##.###.",
        "..########..",
        "....####....",
    ],
    bot: [
        "............",
        ".....##.....",
        ".....##.....",
        ".##########.",
        ".##########.",
        ".##..##..##.",
        "###..##..###",
        "############",
        ".###....###.",
        ".##########.",
        ".##########.",
        "............",
    ],
    github: [
        "....####....",
        "..########..",
        ".##.####.##.",
        ".##......##.",
        "###......###",
        "###......###",
        "###......###",
        "####....####",
        ".###....###.",
        ".###....###.",
        "..##....##..",
        "............",
    ],
    restore: [
        "............",
        "...########.",
        "...########.",
        "...##....##.",
        ".##########.",
        ".##########.",
        ".##....####.",
        ".##....####.",
        ".##....##...",
        ".########...",
        ".########...",
        "............",
    ],
    dot: [
        "............",
        "............",
        "............",
        "....####....",
        "...######...",
        "...######...",
        "...######...",
        "...######...",
        "....####....",
        "............",
        "............",
        "............",
    ],
};

function rowsFromHex(hex) {
    const rows = [];
    for (let y = 0; y < 12; y++) {
        const bits = parseInt(hex.substr(y * 3, 3), 16);
        let row = "";
        for (let x = 0; x < 12; x++) row += bits & (1 << (11 - x)) ? "#" : ".";
        rows.push(row);
    }
    return rows;
}

// One rect per horizontal run of ink, as the design system draws them.
function rectsFromRows(rows) {
    const rects = [];
    rows.forEach((row, y) => {
        for (let x = 0; x < 12; ) {
            if (row[x] !== "#") {
                x++;
                continue;
            }
            const start = x;
            while (x < 12 && row[x] === "#") x++;
            rects.push({ x: start, y, width: x - start });
        }
    });
    return rects;
}

const cache = new Map();

export function iconRects(name) {
    if (!cache.has(name)) {
        const rows = APP_ICONS[name] ?? (DS_ICONS[name] && rowsFromHex(DS_ICONS[name]));
        cache.set(name, rows ? rectsFromRows(rows) : []);
    }
    return cache.get(name);
}
