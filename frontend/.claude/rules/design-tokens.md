# Design token rules

This file outranks any other colour guidance. If a design plugin, a screenshot or your own taste disagrees,
this wins.

## One source of truth

`src/theme/tokens.ts` holds every colour. Nothing else defines one. There are **three** consumers and all
three read the same values:

- **TS** imports from `@/theme` (`brand`, `textOnBrand`, `brandAlpha`, `semantic`, `surface`, `seat`,
  `seatType`, `ratingColor`, `fontFamily`, `cinemaBackdrop`, `customerLightCanvas`, `cinemaGradient`,
  `cinemaGradientPanel`).
- **CSS** uses `var(--cp-*)` from the token block at the top of `src/index.css`.
- **Tailwind v4** (added 2026-09-19) gets the same tokens mapped into its colour namespace in `index.css`, so
  `bg-brand` / `text-on-brand` / `duration-fast` resolve to the tokens and not to a second copy.
- `src/__tests__/design-tokens.test.ts` fails if TS and CSS drift, so change both or neither.
- **antd is configured in exactly one place**: `buildTheme(mode)` in `src/theme/index.ts`. Never pass a colour
  to a component, never write a hex in a component, never add a second `ConfigProvider`.

A component that needs a colour antd already exposes should read it from `antdTheme.useToken()`
(`token.colorBgContainer`, `token.colorBorderSecondary`, …) rather than reaching for a raw token — that is what
keeps dark mode working.

## Tailwind: two things that fail with no error

Tailwind runs with **preflight OFF** — its reset lands on top of antd and breaks the operator screens. In
`index.css`, `antd/dist/reset.css` is imported into a cascade layer BELOW Tailwind's utilities, because an
unlayered JS import would otherwise beat layered utilities on form elements. Layer order is first-appearance
order; **do not reorder those imports.**

1. **Tailwind scans source text with a regex, so a class built at runtime is never generated.**
   `` `text-${tone}` `` emits nothing and fails silently. That is why `src/theme/customerTw.ts` exports fully
   **static** strings (`INK_80`, `INK_BORDER_22`, `TRANSITION_FAST`, …) and why `hover-fine:` variants are
   **pre-fused** there rather than composed at the call site. Add to that file instead of interpolating.
2. **Never use Tailwind's `dark:` variant.** There is no `.dark` class on the DOM — the operator side uses
   antd's `darkAlgorithm`, the customer side swaps `--cp-ink-rgb` and `--cp-backdrop-*`. `dark:` compiles and
   then never matches.

`customerTw.ts` expresses every customer text and border tone as an **alpha of the single `--cp-ink-rgb`
variable** rather than naming a colour, which is what lets the customer zone flip light/dark without a second
palette.

## The palette

| Token                          | Value                                                     | Where it came from                                                                         |
| ------------------------------ | --------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `brand.base`                   | `#D33B56`                                                 | DELIBERATE DIVERGENCE — the QVisionShow Figma pink darkened to clear 4.5:1 (see History)   |
| `brand.hover`                  | `#DC6278`                                                 | DERIVED — base mixed 20% toward white                                                      |
| `brand.active`                 | `#B33249`                                                 | DERIVED — base mixed 15% toward black                                                      |
| `brand.soft`                   | `#F8E2E6`                                                 | DERIVED — base mixed 85% toward white, selected sider item                                 |
| `brand.softer`                 | `#FCF1F3`                                                 | DERIVED — base mixed 93% toward white, tag backgrounds                                     |
| `textOnBrand`                  | `#FFFFFF`                                                 | White clears 4.5:1 on the brand (4.64:1)                                                   |
| `cinemaBackdrop.base/mid/glow` | `#060203` / `#20090D` / `#4A151E`                         | DERIVED — `brand.base` toward black (97% / 85% / 65%); **re-derived with the rebrand**     |
| `customerLightCanvas`          | `#F7F6F4`                                                 | the customer zone's LIGHT canvas — browse/chrome only, never seats or payment              |
| `ratingColor.p/k/t13/t16/t18`  | see Age-rating badges                                     | DERIVED — a rising-restriction ramp, none of them `brand.base`                             |
| `fontFamily.display` / `.body` | Be Vietnam Pro / IBM Plex Sans (+ system fallbacks)       | DERIVED                                                                                    |
| `semantic.danger`              | `#D22F27`                                                 | MEASURED — the delete button                                                               |
| `semantic.warning`             | `#D97706`                                                 | DERIVED — the design has no warning state                                                  |
| `semantic.success`             | `#1DE782`                                                 | DERIVED — the OLD brand green, reused so it doesn't collide with `danger` now brand is red |
| `semantic.info`                | `#2563EB`                                                 | DERIVED                                                                                    |
| `surface.light.*`              | `#FFFFFF` / `#FAFAFA` / `#BFBFBF` / `#EBEBEB` / `#767676` | base and border MEASURED, the rest DERIVED                                                 |
| `seat.*`                       | see Seats — **the whole seat palette was replaced**       | free=grey, selecting=blue, held=amber, sold=red, hover=green                               |
| `seatType.*`                   | see below                                                 | DERIVED — the design never distinguishes seat types                                        |

**MEASURED** means the value was read from the pixels of a Figma frame screenshot (dominant colour of a region),
not estimated by eye. **DERIVED** means this codebase computed it from another token. **DELIBERATE DIVERGENCE**
means it intentionally does not match the Figma file. When you add a colour, say which it is.

## History: the brand colour used to be green, and white-on-brand used to be a deviation

Up to 2026-09-18 `brand.base` was Figma's green `#1DE782`, and white text on it measured only ~1.4:1, so
`textOnBrand` was deliberately a dark green (~9:1) instead of the white the Figma showed. On 2026-09-19 the owner
asked to switch brand to red (CinePlex's `#E4002B`, see the note under "The palette"). White text on THIS colour
clears 4.5:1 on its own (~4.85:1), so `textOnBrand` is plain white again and there is no more deviation to track.
The contrast test still enforces the 4.5:1 floor — it just no longer forces a colour swap to clear it.

### 2026-09-25: red -> `#D33B56`, and why it is not the Figma's pink

The QVisionShow Figma's brand is `#F84565`. White text on it measures **3.48:1**, below the 4.5:1 floor this
file's Never list forbids weakening. Rather than lower the test, `brand.base` was set to `#D33B56`, which is
exactly `#F84565` mixed **15% toward black** — i.e. the DERIVED `active` shade of the Figma pink, so the hue
is the Figma's and only the lightness moved. White on it measures **4.636:1** and clears the floor. The four
brand variants and all three `cinemaBackdrop` steps were re-derived from the new base with the same formulas
(20% white / 15% black / 85% white / 93% white; 97% / 85% / 65% black). **DELIBERATE DIVERGENCE** — the UI is
one step darker than the Figma pink, on purpose, and must not be "corrected" back to `#F84565`.

## Two visual worlds, one brand

- **Operator** (`MainLayout`): light surfaces, white sider, `brand.soft` on the selected menu item, plain white
  content, antd components throughout. Dark mode is antd's `darkAlgorithm` — it is NOT the customer backdrop.
- **Customer** (`CustomerLayout`): Tailwind utilities over the same tokens, no heavy antd. `cinemaBackdrop` is
  **no longer green** — it was re-derived from the red `brand.base` with the rebrand, so the gradient IS tied to
  the brand now. (An earlier version of this file said the opposite; it was correct at the time and is not any
  more.)
- **The customer zone has its OWN light/dark switch**, independent of antd's. Dark is `cinemaGradient` over
  `cinemaBackdrop`; light is `customerLightCanvas`. Text flows from `--cp-ink-rgb`, which is why
  `customerTw.ts` names alphas of that variable instead of colours.
- **Some customer routes stay dark whatever the switch says.** `CustomerLayout` reads two per-route flags off
  `useMatches()`: `forceDark` (seat map, checkout, tickets — the seat palette only reads correctly on dark) and
  `fullBleed` (the account split panel, which must escape `<main>`'s max-width and padding). Set them in the
  route's `handle`, never as a prop.

The brand colour is the same in both. Everything else differs, and mixing them is the mistake to avoid.

## Age-rating badges

`ratingColor` carries one pair per `movie.age_rating` level as a rising-restriction ramp — green `p`, blue `k`,
amber `t13`, orange `t16`, dark red `t18`. Two rules hold and are both tested: **every pair clears 4.5:1**, and
**none of them equals `brand.base`**, so a rating badge can never be mistaken for a call to action. CSS reads
them as `--cp-rating-<level>` and `--cp-rating-<level>-text`.

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
| `vip`      | `#F8E2E6` | `#720016` | bg is `brand.soft`, so there is no second red |
| `couple`   | `#FDE8CE` | `#7A4405` | tint of `semantic.warning`                    |
| `recliner` | `#DDE7FE` | `#1E3A8A` | tint of `semantic.info`                       |

A test enforces 4.5:1 on each pair **and** that the four backgrounds are distinct, so adding a fifth seat type
without a colour fails the suite rather than rendering two types identically.

## Seats

**The seat palette was replaced on 2026-09-19 and no longer follows the brand.** Figma drew only two states
(white free, brand selected); the current spec uses four distinct hues plus a hover, because a seat picker is
read at a glance and "which of these is mine" has to be unmistakable.

| Backend state        | Token                 | Value       | Reads as                                  |
| -------------------- | --------------------- | ----------- | ----------------------------------------- |
| available            | `seat.available`      | `#D1D5DB`   | pale grey, clickable                      |
| available, hovered   | `seat.availableHover` | `#22C55E`   | green — "this one is free"                |
| selected by me       | `seat.selected`       | `#2563EB`   | **blue, NOT the brand red**               |
| held by someone else | `seat.held`           | `#D97706`   | amber — temporary, may free up            |
| sold                 | `seat.sold`           | `#DC2626`   | red, clearly out                          |
| `is_gap`             | `seat.gap`            | transparent | nothing drawn, still occupies a grid cell |

Two consequences worth stating, because both are easy to get backwards:

- **Selected is blue, not brand.** Do not "fix" it to the brand red: sold is red here, and a selected seat that
  looked like a sold one would be the worst possible confusion on this screen. A test asserts the four states
  are all different.
- **`sold` is close to `brand.base`** (`#DC2626` vs `#D33B56`) but is a separate token on purpose — one is a
  state, the other is a call to action. Do not collapse them.

`held` and `sold` differ in **behaviour**, not just appearance, so they must not share a colour.

## Never

- Never write a hex literal in a component, a page or a feature file.
- Never add a colour without saying MEASURED or DERIVED and, if MEASURED, which frame.
- Never introduce a "just for this screen" colour. Note the palette DOES now carry more than one green
  (`semantic.success`, `seat.availableHover`) and more than one red (`brand.base`, `semantic.danger`,
  `seat.sold`) — each is a distinct meaning with a test behind it, not licence to add another.
- Never use `brand.base` for a seat state, or a seat colour for a button.
- Never animate a colour on a large surface — see `.claude/rules/motion.md`; colour transitions are for small
  elements only.
- Never weaken the contrast test to match a design.
