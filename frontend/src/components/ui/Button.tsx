import type { ButtonHTMLAttributes } from 'react';
import { buttonClassName, type ButtonVariant } from './buttonStyles';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  /** Full parent width. */
  block?: boolean;
  /** Fully rounded, as the QVisionShow frame draws its calls to action. */
  pill?: boolean;
}

/** Shared customer <button>; classes built once in buttonStyles.ts, no .css file. */
export const Button = ({
  variant = 'primary',
  block,
  pill,
  className,
  type = 'button',
  ...rest
}: ButtonProps) => (
  <button type={type} className={buttonClassName(variant, { block, pill, className })} {...rest} />
);

export default Button;
