import type { ReactNode } from 'react';
import { Flex, Typography } from 'antd';

interface PageHeaderProps {
  title: string;
  extra?: ReactNode;
}

export const PageHeader = ({ title, extra }: PageHeaderProps) => (
  <Flex align="center" justify="space-between" style={{ marginBottom: 16 }}>
    <Typography.Title level={4} style={{ margin: 0 }}>
      {title}
    </Typography.Title>
    {extra}
  </Flex>
);

export default PageHeader;
