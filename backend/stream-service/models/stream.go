package models

import "time"

// Stream represents a stream entity in the system.
type Stream struct {
	ID          int64      `json:"id" db:"id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	StartTime   time.Time  `json:"start_time" db:"start_time"`
	EndTime     time.Time  `json:"end_time" db:"end_time"`
	StreamKey   string     `json:"stream_key" db:"stream_key"`
	Resolution  string     `json:"resolution" db:"resolution"`
	Bitrate     int        `json:"bitrate" db:"bitrate"`
	FrameRate   int        `json:"frame_rate" db:"frame_rate"`
	Codec       string     `json:"codec" db:"codec"`
	ViewCount   int        `json:"view_count" db:"view_count"`
	Protocol    string     `json:"protocol" db:"protocol"`
	Status      string     `json:"status" db:"status"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"` // Nullable
	UserID      int64      `json:"user_id" db:"user_id"`
}
