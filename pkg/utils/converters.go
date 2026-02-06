package utils

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func FromPgUUID(u pgtype.UUID) uuid.UUID {
	if !u.Valid {
		return uuid.Nil
	}
	return uuid.UUID(u.Bytes)
}

func ToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func FromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func FromPgInt32(i pgtype.Int4) int32 {
	if !i.Valid {
		return 0
	}
	return i.Int32
}

func FromPgBool(b pgtype.Bool) bool {
	if !b.Valid {
		return false
	}
	return b.Bool
}

func FromPgTimestamptz(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	if ts.InfinityModifier != pgtype.Finite {
		return time.Time{}
	}
	return ts.Time
}

// func ToPgTimestamptz(time.Time) pgtype.Timestamptz {

// }