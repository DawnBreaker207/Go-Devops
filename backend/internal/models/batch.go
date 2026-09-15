package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	BatchRunning = "running"
	BatchSuccess = "success"
	BatchFailed  = "failed"
	BatchSkipped = "skipped"
	BatchStopped = "stopped"
)

const (
	TriggerCron    = "cron"
	TriggerManual  = "manual"
	TriggerConfirm = "confirm"
)

type BatchJob struct {
	ID            string     `gorm:"type:uuid;primaryKey" json:"id"`
	JobName       string     `gorm:"type:varchar(128);not null" json:"job_name"`
	TriggeredBy   string     `gorm:"type:varchar(32);not null;default:cron" json:"triggered_by"`
	Status        string     `gorm:"type:varchar(16);not null;default:running" json:"status"`
	ProcessedRows int        `gorm:"not null;default:0" json:"processed_rows"`
	SkippedRows   int        `gorm:"not null;default:0" json:"skipped_rows"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (BatchJob) TableName() string { return "batch_jobs" }

func (j *BatchJob) BeforeCreate(*gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	return nil
}

type DailyAggregate struct {
	ID            string         `gorm:"type:uuid;primaryKey" json:"id"`
	ReportDate    time.Time      `gorm:"type:date;not null;uniqueIndex" json:"report_date"`
	TotalRevenue  int64          `gorm:"not null;default:0" json:"total_revenue"`
	TicketsSold   int            `gorm:"not null;default:0" json:"tickets_sold"`
	SeatsSold     int            `gorm:"not null;default:0" json:"seats_sold"`
	Capacity      int            `gorm:"not null;default:0" json:"capacity"`
	OccupancyRate float64        `gorm:"not null;default:0" json:"occupancy_rate"`
	Breakdown     map[string]any `gorm:"serializer:json;type:jsonb;not null" json:"breakdown"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (DailyAggregate) TableName() string { return "daily_aggregates" }

func (a *DailyAggregate) BeforeCreate(*gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
