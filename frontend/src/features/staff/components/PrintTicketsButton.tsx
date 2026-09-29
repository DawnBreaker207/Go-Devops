import { useRef, useState } from 'react';
import { Alert, Button, Modal, Spin } from 'antd';
import { PrinterOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { ticketApi } from '@/api/ticket.api';
import type { OrderDetail } from '@/types';
import { formatDateTime, formatVND } from '@/utils/format';

interface PrintTicketsButtonProps {
  order: OrderDetail;
}

/** Counter ticket printing: one slip per ticket, printed via the browser. */
export const PrintTicketsButton = ({ order }: PrintTicketsButtonProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [qrByTicket, setQrByTicket] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [qrError, setQrError] = useState<string | null>(null);
  const cancelledRef = useRef(false);

  const openPrint = () => {
    cancelledRef.current = false;
    setOpen(true);
    setLoading(true);
    setQrError(null);
    void Promise.all(
      order.tickets.map((ticket) =>
        ticketApi
          .qr(ticket.id)
          .then((res) => ({ id: ticket.id, qr: res.qr_base64 }))
          .catch(() => ({ id: ticket.id, qr: '' }))
      )
    ).then((rows) => {
      if (cancelledRef.current) return;
      const missing = rows.filter((row) => !row.qr);
      if (missing.length > 0) {
        setQrError(t('boxOffice.printQrMissing', { count: missing.length }));
      }
      setQrByTicket(Object.fromEntries(rows.map((row) => [row.id, row.qr])));
      setLoading(false);
    });
  };

  const closePrint = () => {
    cancelledRef.current = true;
    setOpen(false);
  };

  return (
    <>
      <Button icon={<PrinterOutlined />} onClick={openPrint}>
        {t('boxOffice.printTickets')}
      </Button>
      <Modal
        open={open}
        onCancel={closePrint}
        title={t('boxOffice.printTickets')}
        width={640}
        footer={[
          <Button key="close" onClick={closePrint}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="print"
            type="primary"
            icon={<PrinterOutlined />}
            disabled={loading || order.tickets.length === 0}
            onClick={() => window.print()}
          >
            {t('boxOffice.print')}
          </Button>,
        ]}
      >
        {qrError ? (
          <Alert type="warning" showIcon style={{ marginBottom: 16 }} message={qrError} />
        ) : null}
        {loading ? (
          <div style={{ textAlign: 'center', padding: 32 }}>
            <Spin />
          </div>
        ) : (
          <div className="cp-print-sheet">
            {order.tickets.map((ticket) => (
              <div
                key={ticket.id}
                className="cp-ticket-slip"
                style={{
                  background: '#fff',
                  color: '#000',
                  border: '1px solid #000',
                  borderRadius: 8,
                  padding: 16,
                  marginBottom: 16,
                  display: 'flex',
                  gap: 16,
                  alignItems: 'center',
                  pageBreakInside: 'avoid',
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 18, fontWeight: 800 }}>{order.showtime?.movie_title}</div>
                  <div style={{ fontSize: 13 }}>
                    {order.showtime?.hall_name} ·{' '}
                    {order.showtime ? formatDateTime(order.showtime.start_at) : ''}
                  </div>
                  <div style={{ fontSize: 15, marginTop: 8 }}>
                    {t('booking.seat')}: <strong>{ticket.seat_label}</strong> ·{' '}
                    {t(`booking.seatType_${ticket.seat_type}`, ticket.seat_type)} ·{' '}
                    {formatVND(ticket.price)}
                  </div>
                  <div className="tabular-nums" style={{ fontSize: 13, marginTop: 4 }}>
                    {t('boxOffice.ticketCode')}: <strong>{ticket.code}</strong>
                  </div>
                </div>
                {qrByTicket[ticket.id] ? (
                  <img
                    src={`data:image/png;base64,${qrByTicket[ticket.id]}`}
                    alt={ticket.code}
                    width={120}
                    height={120}
                    style={{ flexShrink: 0 }}
                  />
                ) : null}
              </div>
            ))}
          </div>
        )}
      </Modal>
    </>
  );
};

export default PrintTicketsButton;
