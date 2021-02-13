package infrastructure

import (
	"fmt"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/model"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/infrastructure/persistence/db"
)

type UserRepository struct {
	*db.RedisHandler
	*db.MySQLHandler
}

func NewUserRepository(redisHandler *db.RedisHandler, mysqlHandler *db.MySQLHandler) *UserRepository {
	return &UserRepository{redisHandler, mysqlHandler}
}

func (r *UserRepository) FindSignUpUserByUserID(userID string) (*model.SignUpUser, error) {
	query := `
        SELECT
            user_id
        FROM
            users
        WHERE 
            user_id = ?`

	var user *model.SignUpUser
	if err := r.MySQLHandler.Conn.QueryRowx(query, userID).StructScan(&user); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetUser(userID string) (*model.LoginUser, error) {
	query := `
        SELECT
            user_id,
            name,
            email
        FROM 
            users
        WHERE
            user_id = ?`

	var user model.LoginUser
	if err := r.MySQLHandler.Conn.QueryRowx(query, userID).StructScan(&user); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) AddSessionID(sessionID string, loginUserID string, expiration int) error {
	conn := r.RedisHandler.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", sessionID, loginUserID, "EX", expiration)

	return err
}

func (r *UserRepository) DeleteSessionID(sessionID string) error {
	conn := r.RedisHandler.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", sessionID)

	return err
}
