---
name: fe-component
description: fe-component - add a FrontEnd-CP component (feature-local or cross-feature) that matches the repo's existing antd + props + i18n conventions. Use when building a piece of a screen rather than a whole page.
argument-hint: <what the component does, e.g. "SeatMap grid chon ghe cho trang booking">
---

# /fe-component $ARGUMENTS

Build **$ARGUMENTS**.

## 1. Decide where it lives

- Used by one feature -> `src/features/<domain>/components/<Name>.tsx`.
- Used by two or more features -> `src/components/<Name>.tsx` (currently only `Loading`, `PageHeader`,
  `ErrorBoundary` live there — adding a fourth should be a deliberate choice, say so).
- Do **not** put it in `src/hooks/` or `src/assets/` — both are empty and untracked; see
  `../../context/decisions.md` before creating either.

## 2. Shape

```tsx
interface SeatMapProps {
  seats: SeatMapSeat[];
  onSelect: (id: string) => void;
  disabled?: boolean;
}

export const SeatMap = ({ seats, onSelect, disabled = false }: SeatMapProps) => {
  ...
};

export default SeatMap;
```

- Arrow-function const, local `interface <Name>Props` directly above, named **and** default export.
- No class components — `ErrorBoundary` is the single exception and stays that way.
- Props are data + callbacks. A component does not fetch: pass data in, or call a hook from
  `src/features/<domain>/hooks/`. Never call `apiClient` from a component.

## 3. Conventions that apply

- antd components + `antdTheme.useToken()` tokens; inline `style={{...}}`. No CSS modules, no Tailwind.
- Every user-facing string through `t()`, key added to **both** `vi.json` and `en.json`.
- Toasts via `App.useApp()`, never the static `message`.
- Modals: `{ open, entity | null, confirmLoading, onCancel, onSubmit }`, `destroyOnClose`, `preserve={false}`,
  defaults seeded in a `useEffect` keyed on `[open, entity, form]`.
- Forms: antd `Form` + `rules`. There is no schema validation library — do not add one.
- Give interactive rows/buttons an `aria-label` so they are testable. (The existing test queries antd form labels
  and button text, not `aria-label` — `MoviesPage` sets them, but no test asserts on one yet.)
- Types come from `@/types`; everything else is imported by deep path.

## 4. Verify

```bash
npx tsc --noEmit -p tsconfig.app.json && npm run lint && npm test
```

A component with real logic (seat selection, price math, a reducer) deserves a test in
`src/features/<domain>/__tests__/`. Follow the existing test: local `render...` helper wrapping
`<ConfigProvider><AntdApp><MemoryRouter>`, explicit imports from `vitest`, `userEvent.setup()`.

## 5. Report

In Vietnamese: where you put it and why, its props, which i18n keys you added, and whether you wrote a test.
