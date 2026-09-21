import type { ReactNode } from 'react';
import { Flex, Typography, theme as antdTheme } from 'antd';

interface PageHeaderProps {
  title: string;
  extra?: ReactNode;
}

export const PageHeader = ({ title, extra }: PageHeaderProps) => {
  const { token } = antdTheme.useToken();

  return (
    <Flex
      align="center"
      justify="space-between"
      wrap="wrap"
      gap={12}
      style={{
        marginBottom: 20,
        paddingBottom: 16,
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
      }}
    >
      <Typography.Title level={4} style={{ margin: 0 }}>
        {title}
      </Typography.Title>
      {extra}
    </Flex>
  );
};

export default PageHeader;
