import type { ReactNode } from 'react';
import BookingStepper, { type BookingStep, type StepDef } from './BookingStepper';

export interface BookingLayoutProps {
  current: BookingStep;
  steps: StepDef[];
  onStepBack?: (n: BookingStep) => void;
  main: ReactNode;
  sidebar: ReactNode;
  mobileBar?: ReactNode;
}

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

    <div className="mx-auto flex max-w-300 flex-col gap-6 lg:flex-row lg:items-start">
      <div className="min-w-0 flex-1">{main}</div>
      <div className="hidden w-full flex-none lg:block lg:w-95">{sidebar}</div>
    </div>

    {mobileBar ? (
      <div className="sticky bottom-0 z-10 -mx-4 mt-6 flex items-center gap-3 border-t border-(--cp-chrome-border) bg-(--cp-chrome-bg-strong) p-3 backdrop-blur-md lg:hidden">
        {mobileBar}
      </div>
    ) : null}

    <div className="mt-6 lg:hidden">{sidebar}</div>
  </>
);

export default BookingLayout;
