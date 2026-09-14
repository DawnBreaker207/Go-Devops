package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strconv"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/notify"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// TicketEmailService sends the ticket email of a confirmed booking (F19).
// Email is never on the money path: a failure leaves the booking CONFIRMED
// and the ticket visible on the web (R-ML1).
type TicketEmailService interface {
	// Send mails one booking once. sent=false with nil error means there was
	// nothing to send (not confirmed or already sent).
	Send(ctx context.Context, bookingID string) (sent bool, err error)
	// PendingIDs lists up to limit confirmed bookings still without email.
	PendingIDs(ctx context.Context, limit int) ([]string, error)
}

type ticketEmailService struct {
	db       *gorm.DB
	repo     repository.BookingRepository
	mailer   notify.Mailer
	location *time.Location
}

func NewTicketEmailService(db *gorm.DB, repo repository.BookingRepository, mailer notify.Mailer, location *time.Location) TicketEmailService {
	return &ticketEmailService{db: db, repo: repo, mailer: mailer, location: location}
}

// Send leases the booking's ticket email, sends it and only then marks it sent
// (at least once: a worker dying between send and mark sends again after the
// lease, but never twice at the same time, E-ML2). A failed send backs off;
// after the last try it is given up with one audit row (E-ML1).
func (s *ticketEmailService) Send(ctx context.Context, bookingID string) (bool, error) {
	n, err := s.repo.ClaimEmail(ctx, bookingID)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	msg, err := s.compose(ctx, bookingID)
	if err == nil {
		err = s.mailer.Send(ctx, msg)
	}
	bg := context.WithoutCancel(ctx)
	if err == nil {
		if merr := s.repo.MarkEmailSent(bg, bookingID); merr != nil {
			return true, merr
		}
		return true, nil
	}

	attempts, rerr := s.repo.ReleaseEmailClaim(bg, bookingID)
	if rerr != nil {
		logger.Warn("ticket email retry not scheduled", logger.String("booking_id", bookingID), logger.Err(rerr))
	}
	if attempts < repository.MaxTicketEmailAttempts {
		logger.Warn("ticket email failed; will retry", logger.String("booking_id", bookingID),
			logger.Int("attempt", attempts), logger.Err(err))
		return false, err
	}
	logger.Error("ticket email given up; the tickets stay on the web", logger.String("booking_id", bookingID), logger.Err(err))
	if aerr := audit.In(bg, s.db, audit.Record{
		ActorRole:    "system",
		Action:       "email.ticket_failed",
		ResourceType: "booking",
		ResourceID:   bookingID,
		Outcome:      audit.OutcomeFailure,
		ErrorMessage: err.Error(),
	}); aerr != nil {
		logger.Warn("ticket email failure not audited", logger.Err(aerr))
	}
	return false, err
}

func (s *ticketEmailService) PendingIDs(ctx context.Context, limit int) ([]string, error) {
	return s.repo.PendingEmailIDs(ctx, limit)
}

type emailTicket struct {
	Seat     string
	SeatType string
	Code     string
	Price    string
	QR       template.URL
}

type emailView struct {
	FullName   string
	MovieTitle string
	HallName   string
	StartAt    string
	Total      string
	Tickets    []emailTicket
}

var ticketEmailTemplate = template.Must(template.New("tickets").Parse(`<!doctype html>
<html><body style="font-family:Arial,Helvetica,sans-serif;color:#1f2328;max-width:640px">
<h2 style="margin-bottom:4px">Vé xem phim: {{.MovieTitle}}</h2>
<p>Xin chào {{.FullName}},</p>
<p>Đơn của bạn đã được xác nhận.<br>
<b>Phòng:</b> {{.HallName}}<br>
<b>Suất chiếu:</b> {{.StartAt}}<br>
<b>Tổng tiền:</b> {{.Total}}</p>
<table cellpadding="8" style="border-collapse:collapse">
{{range .Tickets}}<tr style="border-top:1px solid #d0d7de">
<td><img src="{{.QR}}" width="160" height="160" alt="QR {{.Code}}"></td>
<td><b>Ghế {{.Seat}}</b> ({{.SeatType}})<br>Mã vé: <code>{{.Code}}</code><br>Giá: {{.Price}}</td>
</tr>{{end}}
</table>
<p>Vé luôn có trong mục <b>Vé của tôi</b>. Mỗi vé chỉ quét vào cửa được một lần.</p>
</body></html>`))

func (s *ticketEmailService) compose(ctx context.Context, bookingID string) (notify.Message, error) {
	header, err := s.repo.BookingHeader(ctx, bookingID)
	if err != nil {
		return notify.Message{}, err
	}
	if header == nil {
		return notify.Message{}, fmt.Errorf("booking %s not found", bookingID)
	}
	rows, err := s.repo.TicketRows(ctx, bookingID)
	if err != nil {
		return notify.Message{}, err
	}
	view := emailView{
		FullName:   header.FullName,
		MovieTitle: header.MovieTitle,
		HallName:   header.HallName,
		StartAt:    header.StartAt.In(s.location).Format("15:04 02/01/2006"),
		Total:      formatVND(header.TotalAmount),
	}
	for _, t := range rows {
		png, err := qrcode.Encode(t.Code, qrcode.Medium, 256)
		if err != nil {
			return notify.Message{}, fmt.Errorf("render QR: %w", err)
		}
		view.Tickets = append(view.Tickets, emailTicket{
			Seat:     dto.SeatLabel(t.RowLabel, t.ColNumber),
			SeatType: t.SeatType,
			Code:     t.Code,
			Price:    formatVND(t.Price),
			QR:       template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png)),
		})
	}
	var body bytes.Buffer
	if err := ticketEmailTemplate.Execute(&body, view); err != nil {
		return notify.Message{}, fmt.Errorf("render ticket email: %w", err)
	}
	return notify.Message{
		To:      header.Email,
		Subject: "Vé xem phim " + header.MovieTitle,
		HTML:    body.String(),
	}, nil
}

// formatVND renders 120000 as "120.000 ₫".
func formatVND(amount int64) string {
	digits := strconv.FormatInt(amount, 10)
	var out []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, d)
	}
	return string(out) + " ₫"
}
