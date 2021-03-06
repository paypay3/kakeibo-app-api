package handler

import (
	"database/sql"
	"time"
)

type MockAuthRepository struct{}

type MockSqlResult struct {
	sql.Result
}

type MockTime struct{}

func (t MockAuthRepository) GetUserID(sessionID string) (string, error) {
	return "userID1", nil
}

func (r MockSqlResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (m MockTime) Now() time.Time {
	return time.Date(2020, 11, 1, 0, 0, 0, 0, time.UTC)
}
