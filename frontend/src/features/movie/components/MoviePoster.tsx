import { theme as antdTheme } from 'antd';
import { PictureOutlined } from '@ant-design/icons';

interface MoviePosterProps {
  url?: string;
  title: string;
  width?: number;
}

/** Fixed-ratio poster cell keeping table rows stable. */
export const MoviePoster = ({ url, title, width = 40 }: MoviePosterProps) => {
  const { token } = antdTheme.useToken();
  const box: React.CSSProperties = {
    width,
    aspectRatio: '2 / 3',
    borderRadius: token.borderRadius,
    overflow: 'hidden',
    background: token.colorFillSecondary,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: token.colorTextQuaternary,
  };

  if (!url) {
    return (
      <div style={box} aria-hidden>
        <PictureOutlined />
      </div>
    );
  }
  return (
    <div style={box}>
      <img
        src={url}
        alt={title}
        loading="lazy"
        style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
      />
    </div>
  );
};

export default MoviePoster;
