# Motion patterns — cinema app

Recipes for the screens this product actually has. Every value below is a token from `src/motion.ts` or
`var(--motion-*)`; if you find yourself typing a number, stop and read `../../rules/motion.md`.

Motion imports throughout: `import { motion, AnimatePresence, MotionConfig, useReducedMotion } from 'motion/react'`.
Motion takes **seconds**, so pass `seconds('base')`, not `200`.

---

## Movie card

The workhorse. All of it is CSS — no JS needed.

**What moves**: the card lifts by `lift` and its shadow layer fades in; the poster zooms to `zoom` inside an
`overflow: hidden` frame; a gradient overlay and the "Đặt vé" / "Trailer" actions fade in. `base`, `out`.

```css
.movie-card {
  position: relative;
  transition: transform var(--motion-duration-base) var(--motion-ease-out);
}
/* Shadow lives on a layer so we animate its opacity, never box-shadow. */
.movie-card::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  box-shadow: 0 12px 32px rgb(0 0 0 / 0.28);
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--motion-duration-base) var(--motion-ease-out);
}
.movie-card__media {
  overflow: hidden;
}
.movie-card__media img {
  transition: transform var(--motion-duration-base) var(--motion-ease-out);
}
.movie-card__actions {
  opacity: 1; /* visible by default: touch devices have no hover */
  transition: opacity var(--motion-duration-base) var(--motion-ease-out);
}

@media (hover: hover) and (pointer: fine) {
  .movie-card__actions {
    opacity: 0;
  }
  .movie-card:hover,
  .movie-card:focus-within {
    transform: translateY(calc(-1 * var(--motion-distance-lift)));
  }
  .movie-card:hover::after,
  .movie-card:focus-within::after {
    opacity: 1;
  }
  .movie-card:hover .movie-card__media img,
  .movie-card:focus-within .movie-card__media img {
    transform: scale(var(--motion-scale-zoom));
  }
  .movie-card:hover .movie-card__actions,
  .movie-card:focus-within .movie-card__actions {
    opacity: 1;
  }
}
.movie-card:active {
  transform: scale(var(--motion-scale-press));
}
```

`:focus-within` is what makes the keyboard path work — tabbing to the button inside reveals the same state.

## Grid of cards entering

Stagger the first `stagger.max` cards, show the rest at once.

```tsx
<motion.li
  initial={{ opacity: 0, y: distance.reveal }}
  whileInView={{ opacity: 1, y: 0 }}
  viewport={{ once: true, amount: 0.2 }}
  transition={{ duration: seconds('slow'), ease: easing.out, delay: staggerDelay(index) / 1000 }}
/>
```

Never do this to the first screenful of a list that is already visible on load — above-the-fold content must
not start at `opacity: 0`.

## Hero banner

Crossfade at `decorative`. If it autoplays it needs a visible pause control, it pauses on hover and on focus,
and it does not autoplay at all under reduced motion:

```tsx
const reduced = useReducedMotion();
const autoplay = !reduced && !paused;
```

Parallax, if any, lives only here, stays slight, and is off on mobile and under reduced motion.

## Card → detail page (shared element)

The poster is the shared element: `slow`, `inOut`. Everything else fades + `shift`, `moderate`.

Prefer the View Transitions API with a feature detect, and fall back to `layoutId`:

```tsx
if (document.startViewTransition) {
  document.startViewTransition(() => navigate(PATHS.movieDetail(id)));
} else {
  navigate(PATHS.movieDetail(id)); // layoutId on the poster carries the transition
}
```

## Date / showtime tabs

The indicator slides to the selection with a layout animation, `base`. Do not animate `left` or `width`.

```tsx
{
  tabs.map((tab) => (
    <button key={tab.id} onClick={() => setActive(tab.id)}>
      {tab.label}
      {active === tab.id && (
        <motion.span layoutId="showtime-tab-indicator" transition={spring.base} />
      )}
    </button>
  ));
}
```

## Seat map

The screen with the most elements, so it is the one that has to stay cheap.

- Hover: colour only, `instant`. CSS.
- Selecting a seat: `pop` with `spring.base` — the one place a bounce is allowed, and it is ≤ 0.15.
- A sold seat has **no** hover state and no pointer cursor.
- A seat taken by someone else over SSE **fades its colour at `fast`** — no movement. A seat jumping while the
  user is aiming at it is worse than no feedback.
- The running total uses `.tabular-nums`.
- Do not animate all ~200 seats on mount: render them, then animate only what the user touches.

## Checkout steps

- Progress bar: animate `scaleX` with `transform-origin: left`, `base`. Never animate `width`.
- Step content: fade + `shift` in the direction of travel (forward: enter from `+shift`, exit to `-shift`),
  `moderate`, wrapped in `AnimatePresence mode="wait"`.

## Trailer modal

- Backdrop: fade, `base`.
- Dialog: fade + scale from `enter` to 1, `base`, `out`. Closing: `fast`, `in`.
- Lock background scroll, trap focus, close on Esc, and return focus to the trigger.
- antd `Modal` already handles focus and Esc — prefer configuring it over hand-rolling a dialog.

## Toast

In: `shift` + fade, `base`, `out`. Out: `fast`, `in`. It auto-dismisses, and the timer pauses on hover.
Use `App.useApp()`'s `message` / `notification` — do not build a second toast system.

## Seat-hold countdown

`.tabular-nums` so the digits do not shift. Under 60 seconds it changes to the warning colour with a `fast`
colour transition. **It never blinks** — a flashing countdown reads as an error and hurts anyone sensitive to
flicker.

---

## Quick token reference

```ts
import {
  duration,
  seconds,
  easing,
  spring,
  distance,
  scale,
  staggerDelay,
  cssTransition,
} from '@/motion';
```

| Need                    | Use                                                      |
| ----------------------- | -------------------------------------------------------- |
| CSS transition string   | `cssTransition(['transform', 'opacity'], 'base', 'out')` |
| Motion tween            | `{ duration: seconds('base'), ease: easing.out }`        |
| Motion spring           | `spring.base` (bounce 0.15) or `spring.flat` (0)         |
| Stagger delay (seconds) | `staggerDelay(index) / 1000`                             |
