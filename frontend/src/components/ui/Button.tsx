import type { ButtonHTMLAttributes } from 'react';
import { buttonClassName, type ButtonVariant } from './buttonStyles';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  block?: boolean;
  pill?: boolean;
}

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
