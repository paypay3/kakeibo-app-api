package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/paypay3/kakeibo-app-api/acount-rest-service/domain/repository"
)

type DBHandler struct {
	DBRepo repository.DBRepository
}

type HTTPError struct {
	Status       int   `json:"status"`
	ErrorMessage error `json:"error"`
}

type BadRequestErrorMsg struct {
	Message string `json:"message"`
}

type AuthenticationErrorMsg struct {
	Message string `json:"message"`
}

type ConflictErrorMsg struct {
	Message string `json:"message"`
}

type InternalServerErrorMsg struct {
	Message string `json:"message"`
}

func NewDBHandler(DBRepo repository.DBRepository) *DBHandler {
	DBHandler := DBHandler{DBRepo: DBRepo}
	return &DBHandler
}

func NewHTTPError(status int, err error) error {
	switch status {
	case http.StatusBadRequest:
		switch err := err.(type) {
		case *CustomCategoryValidationErrorMsg:
			return &HTTPError{
				Status:       status,
				ErrorMessage: err,
			}
		case *TransactionValidationErrorMsg:
			return &HTTPError{
				Status:       status,
				ErrorMessage: err,
			}
		default:
			return &HTTPError{
				Status:       status,
				ErrorMessage: &BadRequestErrorMsg{"トランザクションを取得できませんでした。"},
			}
		}
	case http.StatusConflict:
		return &HTTPError{
			Status:       status,
			ErrorMessage: err.(*ConflictErrorMsg),
		}
	case http.StatusUnauthorized:
		return &HTTPError{
			Status:       status,
			ErrorMessage: &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"},
		}
	default:
		return &HTTPError{
			Status:       status,
			ErrorMessage: &InternalServerErrorMsg{"500 Internal Server Error"},
		}
	}
}

func (e *HTTPError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	return string(b)
}

func (e *BadRequestErrorMsg) Error() string {
	return e.Message
}

func (e *AuthenticationErrorMsg) Error() string {
	return e.Message
}

func (e *ConflictErrorMsg) Error() string {
	return e.Message
}

func (e *InternalServerErrorMsg) Error() string {
	return e.Message
}

func errorResponseByJSON(w http.ResponseWriter, err error) {
	httpError, ok := err.(*HTTPError)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpError.Status)
	if err := json.NewEncoder(w).Encode(httpError); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func verifySessionID(h *DBHandler, w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return "", err
	}
	sessionID := cookie.Value
	userID, err := h.DBRepo.GetUserID(sessionID)
	if err != nil {
		return "", err
	}
	return userID, nil
}