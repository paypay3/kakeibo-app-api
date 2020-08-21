package repository

import (
	"database/sql"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/model"
)

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
	GetApprovedGroupList(userID string) ([]model.ApprovedGroup, error)
	GetUnApprovedGroupList(userID string) ([]model.UnapprovedGroup, error)
	GetApprovedUsersList(approvedGroupIDList []interface{}) ([]model.ApprovedUser, error)
	GetUnapprovedUsersList(unapprovedGroupIDList []interface{}) ([]model.UnapprovedUser, error)
	GetGroup(groupID int) (*model.Group, error)
	PutGroup(group *model.Group, groupID int) error
	PostGroupAndApprovedUser(group *model.Group, userID string) (sql.Result, error)
	DeleteGroupAndApprovedUser(groupID int, userID string) error
	PostUnapprovedUser(unapprovedUser *model.UnapprovedUser, groupID int) (sql.Result, error)
	GetUnapprovedUser(groupUnapprovedUsersID int) (*model.UnapprovedUser, error)
	FindApprovedUser(groupID int, userID string) error
	FindUnapprovedUser(groupID int, userID string) error
	PostGroupApprovedUserAndDeleteGroupUnapprovedUser(groupID int, userID string) (sql.Result, error)
	GetApprovedUser(approvedUsersID int) (*model.ApprovedUser, error)
	DeleteGroupUnapprovedUser(groupID int, userID string) error
	DeleteGroupApprovedUser(groupID int, userID string) error
}
