import type { InputHTMLAttributes } from 'react';

export interface FieldInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hint?: string;
  error?: string;
}

/** Shared labeled input for customer auth forms. Fixed light colors by Figma design (form always sits on light, even inside dark overlays); unlike most customer components it never follows light/dark. */
export const FieldInput = ({ label, hint, error, id, className, ...rest }: FieldInputProps) => (
  <div className="mb-3.5">
    <label htmlFor={id} className="mb-1.5 block text-[13px] text-[#444]">
      {label}
    </label>
    <input
      id={id}
      className={[
        'min-h-10 w-full rounded-lg border border-[var(--cp-surface-border)] bg-white px-3 text-sm text-[#141414]',
        'transition-[border-color] duration-fast ease-out focus:border-brand focus:outline-none',
        className ?? '',
      ]
        .filter(Boolean)
        .join(' ')}
      {...rest}
    />
    {hint ? <p className="mt-0 mb-3.5 text-[13px] text-[#666]">{hint}</p> : null}
    {error ? <span className="mt-1 block text-xs text-danger">{error}</span> : null}
  </div>
);

export default FieldInput;
