import { Spin } from 'antd';

interface LoadingProps {
  fullscreen?: boolean;
  tip?: string;
}

export const Loading = ({ fullscreen = false, tip }: LoadingProps) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: fullscreen ? '100vh' : 240,
      width: '100%',
    }}
  >
    <Spin size="large" tip={tip}>
      {tip ? <div style={{ padding: 24 }} /> : null}
    </Spin>
  </div>
);

export default Loading;
