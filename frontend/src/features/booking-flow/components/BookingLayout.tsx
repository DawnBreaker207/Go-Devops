import type { ReactNode } from 'react';
import BookingStepper, { type BookingStep, type StepDef } from './BookingStepper';

export interface BookingLayoutProps {
  current: BookingStep;
  steps: StepDef[];
  /** Click a past step to go back - omit and the stepper is display-only. */
  onStepBack?: (n: BookingStep) => void;
  /** Left column: each step's main content. */
  main: ReactNode;
  /** Right column on desktop (sticky), full content restacked below on <lg. */
  sidebar: ReactNode;
  /** Slim sticky bottom bar, <lg only (e.g. total + button). */
  mobileBar?: ReactNode;
}

/** Shared frame for all booking steps: stepper on top + 2-column grid. Only the left column changes. */
export const BookingLayout = ({
  current,
  steps,
  onStepBack,
  main,
  sidebar,
  mobileBar,
}: BookingLayoutProps) => (
  <>
    <BookingStepper current={current} steps={steps} onStepBack={onStepBack} />

    {/* Follows chrome (header/BottomNav): theme-reactive surfaces. */}
    <div className="mx-auto flex max-w-300 flex-col gap-6 lg:flex-row lg:items-start">
      <div className="min-w-0 flex-1">{main}</div>
      <div className="hidden w-full flex-none lg:block lg:w-95">{sidebar}</div>
    </div>

    {mobileBar ? (
      // Follows chrome: theme-reactive surface.
      <div className="sticky bottom-0 z-10 -mx-4 mt-6 flex items-center gap-3 border-t border-(--cp-chrome-border) bg-(--cp-chrome-bg-strong) p-3 backdrop-blur-md lg:hidden">
        {mobileBar}
      </div>
    ) : null}

    <div className="mt-6 lg:hidden">{sidebar}</div>
  </>
);

export default BookingLayout;
