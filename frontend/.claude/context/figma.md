# The Figma reference

`https://www.figma.com/design/z6T2ZnlTduFtA49DvH7uRi/Movie-Ticket-Booking-Website--Community-`

The owner's chosen visual reference. **It is the visual language, not the information architecture** — the menu
and the screen list mirror the backend, not this file. See `context/decisions.md` and `../CLAUDE.local.md`.

> **Stale on colour since 2026-09-19. Read this first.**
> Every "brand green" below refers to this file's `#1DE782`, which is **no longer the brand**: the owner moved
> the brand to red `#E4002B` (CinePlex's), and the seat palette was replaced outright — free is grey, selecting
> is **blue**, held amber, sold red. The dark backdrop was re-derived from the red too, so it is no longer
> green-tinted. `.claude/rules/design-tokens.md` is the current source; it wins over this file on every colour.
> What remains useful here is the **layout and composition** of each frame — where things sit, what a screen is
> made of, and which frames have no backend behind them. Read it for structure, not for hex values.

## How to read it

It is not fetchable. The page is an app shell and the design lives on a WebGL canvas, so WebFetch returns
nothing and `api.figma.com` answers 403 without a personal token. It **is** readable through the
chrome-devtools MCP, and only this sequence is reliable:

1. Open the file URL, then click **"Expand UI"** in the left sidebar to get the Layers panel.
2. Click a layer row. The click updates the address bar to that frame's `node-id`.
3. **Navigate to `...?node-id=<id>` as a fresh page load.** Figma then centres and zooms to that frame (~72%).
   Zoom shortcuts, `Shift+2` and synthetic wheel events do **not** work on the canvas. The URL does.
4. Screenshot to a file, then read the exact colours by sampling the dominant colour of a pixel region —
   `dev-local/sample-png.py` does this with no dependencies. **Do not eyeball hex values.**

Frames are **1440×1024**. 24 frames across 5 pages: Presentation, User Flow, Wireframing, UI Design, Component.
Everything below is on **UI Design**.

## Frame inventory

| Frame                    | node-id    | Built?                  |
| ------------------------ | ---------- | ----------------------- |
| Sign In (customer)       | `77-626`   | no                      |
| Login (customer)         | —          | no                      |
| Home                     | `77-855`   | no                      |
| Home2                    | `108-510`  | no                      |
| Select Seat              | `108-705`  | no                      |
| Order detail             | `122-1018` | no                      |
| Payment Information      | `122-1090` | no                      |
| Payment Success          | `122-1100` | no                      |
| My ticket                | `240-380`  | no                      |
| Admin Movie              | `122-1302` | **yes** — `MoviesPage`  |
| Admin Theater            | `122-1389` | **yes** — `HallsPage`   |
| Admin Dashboard (Report) | `122-1620` | **yes** — `ReportsPage` |

The Figma has **no Showtimes screen and no dashboard tiles**, but the backend has both and this app ships both.
It also has Theaters / Users / Report, which this app does not have yet.

## Per-screen specs

Only the frames actually opened and measured are described. Open the rest with the method above before building
them, and extend this file as you go.

### Admin Movie — `122-1302`

The template for every admin list screen.

- White top bar, right-aligned: a circular brand-green avatar with the account's initial, then the account name.
- White sider, plain text nav, no icons in the design. The selected item is a `brand.soft` pill.
- Content: a **"+ Create new"** button at the top right, then a bordered table.
- Columns: `#`, **Cover** (poster thumbnail), Title, Status, Action.
- Action column: three icon buttons — view (grey outline), edit (grey outline), delete (**filled red**).

What this codebase does differently, on purpose: it keeps antd's `Table` (paging, search, filters, sorting),
which the design's plain table has none of, and it drops the `#` column because the table is paged.

### Admin Theater — `122-1389`

Two stacked tables on one page, no tabs: **Theater** (`#`, Name, Status, Action) and, under it, **Seats**
(`#`, Name, Type, Seats, Action). The Seats table is a placeholder — a list of hall names, not a seat map.

**This frame contradicts Admin Movie on the shell**, measured from its own pixels: the selected sider item is
grey `#DFDFDF`, not `brand.soft`, and the avatar is dark `#211A26`, not brand green. Admin Movie is the one
this codebase follows; do not "fix" the app to match this frame. The design file is simply inconsistent here.

What this codebase does differently, on purpose: it takes the two-section idea (hall identity, then seats) but
the seats section becomes a **real seat grid** on its own route (`/halls/:id/seats`), because the backend has a
per-seat grid with types, gaps and 2-column seats that a flat list cannot express. Paging, search and the
price editor have no counterpart in the design at all.

### Admin Dashboard / Report — `122-1620`

Same shell. The content is a single bordered table: `#`, Movie name, Ticket sales, Total sales. Nothing else —
no tiles, no chart, no date range.

`ReportsPage` keeps that table as its first section ("Theo phim"), but the backend forces three additions the
design has no place for: the endpoint is **range-based**, so a date range is mandatory; it aggregates by **day**
and not by movie, so the per-movie table has to be rolled up client-side from the `breakdown.showtimes` jsonb;
and a day the `closeDay` job has not closed is **absent rather than zero**, which the screen has to say out loud
or every gap reads as "no revenue".

### Home — `77-855`

- Full-bleed `cinemaGradient`; the glow sits toward the left.
- Top bar: "Cinemas" logo left; **Login** (filled brand) and **Register** (outlined) right.
- A centred **"Now Showing"** heading.
- A poster grid, 4 per row, each poster with the title centred underneath. Posters are 2:3 with rounded corners.

### Select Seat — `108-705`

- Same gradient. A left-aligned **"Seat"** heading.
- The seat grid is centred: white rounded squares carrying the seat label, selected seats in brand green.
- Below the grid, a **white full-width bar marked "X"** stands for the screen.
- A fixed bottom bar on the brighter part of the gradient: **TOTAL** and its amount, **SEAT** and the selected
  labels, then **Back** (outlined) and **Proceed Payment** (filled brand).

Only two seat states are drawn. Held, sold and gap are this codebase's own — see `rules/design-tokens.md`.

### Order detail — `122-1018`

- Same gradient, content centred in a single column with **no card surface** — text sits straight on the
  background.
- "Booking Detail" title, then a "Schedule" section of label/value pairs (label muted, value bright white).
- A "Transaction Detail" section: line items right-aligned with quantities, then a thin divider, then
  **Total payment**.
- Fine print, then a full-width **Checkout Ticket** button in brand green.

## What the Figma does not answer

Derive these, and say so where you do:

- Loading, empty and error states. None are drawn anywhere.
- Warning and info colours. Only brand green and red exist.
- Seat states beyond available/selected, and the four seat TYPES (standard/vip/couple/recliner) — the design
  never distinguishes them, so `seatType` in `tokens.ts` is entirely DERIVED.
- Any admin screen for showtimes, any price editor, and any way to draw an aisle or a 2-column seat.
- Responsive behaviour — every frame is a fixed 1440 desktop.
- The currency is RM in the design; this product is **int64 whole VND**, so amounts render through `formatVND`.
