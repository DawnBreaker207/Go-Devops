---
name: fe-motion
description: fe-motion - add or review motion on a FrontEnd-CP page or component using the project motion tokens. Use for hiệu ứng, chuyển động mượt, hover, chuyển cảnh, cuộn (scroll reveal), animation, transition work on any screen.
argument-hint: <page or component, e.g. "MovieCard" or "/showtimes">
---

# /fe-motion $ARGUMENTS

Give **$ARGUMENTS** motion that feels deliberate rather than decorated. Read `patterns.md` in this directory for
the ready-made recipes; it covers the cinema-app cases (movie card, hero, seat map, checkout, modal, toast,
hold countdown).

## 1. Load the rules first

- `../../rules/motion.md` — what may be animated, direction, duration, hover/focus, scroll, reduced motion.
- `src/motion.ts` and the `MOTION TOKENS` block in `src/index.css` — the only numbers you are allowed to use.

If a token does not exist for what you need, say so and propose adding one. Do not invent a value inline.

## 2. List the motion moments before writing code

Write out, for the target: every state change a user can cause or observe (mount, hover, focus, press, select,
error, loading, exit, route change, realtime update). For each, name the token pair you will use, e.g.
`hover -> transform lift, base, out`. Show me that list.

Then cut it down: **one screen gets at most one prominent moment** — usually the entrance or the hero.
Everything else is a micro-interaction the user should feel and not notice.

## 3. Apply

Follow `patterns.md`. Defaults that hold unless the pattern says otherwise:

- Hover / focus / press / colour → **CSS**, scoped with `@media (hover: hover) and (pointer: fine)`.
- Mount / unmount, layout change, shared element, scroll reveal, stagger → **Motion** (`motion/react`).
- Transform and opacity only. Shadow changes animate a shadow layer's opacity.

## 4. Self-review before you claim it works

- [ ] No raw numbers, no `transition: all` — every value is a token.
- [ ] Only `transform` / `opacity` animate (plus colour on small elements, shadow via a layer).
- [ ] Exit is one duration step shorter than enter, and uses `in` easing.
- [ ] Every hover has a `:focus-visible` equivalent; nothing is reachable by hover only on touch.
- [ ] Buttons and tappable cards have a press state.
- [ ] Reduced motion: no translate, no zoom, no parallax, no autoplay — check by toggling the OS setting or
      emulating it in DevTools.
- [ ] Skeletons match the real box; no layout shift on load.
- [ ] Under ~20 elements animate at once; `will-change` is not left on permanently.
- [ ] Nothing above the fold starts at `opacity: 0`.
- [ ] Digits that change in place use `.tabular-nums`.

## 5. Check it in the browser, not just in the test runner

The dev server is `npm run dev` on :3000. With chrome-devtools available:

- Console clean — no warnings from the animation library.
- **Layout shift**: run a performance trace on the screen and confirm no CLS from an entrance or a skeleton swap.
- **Keyboard**: tab through the screen; every hover affordance must appear on focus, and focus must stay visible
  while an element animates.
- Emulate `prefers-reduced-motion: reduce` and confirm movement stops but the UI still reads.

## 6. Hand back

Report in Vietnamese:

- the motion moments you implemented and the token pair for each,
- what you verified in the browser and what you could not,
- **a short list of what I have to judge with my own eyes** — feel and timing are not testable. Say exactly what
  to look at and what would count as wrong (e.g. "hover a movie card: the lift should read as a nudge, not a
  jump; if the poster zoom feels like it overshoots, drop `zoom` to a new token").
