import type { ButtonHTMLAttributes } from 'react';
import { buttonClassName, type ButtonVariant } from './buttonStyles';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  /** Full parent width. */
  block?: boolean;
}

/** Shared customer <button>; classes built once in buttonStyles.ts, no .css file. */
export const Button = ({
  variant = 'primary',
  block,
  className,
  type = 'button',
  ...rest
}: ButtonProps) => (
  <button type={type} className={buttonClassName(variant, { block, className })} {...rest} />
);

export default Button;
