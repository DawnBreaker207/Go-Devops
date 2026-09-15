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
	BulkUpdateSeats(ctx context.Context, hallID string, req dto.BulkSeatUpdateRequest) ([]dto.SeatResponse, error)
	SetPrices(ctx context.Context, hallID string, req dto.PriceRequest) ([]dto.HallPriceResponse, error)
	Clone(ctx context.Context, hallID string, req dto.CloneHallRequest) (*dto.HallResponse, error)
	UpdateHall(ctx context.Context, hallID string, req dto.UpdateHallRequest) (*dto.HallResponse, error)
	RegenerateLayout(ctx context.Context, hallID string, req dto.HallRequest) (*dto.HallResponse, error)
	DeleteHall(ctx context.Context, hallID string) error
	Templates() []dto.HallTemplateResponse
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

// hallTemplate is a built-in starting layout an admin can pick instead of
// typing out rows/seat_types/gaps by hand.
type hallTemplate struct {
	rows, seatsPerRow int
	seatTypes         map[string][]string
	gaps              []string
	spans             []string
	aisleAfterCols    []int
	screenPosition    string
}

// hallTemplates: small (~60 seats, no frills), medium (~120, a VIP block and
// a couple row), large (~200, VIP + couple + a center aisle).
var hallTemplates = map[string]hallTemplate{
	"small": {
		rows: 6, seatsPerRow: 10,
		screenPosition: models.ScreenFront,
	},
	"medium": {
		rows: 10, seatsPerRow: 12,
		seatTypes:      map[string][]string{"vip": {"6", "7"}},
		spans:          []string{"J2", "J4", "J6", "J8", "J10"},
		aisleAfterCols: []int{6},
		screenPosition: models.ScreenFront,
	},
	"large": {
		rows: 14, seatsPerRow: 14,
		seatTypes:      map[string][]string{"vip": {"6", "7", "8"}, "recliner": {"1", "2"}},
		spans:          []string{"N2", "N4", "N6", "N8", "N10", "N12"},
		aisleAfterCols: []int{4, 10},
		screenPosition: models.ScreenFront,
	},
}

// Templates previews the built-in layouts (name, size, seats per type) so an
// admin can pick one before creating a hall.
func (s *hallService) Templates() []dto.HallTemplateResponse {
	names := make([]string, 0, len(hallTemplates))
	for name := range hallTemplates {
		names = append(names, name)
	}
	slices.Sort(names)

	out := make([]dto.HallTemplateResponse, 0, len(names))
	for _, name := range names {
		t := hallTemplates[name]
		seats, err := generateSeats(t.rows, t.seatsPerRow, t.seatTypes, t.gaps, t.spans)
		if err != nil {
			continue // a built-in template must be valid; skip rather than fail the whole list
		}
		byType := make(map[string]int, len(models.AllSeatTypes))
		total := 0
		for _, seat := range seats {
			if seat.IsGap {
				continue
			}
			byType[seat.SeatType]++
			total++
		}
		out = append(out, dto.HallTemplateResponse{
			Name: name, Rows: t.rows, SeatsPerRow: t.seatsPerRow, SeatCount: total, ByType: byType,
		})
	}
	return out
}

// resolveLayout fills unset fields of req from the named template; fields the
// request set explicitly are kept as is.
func resolveLayout(req dto.HallRequest) (dto.HallRequest, error) {
	if req.Template == "" {
		if req.Rows <= 0 || req.SeatsPerRow <= 0 {
			return req, apperrors.Validation("rows and seats_per_row are required without a template")
		}
		if req.ScreenPosition == "" {
			req.ScreenPosition = models.ScreenFront
		}
		return req, nil
	}
	t, ok := hallTemplates[req.Template]
	if !ok {
		return req, apperrors.Validation("unknown template").WithDetails(map[string]string{"template": req.Template})
	}
	if req.Rows <= 0 {
		req.Rows = t.rows
	}
	if req.SeatsPerRow <= 0 {
		req.SeatsPerRow = t.seatsPerRow
	}
	if len(req.SeatTypes) == 0 {
		req.SeatTypes = t.seatTypes
	}
	if len(req.Gaps) == 0 {
		req.Gaps = t.gaps
	}
	if len(req.Spans) == 0 {
		req.Spans = t.spans
	}
	if len(req.AisleAfterCols) == 0 {
		req.AisleAfterCols = t.aisleAfterCols
	}
	if req.ScreenPosition == "" {
		req.ScreenPosition = t.screenPosition
	}
	return req, nil
}

func (s *hallService) Create(ctx context.Context, req dto.HallRequest) (*dto.HallResponse, error) {
	req, err := resolveLayout(req)
	if err != nil {
		return nil, err
	}
	seats, err := generateSeats(req.Rows, req.SeatsPerRow, req.SeatTypes, req.Gaps, req.Spans)
	if err != nil {
		return nil, err
	}
	if err := validatePrices(req.Prices); err != nil {
		return nil, err
	}

	hall := &models.Hall{
		Name:           strings.TrimSpace(req.Name),
		Rows:           req.Rows,
		SeatsPerRow:    req.SeatsPerRow,
		SeatTypes:      req.SeatTypes,
		Gaps:           req.Gaps,
		ScreenPosition: req.ScreenPosition,
		AisleAfterCols: req.AisleAfterCols,
		Active:         true,
	}
	normalizeHallJSON(hall)

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
			rec.After = map[string]any{"name": hall.Name, "rows": hall.Rows, "seats_per_row": hall.SeatsPerRow, "template": req.Template}
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

// normalizeHallJSON: nil would be stored as JSON null in the not-null jsonb columns.
func normalizeHallJSON(hall *models.Hall) {
	if hall.Gaps == nil {
		hall.Gaps = []string{}
	}
	if hall.SeatTypes == nil {
		hall.SeatTypes = map[string][]string{}
	}
	if hall.AisleAfterCols == nil {
		hall.AisleAfterCols = []int{}
	}
}

// Clone copies a hall's actual current seat grid — including any manual bulk
// edits, not just its original template — under a new name.
func (s *hallService) Clone(ctx context.Context, hallID string, req dto.CloneHallRequest) (*dto.HallResponse, error) {
	source, err := s.hallRepo.FindByID(ctx, hallID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, apperrors.ErrHallNotFound
	}
	sourceSeats, err := s.hallRepo.SeatsByHall(ctx, hallID)
	if err != nil {
		return nil, err
	}
	var prices []models.HallPrice
	if req.CopyPrices {
		if prices, err = s.hallRepo.PricesByHall(ctx, hallID); err != nil {
			return nil, err
		}
	}

	clone := &models.Hall{
		Name:           strings.TrimSpace(req.Name),
		Rows:           source.Rows,
		SeatsPerRow:    source.SeatsPerRow,
		SeatTypes:      source.SeatTypes,
		Gaps:           source.Gaps,
		ScreenPosition: source.ScreenPosition,
		AisleAfterCols: source.AisleAfterCols,
		Active:         true,
	}
	normalizeHallJSON(clone)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.CreateHall(tx, clone); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrHallNameExists
			}
			return err
		}
		seats := make([]models.Seat, len(sourceSeats))
		for i, seat := range sourceSeats {
			seats[i] = models.Seat{
				HallID: clone.ID, RowIndex: seat.RowIndex, RowLabel: seat.RowLabel,
				ColNumber: seat.ColNumber, SeatType: seat.SeatType, IsGap: seat.IsGap, ColSpan: seat.ColSpan,
			}
		}
		if err := s.hallRepo.CreateSeats(tx, seats); err != nil {
			return err
		}
		if len(prices) > 0 {
			priceMap := make(map[string]int64, len(prices))
			for _, p := range prices {
				priceMap[p.SeatType] = p.Price
			}
			if err := s.hallRepo.UpsertPrices(tx, clone.ID, priceMap); err != nil {
				return err
			}
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = clone.ID
			rec.After = map[string]any{"name": clone.Name, "cloned_from": hallID, "copy_prices": req.CopyPrices}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewHallResponse(clone)
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

// BulkUpdateSeats applies every change of the request to the seats it selects,
// all in one transaction: one bad change rolls the whole batch back. It never
// creates or removes a span (col_span is untouched) — that only happens
// through RegenerateLayout.
func (s *hallService) BulkUpdateSeats(ctx context.Context, hallID string, req dto.BulkSeatUpdateRequest) ([]dto.SeatResponse, error) {
	var updated []models.Seat
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.LockSchedule(tx, hallID); err != nil {
			return err
		}
		hasBookings, err := s.hallRepo.HallHasBookings(tx, hallID)
		if err != nil {
			return err
		}
		if hasBookings {
			return apperrors.ErrHallHasBookings
		}
		seats, err := s.hallRepo.SeatsByHall(ctx, hallID)
		if err != nil {
			return err
		}
		if len(seats) == 0 {
			return apperrors.ErrHallNotFound
		}

		touched := map[string]*models.Seat{}
		for _, change := range req.Changes {
			matched, err := selectSeats(seats, change.Selector)
			if err != nil {
				return err
			}
			if len(matched) == 0 {
				return apperrors.ErrSeatValidation.WithDetails(map[string]string{"selector": "matched no seat"})
			}
			for _, idx := range matched {
				seat := &seats[idx]
				if change.SeatType != "" {
					seat.SeatType = change.SeatType
				}
				if change.IsGap != nil {
					seat.IsGap = *change.IsGap
				}
				touched[seat.ID] = seat
			}
		}
		for _, seat := range touched {
			if err := s.hallRepo.UpdateSeat(tx, seat); err != nil {
				return err
			}
			updated = append(updated, *seat)
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hallID
			rec.After = map[string]any{"changes": len(req.Changes), "seats_touched": len(touched)}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(updated, func(a, b models.Seat) int {
		if a.RowIndex != b.RowIndex {
			return a.RowIndex - b.RowIndex
		}
		return a.ColNumber - b.ColNumber
	})
	return dto.NewSeatResponses(updated), nil
}

// selectSeats resolves one selector (exactly one of labels/rows/cols/range) to
// indexes into seats.
func selectSeats(seats []models.Seat, sel dto.SeatSelector) ([]int, error) {
	set := 0
	if len(sel.Labels) > 0 {
		set++
	}
	if len(sel.Rows) > 0 {
		set++
	}
	if len(sel.Cols) > 0 {
		set++
	}
	if sel.Range != "" {
		set++
	}
	if set != 1 {
		return nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"selector": "exactly one of labels, rows, cols, range is required"})
	}

	switch {
	case len(sel.Labels) > 0:
		want := make(map[string]bool, len(sel.Labels))
		for _, l := range sel.Labels {
			want[strings.ToUpper(l)] = true
		}
		var idx []int
		for i, seat := range seats {
			if want[dto.SeatLabel(seat.RowLabel, seat.ColNumber)] {
				idx = append(idx, i)
			}
		}
		return idx, nil
	case len(sel.Rows) > 0:
		want := make(map[string]bool, len(sel.Rows))
		for _, r := range sel.Rows {
			want[strings.ToUpper(strings.TrimSpace(r))] = true
		}
		var idx []int
		for i, seat := range seats {
			if want[seat.RowLabel] {
				idx = append(idx, i)
			}
		}
		return idx, nil
	case len(sel.Cols) > 0:
		want := make(map[int]bool, len(sel.Cols))
		for _, c := range sel.Cols {
			want[c] = true
		}
		var idx []int
		for i, seat := range seats {
			if want[seat.ColNumber] {
				idx = append(idx, i)
			}
		}
		return idx, nil
	default:
		fromRow, fromCol, toRow, toCol, err := parseRange(sel.Range)
		if err != nil {
			return nil, err
		}
		var idx []int
		for i, seat := range seats {
			rowNum := dto.RowNumber(seat.RowLabel)
			if rowNum >= fromRow && rowNum <= toRow && seat.ColNumber >= fromCol && seat.ColNumber <= toCol {
				idx = append(idx, i)
			}
		}
		return idx, nil
	}
}

// parseRange parses "A1:C4" into its row/column bounds.
func parseRange(r string) (fromRow, fromCol, toRow, toCol int, err error) {
	parts := strings.SplitN(r, ":", 2)
	if len(parts) != 2 {
		return 0, 0, 0, 0, apperrors.ErrSeatValidation.WithDetails(map[string]string{"range": r})
	}
	fromLabel, fCol, err1 := splitLabel(parts[0])
	toLabel, tCol, err2 := splitLabel(parts[1])
	if err1 != nil || err2 != nil || fromLabel == "" || toLabel == "" {
		return 0, 0, 0, 0, apperrors.ErrSeatValidation.WithDetails(map[string]string{"range": r})
	}
	fromRow, toRow = dto.RowNumber(fromLabel), dto.RowNumber(toLabel)
	if fromRow == 0 || toRow == 0 || fromRow > toRow || fCol > tCol {
		return 0, 0, 0, 0, apperrors.ErrSeatValidation.WithDetails(map[string]string{"range": r})
	}
	return fromRow, fCol, toRow, tCol, nil
}

func splitLabel(label string) (string, int, error) {
	i := 0
	for i < len(label) && (label[i] >= 'A' && label[i] <= 'Z' || label[i] >= 'a' && label[i] <= 'z') {
		i++
	}
	if i == 0 || i == len(label) {
		return "", 0, apperrors.ErrSeatValidation
	}
	col, err := strconv.Atoi(label[i:])
	if err != nil || col < 1 {
		return "", 0, apperrors.ErrSeatValidation
	}
	return strings.ToUpper(label[:i]), col, nil
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

// UpdateHall changes name/screen/aisle/active. Deactivating is refused (409)
// while the hall still has an open showtime to come: those must be closed or
// rescheduled elsewhere first. Once inactive, showtime creation refuses the
// hall (H6-style guard in showtime_service).
func (s *hallService) UpdateHall(ctx context.Context, hallID string, req dto.UpdateHallRequest) (*dto.HallResponse, error) {
	var hall *models.Hall
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.LockSchedule(tx, hallID); err != nil {
			return err
		}
		current, err := s.hallRepo.FindByID(ctx, hallID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrHallNotFound
		}
		before := map[string]any{"name": current.Name, "active": current.Active}

		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.ScreenPosition != nil {
			current.ScreenPosition = *req.ScreenPosition
		}
		if req.AisleAfterCols != nil {
			current.AisleAfterCols = req.AisleAfterCols
		}
		if req.Active != nil && *req.Active != current.Active {
			if !*req.Active {
				open, err := s.hallRepo.HasOpenUpcomingShowtimes(tx, hallID)
				if err != nil {
					return err
				}
				if open {
					return apperrors.ErrHallStillSelling
				}
			}
			current.Active = *req.Active
		}
		normalizeHallJSON(current)
		if err := s.hallRepo.UpdateHall(tx, current); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrHallNameExists
			}
			return err
		}
		hall = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hallID
			rec.Before = before
			rec.After = map[string]any{"name": hall.Name, "active": hall.Active}
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

// RegenerateLayout replaces a hall's whole seat grid. Only legal while the
// hall has never had a single booking (any status, any showtime, even a
// deleted one): booking_seats/tickets keep a foreign key into showtime_seats,
// which points at seats, so a hall that ever sold or held anything can not
// have its seats deleted without violating that key.
func (s *hallService) RegenerateLayout(ctx context.Context, hallID string, req dto.HallRequest) (*dto.HallResponse, error) {
	req, err := resolveLayout(req)
	if err != nil {
		return nil, err
	}
	seats, err := generateSeats(req.Rows, req.SeatsPerRow, req.SeatTypes, req.Gaps, req.Spans)
	if err != nil {
		return nil, err
	}

	var hall *models.Hall
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.LockSchedule(tx, hallID); err != nil {
			return err
		}
		current, err := s.hallRepo.FindByID(ctx, hallID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrHallNotFound
		}
		everHad, err := s.hallRepo.HallEverHadBooking(tx, hallID)
		if err != nil {
			return err
		}
		if everHad {
			return apperrors.ErrHallEverHadBookings
		}

		// showtime_seats.seat_id would otherwise block deleting the old grid.
		if err := s.hallRepo.DeleteShowtimeSeatsByHall(tx, hallID); err != nil {
			return err
		}
		if err := s.hallRepo.DeleteSeats(tx, hallID); err != nil {
			return err
		}
		for i := range seats {
			seats[i].HallID = hallID
		}
		if err := s.hallRepo.CreateSeats(tx, seats); err != nil {
			return err
		}
		// Any showtime still open to come gets a fresh seat state on the new grid.
		if err := s.hallRepo.CreateShowtimeSeatsForHall(tx, hallID); err != nil {
			return err
		}

		current.Rows, current.SeatsPerRow = req.Rows, req.SeatsPerRow
		current.SeatTypes, current.Gaps = req.SeatTypes, req.Gaps
		current.ScreenPosition, current.AisleAfterCols = req.ScreenPosition, req.AisleAfterCols
		normalizeHallJSON(current)
		if err := s.hallRepo.UpdateHall(tx, current); err != nil {
			return err
		}
		hall = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hallID
			rec.After = map[string]any{"rows": req.Rows, "seats_per_row": req.SeatsPerRow, "seat_count": len(seats)}
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

// DeleteHall soft-deletes a hall; refused (409) while it has a showtime not
// yet ended, open or closed.
func (s *hallService) DeleteHall(ctx context.Context, hallID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hallRepo.LockSchedule(tx, hallID); err != nil {
			return err
		}
		current, err := s.hallRepo.FindByID(ctx, hallID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrHallNotFound
		}
		unfinished, err := s.hallRepo.HasUnfinishedShowtimes(tx, hallID)
		if err != nil {
			return err
		}
		if unfinished {
			return apperrors.ErrHallHasUpcomingShowtimes
		}
		if err := s.hallRepo.DeleteHall(tx, hallID); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = hallID
			rec.After = map[string]any{"deleted": true}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
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
		neighbor := dto.SeatLabel(row, col+1)
		if span[label] || consumed[label] {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "duplicate anchor"})
		}
		if gapSet[label] {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "gap cannot span"})
		}
		// The column this anchor would consume must be free too: it can not
		// already be another anchor, nor already consumed by one (checking
		// only the new anchor's own label misses this, e.g. spans ["A3","A2"]
		// processed in that order would otherwise silently drop seat A4).
		if span[neighbor] || consumed[neighbor] {
			return nil, nil, apperrors.ErrSeatValidation.WithDetails(map[string]string{"span": s, "reason": "overlapping span"})
		}
		span[label] = true
		consumed[neighbor] = true
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
