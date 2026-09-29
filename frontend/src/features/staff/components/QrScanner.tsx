import { useEffect, useRef, useState } from 'react';
import { Alert, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import jsQR from 'jsqr';

interface QrScannerProps {
  /** Called once per decoded code; the parent closes the scanner. */
  onScan: (code: string) => void;
}

// Decode cooldown: one physical QR must not fire a burst of results.
const SCAN_COOLDOWN_MS = 1500;
const FRAME_WIDTH = 640;

export const QrScanner = ({ onScan }: QrScannerProps) => {
  const { t } = useTranslation();
  const videoRef = useRef<HTMLVideoElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(true);
  const onScanRef = useRef(onScan);

  const supported = typeof navigator !== 'undefined' && !!navigator.mediaDevices?.getUserMedia;

  useEffect(() => {
    onScanRef.current = onScan;
    if (!supported) return undefined;
    let cancelled = false;
    let rafId = 0;
    let lastFire = 0;
    let stream: MediaStream | null = null;
    let scratch: { canvas: HTMLCanvasElement; context: CanvasRenderingContext2D } | null = null;

    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: 'environment' }, audio: false })
      .then((opened) => {
        if (cancelled) {
          opened.getTracks().forEach((track) => track.stop());
          return;
        }
        stream = opened;
        const video = videoRef.current;
        if (!video) return;
        video.srcObject = opened;
        void video.play().then(() => {
          if (!cancelled) setStarting(false);
        });

        const tick = () => {
          if (cancelled) return;
          const v = videoRef.current;
          if (v && v.readyState === v.HAVE_ENOUGH_DATA && v.videoWidth > 0) {
            if (!scratch) {
              const canvas = document.createElement('canvas');
              const context = canvas.getContext('2d', { willReadFrequently: true });
              if (!context) {
                setError(t('boxOffice.scanFailed'));
                return;
              }
              scratch = { canvas, context };
            }
            const scale = FRAME_WIDTH / v.videoWidth;
            scratch.canvas.width = FRAME_WIDTH;
            scratch.canvas.height = Math.max(1, Math.round(v.videoHeight * scale));
            scratch.context.drawImage(v, 0, 0, scratch.canvas.width, scratch.canvas.height);
            const frame = scratch.context.getImageData(
              0,
              0,
              scratch.canvas.width,
              scratch.canvas.height
            );
            const decoded = jsQR(frame.data, frame.width, frame.height);
            const now = Date.now();
            if (decoded?.data && now - lastFire > SCAN_COOLDOWN_MS) {
              lastFire = now;
              onScanRef.current(decoded.data.trim());
            }
          }
          rafId = requestAnimationFrame(tick);
        };
        rafId = requestAnimationFrame(tick);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setStarting(false);
        setError(
          err instanceof DOMException && err.name === 'NotAllowedError'
            ? t('boxOffice.scanDenied')
            : t('boxOffice.scanFailed')
        );
      });

    return () => {
      cancelled = true;
      cancelAnimationFrame(rafId);
      stream?.getTracks().forEach((track) => track.stop());
    };
    // Intentionally [t]-only: onScan rides the ref (a fresh parent closure per
    // render must not restart the camera) and `supported` never flips at runtime.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [t]);

  if (!supported) {
    return <Alert type="error" showIcon message={t('boxOffice.scanUnsupported')} />;
  }

  if (error) {
    return <Alert type="error" showIcon message={error} />;
  }

  return (
    <div style={{ position: 'relative', background: '#000', borderRadius: 8, overflow: 'hidden' }}>
      {starting ? (
        <div style={{ textAlign: 'center', padding: 32 }}>
          <Spin />
        </div>
      ) : null}
      {/* playsInline keeps iOS Safari in the modal instead of fullscreen. */}
      <video ref={videoRef} playsInline muted style={{ width: '100%', display: 'block' }} />
    </div>
  );
};

export default QrScanner;
