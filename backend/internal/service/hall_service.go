package service

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"gorm.io/gorm"
)

type HallService interface {
	List(ctx context.Context, query dto.PageQuery) ([]dto.HallResponse, int64, error)
	GetByID(ctx context.Context, id string) (*dto.HallResponse, error)
	SeatsByHall(ctx context.Context, hallID string) ([]models.Seat, error)
	PricesByHall(ctx context.Context, hallID string) ([]models.HallPrice, error)
	Create(ctx context.Context, req dto.HallRequest) (*dto.HallResponse, error)
	UpdateSeat(ctx context.Context, hallID, seatID string, req dto.SeatUpdateRequest) (*dto.SeatResponse, error)
	SetPrices(ctx context.Context, hallID string, req dto.PriceRequest) ([]dto.HallPriceResponse, error)
}

type hallService struct {
	db       *gorm.DB
	hallRepo *repository.HallRepository
}

func NewHallService(db *gorm.DB, hallRepo *repository.HallRepository) HallService {
	return &hallService{db: db, hallRepo: hallRepo}
}

func (s *hallService) List(ctx context.Context, query dto.PageQuery) ([]dto.HallResponse, int64, error) {
	halls, total, err := s.hallRepo.List(ctx, query.Search, query.PageSize, query.Offset())
	if err != nil {
		return nil, 0, err
	}
	return dto.NewHallResponses(halls), total, nil
}

func (s *hallService) GetByID(ctx context.Context, id string) (*dto.HallResponse, error) {
	hall, err := s.hallRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if hall == nil {
		return nil, apperrors.ErrHallNotFound
	}
	result := dto.NewHallResponse(hall)
	return &result, nil
}

func (s *hallService) SeatsByHall(ctx context.Context, hallID string) ([]models.Seat, error) {
	if _, err := s.GetByID(ctx, hallID); err != nil {
		return nil, err
	}
	return s.hallRepo.SeatsByHall(ctx, hallID)
}

func (s *hallService) PricesByHall(ctx context.Context, hallID string) ([]models.HallPrice, error) {
	if _, err := s.GetByID(ctx, hallID); err != nil {
		return nil, err
	}
	return s.hallRepo.PricesByHall(ctx, hallID)
}

func (s *hallService) Create(ctx context.Context, req dto.HallRequest) (*dto.HallResponse, error) {
	seats, err := generateSeats(req.Rows, req.SeatsPerRow, req.SeatTypes, req.Gaps, req.Spans)
	if err != nil {
		return nil, err
	}
	if err := validatePrices(req.Prices); err != nil {
		return nil, err
	}

	hall := &models.Hall{
		Name:        strings.TrimSpace(req.Name),
		Rows:        req.Rows,
		SeatsPerRow: req.SeatsPerRow,
		SeatTypes:   req.SeatTypes,
		Gaps:        req.Gaps,
	}
	// Nil would be stored as JSON null in the not-null jsonb columns.
	if hall.Gaps == nil {
		hall.Gaps = []string{}
	}
	if hall.SeatTypes == nil {
		hall.SeatTypes = map[string][]string{}
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.CreateHall(tx, hall); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrHallNameExists
			}
			return err
		}
		for i := range seats {
			seats[i].HallID = hall.ID
		}
		if err := s.hallRepo.CreateSeats(tx, seats); err != nil {
			return err
		}
		if err := s.hallRepo.UpsertPrices(tx, hall.ID, req.Prices); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hall.ID
			rec.After = map[string]any{"name": hall.Name, "rows": hall.Rows, "seats_per_row": hall.SeatsPerRow}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewHallResponse(hall)
	return &result, nil
}

func (s *hallService) UpdateSeat(ctx context.Context, hallID, seatID string, req dto.SeatUpdateRequest) (*dto.SeatResponse, error) {
	var seat *models.Seat
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock first, check after: an in-flight hold share-locks its showtime, so it commits
		// and is seen by the check; a later hold sees the new layout.
		if err := s.hallRepo.LockSchedule(tx, hallID); err != nil {
			return err
		}
		var err error
		seat, err = s.hallRepo.FindSeat(ctx, hallID, seatID)
		if err != nil {
			return err
		}
		if seat == nil {
			return apperrors.ErrSeatNotFound
		}
		before := map[string]any{"seat_type": seat.SeatType, "is_gap": seat.IsGap}

		hasBookings, err := s.hallRepo.HallHasBookings(tx, hallID)
		if err != nil {
			return err
		}
		if hasBookings {
			return apperrors.ErrHallHasBookings
		}
		if req.SeatType != "" {
			seat.SeatType = req.SeatType
		}
		if req.IsGap != nil {
			seat.IsGap = *req.IsGap
		}
		if err := s.hallRepo.UpdateSeat(tx, seat); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = seat.ID
			rec.Before = before
			rec.After = map[string]any{"seat_type": seat.SeatType, "is_gap": seat.IsGap}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewSeatResponse(seat)
	return &result, nil
}

func (s *hallService) SetPrices(ctx context.Context, hallID string, req dto.PriceRequest) ([]dto.HallPriceResponse, error) {
	if _, err := s.GetByID(ctx, hallID); err != nil {
		return nil, err
	}
	if err := validatePrices(req.Prices); err != nil {
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.UpsertPrices(tx, hallID, req.Prices); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hallID
			rec.After = map[string]any{"prices": req.Prices}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	response := make([]dto.HallPriceResponse, 0, len(req.Prices))
	for _, seatType := range models.AllSeatTypes {
		response = append(response, dto.HallPriceResponse{SeatType: seatType, Price: req.Prices[seatType]})
	}
	return response, nil
}

func generateSeats(rows, seatsPerRow int, seatTypes map[string][]string, gaps, spans []string) ([]models.Seat, error) {
	rowTypes, err := rowTypeMap(seatTypes, rows)
	if err != nil {
		return nil, err
	}

	gapSet := make(map[string]bool, len(gaps))
	for _, g := range gaps {
		row, col, err := parseGapLabel(g, rows, seatsPerRow)
		if err != nil {
			return nil, err
		}
		gapSet[dto.SeatLabel(row, col)] = true
	}

	spanSet, consumeSet, err := spanSet(spans, rows, seatsPerRow, gapSet)
	if err != nil {
		return nil, err
	}

	seats := make([]models.Seat, 0, rows*seatsPerRow)
	for r := 1; r <= rows; r++ {
		rowLabel := dto.RowLabel(r)
		for c := 1; c <= seatsPerRow; c++ {
			label := dto.SeatLabel(rowLabel, c)
			if consumeSet[label] {
				// The column is the right half of a 2-column seat anchored at c-1.
				continue
			}
			seatType := rowTypes[r]
			if seatType == "" {
				seatType = models.SeatStandard
			}
			colSpan := 1
			col := c
			if spanSet[label] {
				colSpan = 2
				c++ // the neighbor column is consumed by this seat
			}
			seats = append(seats, models.Seat{
				HallID:    "",
				RowIndex:  r,
				RowLabel:  rowLabel,
				ColNumber: col,
				SeatType:  seatType,
				IsGap:     gapSet[label],
				ColSpan:   colSpan,
			})
		}
	}
	return seats, nil
}

// A span anchor D3 makes one seat covering columns D3-D4; a col_span=2 seat
// swallows row c+1, which must exist and must not be a gap or another anchor.
func spanSet(spans []string, rows, seatsPerRow int, gapSet map[string]bool) (span, consumed map[string]bool, err error) {
	span = make(map[string]bool, len(spans))
	consumed = make(map[string]bool, len(spans))
	for _, s := range spans {
		row, col, parseErr := parseGapLabel(s, rows, seatsPerRow)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if col == seatsPerRow {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "no room for a second column"})
		}
		label := dto.SeatLabel(row, col)
		if span[label] || consumed[label] {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "duplicate anchor"})
		}
		if gapSet[label] {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "gap cannot span"})
		}
		span[label] = true
		consumed[dto.SeatLabel(row, col+1)] = true
	}
	return span, consumed, nil
}

func rowTypeMap(seatTypes map[string][]string, rows int) (map[int]string, error) {
	res := make(map[int]string, len(seatTypes))
	for seatType, list := range seatTypes {
		if !slices.Contains(models.AllSeatTypes, seatType) {
			return nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"seat_type": seatType})
		}
		for _, r := range list {
			n, err := strconv.Atoi(strings.TrimSpace(r))
			if err != nil || n < 1 || n > rows {
				return nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"row": r})
			}
			if prev, dup := res[n]; dup && prev != seatType {
				return nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"row": r})
			}
			res[n] = seatType
		}
	}
	return res, nil
}

func parseGapLabel(label string, rows, seatsPerRow int) (string, int, error) {
	i := 0
	for i < len(label) && (label[i] >= 'A' && label[i] <= 'Z' || label[i] >= 'a' && label[i] <= 'z') {
		i++
	}
	if i == 0 || i == len(label) {
		return "", 0, apperrors.ErrSeatValidation.WithDetails(map[string]string{"gap": label})
	}
	row := dto.RowNumber(label[:i])
	col, err := strconv.Atoi(label[i:])
	if err != nil || row < 1 || row > rows || col < 1 || col > seatsPerRow {
		return "", 0, apperrors.ErrSeatValidation.WithDetails(map[string]string{"gap": label})
	}
	return strings.ToUpper(label[:i]), col, nil
}

func validatePrices(prices map[string]int64) error {
	if len(prices) != len(models.AllSeatTypes) {
		return apperrors.ErrSeatValidation
	}
	for _, seatType := range models.AllSeatTypes {
		price, ok := prices[seatType]
		if !ok || price <= 0 {
			return apperrors.ErrSeatValidation.WithDetails(map[string]string{"seat_type": seatType})
		}
	}
	return nil
}
