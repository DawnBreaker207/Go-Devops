# Design token rules

This file outranks any other colour guidance. If a design plugin, a screenshot or your own taste disagrees,
this wins.

## One source of truth

`src/theme/tokens.ts` holds every colour. Nothing else defines one.

- **TS** imports from `@/theme` (`brand`, `textOnBrand`, `semantic`, `surface`, `seat`, `cinemaBackdrop`,
  `cinemaGradient`).
- **CSS** uses `var(--cp-*)` from the token block at the top of `src/index.css`.
- `src/__tests__/design-tokens.test.ts` fails if the two drift, so change both or neither.
- **antd is configured in exactly one place**: `buildTheme(mode)` in `src/theme/index.ts`. Never pass a colour
  to a component, never write a hex in a component, never add a second `ConfigProvider`.

A component that needs a colour antd already exposes should read it from `antdTheme.useToken()`
(`token.colorBgContainer`, `token.colorBorderSecondary`, …) rather than reaching for a raw token — that is what
keeps dark mode working.

## The palette

| Token                                  | Value                                                     | Where it came from                                                                         |
| -------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `brand.base`                           | `#E4002B`                                                 | DELIBERATE DIVERGENCE — CinePlex's red, not Figma (see note above)                         |
| `brand.hover`                          | `#E93355`                                                 | DERIVED — base mixed 20% toward white                                                      |
| `brand.active`                         | `#C20025`                                                 | DERIVED — base mixed 15% toward black                                                      |
| `brand.soft`                           | `#FBD9DF`                                                 | DERIVED — base mixed 85% toward white, selected sider item                                 |
| `brand.softer`                         | `#FDEDF0`                                                 | DERIVED — base mixed 93% toward white, tag backgrounds                                     |
| `textOnBrand`                          | `#FFFFFF`                                                 | White clears 4.5:1 on the red (~4.85:1)                                                    |
| `cinemaBackdrop.base/mid/glow`         | `#020700` / `#083D1F` / `#0C582F`                         | MEASURED — the customer gradient                                                           |
| `semantic.danger`                      | `#D22F27`                                                 | MEASURED — the delete button                                                               |
| `semantic.warning`                     | `#D97706`                                                 | DERIVED — the design has no warning state                                                  |
| `semantic.success`                     | `#1DE782`                                                 | DERIVED — the OLD brand green, reused so it doesn't collide with `danger` now brand is red |
| `semantic.info`                        | `#2563EB`                                                 | DERIVED                                                                                    |
| `surface.light.*`                      | `#FFFFFF` / `#FAFAFA` / `#BFBFBF` / `#EBEBEB` / `#767676` | base and border MEASURED, the rest DERIVED                                                 |
| `seat.available` / `seat.selected`     | `#FFFFFF` / `#E4002B`                                     | `available` MEASURED; `selected` = `brand.base`                                            |
| `seat.sold` / `seat.held` / `seat.gap` | `#3A3F3C` / `#D97706` / transparent                       | DERIVED — the design shows only two seat states                                            |
| `seatType.*`                           | see below                                                 | DERIVED — the design never distinguishes seat types                                        |

**MEASURED** means the value was read from the pixels of a Figma frame screenshot (dominant colour of a region),
not estimated by eye. **DERIVED** means this codebase computed it from another token. **DELIBERATE DIVERGENCE**
means it intentionally does not match the Figma file. When you add a colour, say which it is.

## History: the brand colour used to be green, and white-on-brand used to be a deviation

Up to 2026-09-18 `brand.base` was Figma's green `#1DE782`, and white text on it measured only ~1.4:1, so
`textOnBrand` was deliberately a dark green (~9:1) instead of the white the Figma showed. On 2026-09-19 the owner
asked to switch brand to red (CinePlex's `#E4002B`, see the note under "The palette"). White text on THIS colour
clears 4.5:1 on its own (~4.85:1), so `textOnBrand` is plain white again and there is no more deviation to track.
The contrast test still enforces the 4.5:1 floor — it just no longer forces a colour swap to clear it.

## Two visual worlds, one brand

- **Admin** (`MainLayout`): light surfaces, white sider, `brand.soft` on the selected menu item, plain white
  content. Dark mode is antd's `darkAlgorithm` — it is NOT the customer gradient.
- **Customer** (`CustomerLayout`): `cinemaGradient` over `cinemaBackdrop` (still green-tinted — that gradient was
  never tied to `brand`, see `cinemaBackdrop` in tokens.ts), white text, brand red for every primary action.

The brand colour is the same in both. Everything else differs, and mixing them is the mistake to avoid.

## Two seat axes, not one

`seat` and `seatType` are different things and must not be mixed:

- **`seat`** is the customer seat picker, on the dark backdrop. It says what a seat's **availability** is —
  available, selected, held, sold, gap.
- **`seatType`** is the admin seat editor, on a light surface. It says what **kind** of seat it is —
  standard, vip, couple, recliner. Backend calls these `models.AllSeatTypes`, and **no endpoint returns the
  list**, so the four are hard-coded in `src/types/hall.ts`.

| Seat type  | bg        | fg        | Note                                          |
| ---------- | --------- | --------- | --------------------------------------------- |
| `standard` | `#EDF0EE` | `#3A3F3C` | fg reuses `seat.sold`; `#767676` fails 4.5:1  |
| `vip`      | `#FBD9DF` | `#720016` | bg is `brand.soft`, so there is no second red |
| `couple`   | `#FDE8CE` | `#7A4405` | tint of `semantic.warning`                    |
| `recliner` | `#DDE7FE` | `#1E3A8A` | tint of `semantic.info`                       |

A test enforces 4.5:1 on each pair **and** that the four backgrounds are distinct, so adding a fifth seat type
without a colour fails the suite rather than rendering two types identically.

## Seats

Figma defines only _available_ (white) and _selected_ (brand colour, now red). The backend has more states, so:

| Backend state        | Token            | Reads as                                  |
| -------------------- | ---------------- | ----------------------------------------- |
| available            | `seat.available` | white, clickable                          |
| selected by me       | `seat.selected`  | brand red                                 |
| held by someone else | `seat.held`      | amber — temporary, may free up            |
| sold                 | `seat.sold`      | dark grey, clearly out                    |
| `is_gap`             | `seat.gap`       | nothing drawn, still occupies a grid cell |

`held` and `sold` differ in **behaviour**, not just appearance, so they must not share a colour.

## Never

- Never write a hex literal in a component, a page or a feature file.
- Never add a colour without saying MEASURED or DERIVED and, if MEASURED, which frame.
- Never introduce a second green, a second red, or a "just for this screen" colour.
- Never animate a colour on a large surface — see `.claude/rules/motion.md`; colour transitions are for small
  elements only.
- Never weaken the contrast test to match a design.
