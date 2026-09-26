import { Link, type LinkProps } from 'react-router-dom';
import { buttonClassName, type ButtonVariant } from './buttonStyles';

export interface LinkButtonProps extends LinkProps {
  variant?: ButtonVariant;
  block?: boolean;
  pill?: boolean;
}

/** Navigation-styled <Link>; shares Button styling via buttonClassName. */
export const LinkButton = ({
  variant = 'primary',
  block,
  pill,
  className,
  ...rest
}: LinkButtonProps) => (
  <Link className={buttonClassName(variant, { block, pill, className })} {...rest} />
);

export default LinkButton;
