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
	FindUserID(userID string) error
	FindEmail(email string) error
	CreateUser(user *model.SignUpUser) error
	DeleteUser(signUpUser *model.SignUpUser) error
	FindUser(user *model.LoginUser) (*model.LoginUser, error)
	SetSessionID(sessionID string, loginUserID string, expiration int) error
	DeleteSessionID(sessionID string) error
}

type GroupRepository interface {
	GetGroupList(userID string) ([]model.Group, error)
	GetGroupUsersList(groupList []model.Group) ([]model.GroupUser, error)
	GetGroupUnapprovedUsersList(groupList []model.Group) ([]model.GroupUnapprovedUser, error)
	GetGroup(groupID int) (*model.Group, error)
	PostGroupAndGroupUser(group *model.Group, userID string) (sql.Result, error)
	DeleteGroupAndGroupUser(groupID int, userID string) error
	PutGroup(group *model.Group, groupID int) error
	PostGroupUnapprovedUser(groupUnapprovedUser *model.GroupUnapprovedUser, groupID int) (sql.Result, error)
	GetGroupUnapprovedUser(groupUnapprovedUsersID int) (*model.GroupUnapprovedUser, error)
	FindGroupUser(groupID int, userID string) error
	FindGroupUnapprovedUser(groupID int, userID string) error
}
