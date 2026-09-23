import type { ReactNode } from 'react';
import { Card, theme as antdTheme } from 'antd';

interface TableCardProps {
  title?: ReactNode;
  extra?: ReactNode;
  style?: React.CSSProperties;
  children: ReactNode;
}

/** Card wrapper fitting a table flush with no body padding. */
export const TableCard = ({ title, extra, style, children }: TableCardProps) => {
  const { token } = antdTheme.useToken();

  return (
    <Card
      title={title}
      extra={extra}
      variant="borderless"
      style={{ boxShadow: token.boxShadowTertiary, ...style }}
      styles={{ body: { padding: 0 } }}
    >
      {children}
    </Card>
  );
};

export default TableCard;
