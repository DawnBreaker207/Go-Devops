import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Button, Result } from 'antd';

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Diem cam de gan Sentry / log service sau nay
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  private handleReload = () => {
    window.location.reload();
  };

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    return (
      <Result
        status="error"
        title="Ứng dụng gặp sự cố"
        subTitle={error.message}
        extra={
          <Button type="primary" onClick={this.handleReload}>
            Tải lại trang
          </Button>
        }
      />
    );
  }
}

export default ErrorBoundary;
