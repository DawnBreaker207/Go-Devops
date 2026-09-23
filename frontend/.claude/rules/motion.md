---
paths:
  - '**/src/features/**/*.tsx'
  - '**/src/components/**/*.tsx'
  - '**/src/layouts/**/*.tsx'
  - '**/src/pages/**/*.tsx'
  - '**/src/routes/**/*.tsx'
  - '**/src/**/*.css'
  - '**/src/motion.ts'
---

# Motion rules

This file outranks any other aesthetic guidance, including a design plugin's. If they disagree, this wins.

## Tokens are the only numbers

Every duration, easing, distance, scale and stagger comes from `src/motion.ts` (TS) or the matching
`var(--motion-*)` custom property in `src/index.css`. The two are kept in step by
`src/__tests__/motion-tokens.test.ts`, which fails if they drift — so change both or neither.

| Group    | Tokens                                                                                               |
| -------- | ---------------------------------------------------------------------------------------------------- |
| duration | `instant` 100ms · `fast` 150ms · `base` 200ms · `moderate` 300ms · `slow` 450ms · `decorative` 600ms |
| easing   | `out` enter + hover · `in` exit · `inOut` moving on screen · `linear` spinners only                  |
| spring   | `spring.base` (bounce 0.15) · `spring.flat` (bounce 0) — never more than 0.15                        |
| distance | `lift` 4px · `shift` 12px · `reveal` 16px                                                            |
| scale    | `enter` 0.96 · `press` 0.98 · `zoom` 1.05 · `pop` 1.08                                               |
| stagger  | 50ms per item, first 6 only, the rest appear together (`staggerDelay(i)`)                            |

`decorative` is for a hero or a background, never for UI feedback.

**Never** write a raw number in a component, and **never** write `transition: all`. In CSS list the properties:
`transition: transform var(--motion-duration-base) var(--motion-ease-out), opacity ...`, or build it with
`cssTransition(['transform', 'opacity'], 'base', 'out')` from `src/motion.ts`.

## One library, and only where CSS cannot reach

Hover, focus, press and simple state colour changes are **CSS**. Reach for JS motion only for orchestration
CSS cannot do: enter/exit of a mounted element, layout/FLIP animation, shared element, scroll reveal, stagger.
The chosen library is **Motion** (`npm install motion`, `import { motion, AnimatePresence, MotionConfig,
useReducedMotion } from 'motion/react'`). Do not add a second animation library, a smooth-scroll library
(Lenis and friends) or GSAP — ask first.

## What may be animated

- **Only `transform` and `opacity`.** They are the two the compositor can handle without layout or paint.
- Never animate `width`, `height`, `top`, `left`, `margin` or `padding`. To change size or position, use a
  layout animation (FLIP) — `layout` / `layoutId` in Motion.
- Colour (`color`, `background-color`, `border-color`) may transition on **small** elements: a button, a tag,
  a seat. Not on a full-width panel.
- Shadows: animate the `opacity` of a shadow layer (a `::before`/`::after` that carries the `box-shadow`),
  never `box-shadow` itself.

## Direction and duration

- Entering: `out` easing. Leaving: `in` easing and **one duration step shorter** than the entrance.
- `linear` is only for a spinner, a progress bar or a loop.
- UI feedback (hover, press, toggle, toast) ≤ `moderate`. Page transition and shared element ≤ `slow`.
- No animation may block input. The user can always click through or past it.

## Pointer, hover and focus

There is **no Tailwind** in this repo, so scope hover by hand:

```css
@media (hover: hover) and (pointer: fine) {
  .movie-card:hover .movie-card__media img {
    transform: scale(var(--motion-scale-zoom));
  }
}
```

- Every hover effect needs a `:focus-visible` equivalent, on the same element or its card.
- Buttons and tappable cards get a press state: `transform: scale(var(--motion-scale-press))` on `:active`.
- Anything revealed by hover must be **visible by default on touch** — never hide an action behind hover only.

## Scroll

- Native scrolling stays native. No scroll-jacking, no scroll hijack libraries.
- Reveal fires **once** when ~20% of the element is in view: fade + `reveal` translate, `slow`, staggered.
  `viewport={{ once: true, amount: 0.2 }}`.
- Parallax only in a hero, very slight, disabled on mobile and under reduced motion.
- **Above-the-fold content is never hidden waiting for an animation.** Do not ship a heading or a hero image
  with `opacity: 0` as its initial state.

## Reduced motion

- `MotionConfig reducedMotion="user"` wraps the app (it drops transform and layout animations and keeps
  opacity and colour). CSS is handled by the `@media (prefers-reduced-motion: reduce)` block in
  `src/index.css`, which zeroes the distance and scale tokens and clamps every duration to `fast`.
- Under reduced motion: no translation, no zoom, no parallax, no autoplay. A fade of at most `fast` is fine.
- For anything the tokens cannot express (an autoplaying carousel, a looping video), branch on
  `useReducedMotion()`.

## Loading

- A skeleton must match the real content's box so swapping it in causes **no layout shift**.
- Skeleton → content is a `fast` fade.
- A button in its loading state keeps its width; do not let the label collapse.
- Never stretch an animation to cover a slow API. Show the real state instead.

## Performance

- Keep concurrently animating elements under ~20. A long list staggers its first 6 and shows the rest at once.
- `will-change` is set only while an animation is running, never permanently in a stylesheet.
- Prefer one animated parent over many animated children.

## Numbers that make text jump

Money, countdowns and any digit that changes in place uses the `.tabular-nums` class from `src/index.css`.
