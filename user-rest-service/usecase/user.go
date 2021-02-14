package usecase

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/xerrors"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/apierrors"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/userdomain"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/vo"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/interfaces/presenter"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/usecase/input"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/usecase/interfaces"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/usecase/output"
)

type UserUsecase interface {
	SignUp(in *input.SignUpUser) (*output.SignUpUser, error)
	Login(in *input.LoginUser) (*output.LoginUser, error)
}

type userUsecase struct {
	userRepository userdomain.Repository
	accountApi     interfaces.AccountApi
}

func NewUserUsecase(userRepository userdomain.Repository, accountApi interfaces.AccountApi) *userUsecase {
	return &userUsecase{
		userRepository: userRepository,
		accountApi:     accountApi,
	}
}

func (u *userUsecase) SignUp(in *input.SignUpUser) (*output.SignUpUser, error) {
	var userValidationError presenter.ValidationError

	userID, err := userdomain.NewUserID(in.UserID)
	if err != nil {
		userValidationError.UserID = "ユーザーIDを正しく入力してください"
	}

	name, err := userdomain.NewName(in.Name)
	if err != nil {
		userValidationError.Name = "名前を正しく入力してください"
	}

	email, err := vo.NewEmail(in.Email)
	if err != nil {
		userValidationError.Email = "メールアドレスを正しく入力してください"
	}

	password, err := vo.NewPassword(in.Password)
	if err != nil {
		if xerrors.Is(err, vo.ErrInvalidPassword) {
			userValidationError.Password = "パスワードを正しく入力してください"
		} else {
			return nil, err
		}
	}

	if !userValidationError.IsEmpty() {
		return nil, apierrors.NewBadRequestError(&userValidationError)
	}

	signUpUser := userdomain.NewSignUpUser(userID, name, email, password)

	if err := checkForUniqueUser(u, signUpUser); err != nil {
		return nil, err
	}

	if err := u.userRepository.CreateSignUpUser(signUpUser); err != nil {
		return nil, err
	}

	if err := u.accountApi.PostInitStandardBudgets(signUpUser.UserID()); err != nil {
		if err := u.userRepository.DeleteSignUpUser(signUpUser); err != nil {
			return nil, err
		}

		return nil, err
	}

	return &output.SignUpUser{
		UserID: signUpUser.UserID().Value(),
		Name:   signUpUser.Name().Value(),
		Email:  signUpUser.Email().Value(),
	}, nil
}
func (u *userUsecase) Login(in *input.LoginUser) (*output.LoginUser, error) {
	var userValidationError presenter.ValidationError

	email, err := vo.NewEmail(in.Email)
	if err != nil {
		userValidationError.Email = "メールアドレスを正しく入力してください"
	}

	password, err := vo.NewPassword(in.Password)
	if err != nil {
		if xerrors.Is(err, vo.ErrInvalidPassword) {
			userValidationError.Password = "パスワードを正しく入力してください"
		} else {
			return nil, err
		}
	}

	if !userValidationError.IsEmpty() {
		return nil, apierrors.NewBadRequestError(&userValidationError)
	}

	loginUser, err := userdomain.NewLoginUser(email, password)
	if err != nil {
		return nil, apierrors.NewBadRequestError(err)
	}

	dbLoginUser, err := u.userRepository.FindLoginUserByEmail(loginUser.Email())
	if err != nil {
		var notFoundError *apierrors.NotFoundError
		if xerrors.As(err, &notFoundError) {
			return nil, apierrors.NewAuthenticationError(apierrors.NewErrorString("認証に失敗しました"))
		}

		return nil, err
	}

	hashPassword := dbLoginUser.Password().Value()

	if err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(in.Password)); err != nil {
		return nil, apierrors.NewAuthenticationError(apierrors.NewErrorString("認証に失敗しました"))
	}

	sessionID := uuid.New().String()
	expiration := 86400 * 30

	if err := u.userRepository.AddSessionID(sessionID, dbLoginUser.UserID(), expiration); err != nil {
		return nil, err
	}

	return &output.LoginUser{
		UserID:    dbLoginUser.UserID().Value(),
		Name:      dbLoginUser.Name().Value(),
		Email:     dbLoginUser.Email().Value(),
		SessionID: sessionID,
		Expires:   time.Now().Add(time.Duration(expiration) * time.Second),
	}, nil
}

func checkForUniqueUser(u *userUsecase, signUpUser *userdomain.SignUpUser) error {
	var notFoundError *apierrors.NotFoundError

	_, errUserID := u.userRepository.FindSignUpUserByUserID(signUpUser.UserID())
	if errUserID != nil && !xerrors.As(errUserID, &notFoundError) {
		return errUserID
	}

	_, errEmail := u.userRepository.FindSignUpUserByEmail(signUpUser.Email())
	if errEmail != nil && !xerrors.As(errEmail, &notFoundError) {
		return errEmail
	}

	existsUserByUserID := !xerrors.As(errUserID, &notFoundError)
	existsUserByEmail := !xerrors.As(errEmail, &notFoundError)

	if !existsUserByUserID && !existsUserByEmail {
		return nil
	}

	var userConflictError presenter.ConflictError

	if existsUserByUserID {
		userConflictError.UserID = "このユーザーIDは既に利用されています"
	}

	if existsUserByEmail {
		userConflictError.Email = "このメールアドレスは既に利用されています"
	}

	return apierrors.NewConflictError(&userConflictError)
}
