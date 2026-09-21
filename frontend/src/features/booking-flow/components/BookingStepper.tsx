import { INK, INK_40, INK_65, INK_BORDER_25 } from '@/theme/customerTw';

export type BookingStep = 1 | 2 | 3;

export interface StepDef {
  n: BookingStep;
  label: string;
  icon: string;
}

export interface BookingStepperProps {
  current: BookingStep;
  steps: StepDef[];
  /** Fired only for a PAST step (n < current) - forward moves go through action buttons. */
  onStepBack?: (n: BookingStep) => void;
}

/** Step bar (1 Seats -> 2 Combo -> 3 Payment), sticky. No route navigation: past steps call `onStepBack`, future steps are indicators. */
export const BookingStepper = ({ current, steps, onStepBack }: BookingStepperProps) => {
  return (
    // Follows chrome header: theme-reactive, never fixed-dark.
    <div className="sticky top-0 z-10 -mx-4 mb-5 border-b border-(--cp-chrome-border) bg-(--cp-chrome-bg) px-4 py-3 backdrop-blur-md sm:-mx-8 sm:px-8">
      <div className="mx-auto flex max-w-300 items-center justify-between gap-6">
        {steps.map((step, i) => {
          const state = step.n === current ? 'active' : step.n < current ? 'done' : 'todo';
          const backable = step.n < current && onStepBack !== undefined;
          const inner = (
            <>
              <span
                className={[
                  'flex h-7 w-7 items-center justify-center rounded-full text-xs',
                  state === 'active'
                    ? 'bg-brand text-on-brand'
                    : `border ${INK_BORDER_25} ${INK_65}`,
                ].join(' ')}
                aria-hidden="true"
              >
                {step.icon}
              </span>
              <span className={state === 'active' ? 'border-b-2 border-brand pb-0.5' : ''}>
                {step.n}. {step.label}
              </span>
            </>
          );
          return (
            <div key={step.n} className="flex items-center gap-6">
              <div
                className={[
                  'flex items-center gap-2 text-sm font-semibold',
                  state === 'active' ? INK : state === 'done' ? INK_65 : INK_40,
                ].join(' ')}
              >
                {backable ? (
                  <button
                    type="button"
                    onClick={() => onStepBack?.(step.n)}
                    aria-label={step.label}
                    className={`flex cursor-pointer items-center gap-2 border-none bg-transparent p-0 no-underline transition-colors duration-fast ease-out hover-fine:text-brand ${INK}`}
                  >
                    {inner}
                  </button>
                ) : (
                  inner
                )}
              </div>
              {i < steps.length - 1 ? (
                <span className="h-px w-8 bg-[rgb(var(--cp-ink-rgb))]/15" aria-hidden="true" />
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default BookingStepper;
