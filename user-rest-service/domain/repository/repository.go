package repository

import (
	"database/sql"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/model"
)

type DBRepository interface {
	AuthRepository
	UserRepository
	GroupRepository
}

type AuthRepository interface {
	GetUserID(sessionID string) (string, error)
}

type UserRepository interface {
	FindID(signUpUser *model.SignUpUser) error
	FindEmail(signUpUser *model.SignUpUser) error
	CreateUser(user *model.SignUpUser) error
	DeleteUser(signUpUser *model.SignUpUser) error
	FindUser(user *model.LoginUser) (*model.LoginUser, error)
	SetSessionID(sessionID string, loginUserID string, expiration int) error
	DeleteSessionID(sessionID string) error
}

type GroupRepository interface {
	GetGroup(groupID int) (*model.Group, error)
	PostGroup(group *model.Group) (sql.Result, error)
	DeleteGroup(groupID int) error
	PostGroupUser(groupID int, userID string) (sql.Result, error)
	DeleteGroupUser(groupID int, userID string) error
}
