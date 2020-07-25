package handler

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-playground/validator"

	"github.com/gorilla/mux"

	"github.com/paypay3/kakeibo-app-api/acount-rest-service/domain/model"

	"github.com/garyburd/redigo/redis"
)

type SearchQuery struct {
	TransactionType string
	BigCategoryID   string
	Shop            string
	Memo            string
	LowAmount       string
	HighAmount      string
	StartDate       string
	EndDate         string
	Sort            string
	SortType        string
	Limit           string
	UserID          string
}

type NoSearchContentMsg struct {
	Message string `json:"message"`
}

type DeleteTransactionMsg struct {
	Message string `json:"message"`
}

type TransactionValidationErrorMsg struct {
	Message []string `json:"message"`
}

func (e *TransactionValidationErrorMsg) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	return string(b)
}

func validateTransaction(transactionReceiver *model.TransactionReceiver) error {
	var transactionValidationErrorMsg TransactionValidationErrorMsg

	validate := validator.New()
	validate.RegisterCustomTypeFunc(validateValuer, model.Date{}, model.NullString{}, model.NullInt64{})
	validate.RegisterValidation("blank", blankValidation)
	validate.RegisterValidation("date", dateValidation)
	validate.RegisterValidation("either_id", eitherIDValidation)
	err := validate.Struct(transactionReceiver)
	if err == nil {
		return nil
	}

	for _, err := range err.(validator.ValidationErrors) {
		var errorMessage string

		fieldName := err.Field()
		switch fieldName {
		case "TransactionType":
			tagName := err.Tag()
			switch tagName {
			case "required":
				errorMessage = "取引タイプが選択されていません。"
			case "oneof":
				errorMessage = "取引タイプを正しく選択してください。"
			}
		case "TransactionDate":
			errorMessage = "日付を正しく選択してください。"
		case "Shop":
			tagName := err.Tag()
			switch tagName {
			case "max":
				errorMessage = "店名は20文字以内で入力してください。"
			case "blank":
				errorMessage = "店名の文字列先頭か末尾に空白がないか確認してください。"
			}
		case "Memo":
			tagName := err.Tag()
			switch tagName {
			case "max":
				errorMessage = "メモは50文字以内で入力してください"
			case "blank":
				errorMessage = "メモの文字列先頭か末尾に空白がないか確認してください。"
			}
		case "Amount":
			errorMessage = "金額が入力されていません。"
		case "BigCategoryID":
			tagName := err.Tag()
			switch tagName {
			case "required":
				errorMessage = "カテゴリーが選択されていません。"
			case "min", "max":
				errorMessage = "カテゴリーを正しく選択してください。"
			case "either_id":
				errorMessage = "中カテゴリーを正しく選択してください。"
			}
		case "MediumCategoryID":
			errorMessage = "中カテゴリーを正しく選択してください。"
		case "CustomCategoryID":
			errorMessage = "中カテゴリーを正しく選択してください。"
		}
		transactionValidationErrorMsg.Message = append(transactionValidationErrorMsg.Message, errorMessage)
	}

	return &transactionValidationErrorMsg
}

func validateValuer(field reflect.Value) interface{} {
	if valuer, ok := field.Interface().(driver.Valuer); ok {
		val, err := valuer.Value()
		if err == nil {
			return val
		}
	}
	return nil
}

func blankValidation(fl validator.FieldLevel) bool {
	text := fl.Field().String()

	if strings.HasPrefix(text, " ") || strings.HasPrefix(text, "　") || strings.HasSuffix(text, " ") || strings.HasSuffix(text, "　") {
		return false
	}

	return true
}

func dateValidation(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}

	stringDate := date.String()
	trimDate := strings.Trim(string(stringDate), "\"")[:10]

	minDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	dateTime, err := time.Parse("2006-01-02", trimDate)
	if err != nil {
		return false
	}
	if dateTime.Before(minDate) || dateTime.After(maxDate) {
		return false
	}

	return true
}

func eitherIDValidation(fl validator.FieldLevel) bool {
	transactionReceiver, ok := fl.Parent().Interface().(*model.TransactionReceiver)
	if !ok {
		return false
	}

	if transactionReceiver.MediumCategoryID.Valid && transactionReceiver.CustomCategoryID.Valid {
		return false
	}

	if transactionReceiver.CustomCategoryID.Valid {
		return true
	}

	if transactionReceiver.MediumCategoryID.Valid {
		return true
	}

	return false
}

func NewSearchQuery(urlQuery url.Values, userID string) SearchQuery {
	startDate := trimDate(urlQuery.Get("start_date"))
	endDate := trimDate(urlQuery.Get("end_date"))

	return SearchQuery{
		TransactionType: urlQuery.Get("transaction_type"),
		BigCategoryID:   urlQuery.Get("big_category_id"),
		Shop:            urlQuery.Get("shop"),
		Memo:            urlQuery.Get("memo"),
		LowAmount:       urlQuery.Get("low_amount"),
		HighAmount:      urlQuery.Get("high_amount"),
		StartDate:       startDate,
		EndDate:         endDate,
		Sort:            urlQuery.Get("sort"),
		SortType:        urlQuery.Get("sort_type"),
		Limit:           urlQuery.Get("limit"),
		UserID:          userID,
	}
}

func trimDate(date string) string {
	if len(date) == 0 {
		return ""
	}

	return date[:10]
}

func generateSqlQuery(searchQuery SearchQuery) (string, error) {
	query := `
        SELECT
            transactions.id id,
            transactions.transaction_type transaction_type,
            transactions.updated_date updated_date,
            transactions.transaction_date transaction_date,
            transactions.shop shop,
            transactions.memo memo,
            transactions.amount amount,
            big_categories.category_name big_category_name,
            medium_categories.category_name medium_category_name,
            custom_categories.category_name custom_category_name
        FROM
            transactions
        LEFT JOIN
            big_categories
        ON
            transactions.big_category_id = big_categories.id
        LEFT JOIN
            medium_categories
        ON
            transactions.medium_category_id = medium_categories.id
        LEFT JOIN
            custom_categories
        ON
            transactions.custom_category_id = custom_categories.id
        WHERE
            transactions.user_id = "{{.UserID}}"

        {{ with $StartDate := .StartDate }}
        AND
            transactions.transaction_date >= "{{ $StartDate }}"
        {{ end }}

        {{ with $EndDate := .EndDate }}
        AND
            transactions.transaction_date <= "{{ $EndDate }}"
        {{ end }}

        {{ with $TransactionType := .TransactionType }}
        AND
            transactions.transaction_type = "{{ $TransactionType }}"
        {{ end }}

        {{ with $BigCategoryID := .BigCategoryID }}
        AND
            transactions.big_category_id = "{{ $BigCategoryID }}"
        {{ end }}

        {{ with $LowAmount := .LowAmount }}
        AND
            transactions.amount >= "{{ $LowAmount }}"
        {{ end }}

        {{ with $HighAmount := .HighAmount }}
        AND
            transactions.amount <= "{{ $HighAmount }}"
        {{ end }}

        {{ with $Shop := .Shop }}
        AND
            transactions.shop
        LIKE
            "%{{ $Shop }}%"
        {{ end }}

        {{ with $Memo := .Memo }}
        AND
            transactions.memo
        LIKE
            "%{{ $Memo }}%"
        {{ end }}

        {{ with $Sort := .Sort}}
        ORDER BY
            transactions.{{ $Sort }}
        {{ else }}
        ORDER BY
            transactions.transaction_date
        {{ end }}

        {{ with $SortType := .SortType}}
        {{ $SortType }}
        {{ else }}
        DESC
        {{ end }}

        {{ with $Limit := .Limit}}
        LIMIT
        {{ $Limit }}
        {{ end }}`

	var buffer bytes.Buffer
	queryTemplate, err := template.New("queryTemplate").Parse(query)
	if err != nil {
		return "", err
	}

	if err := queryTemplate.Execute(&buffer, searchQuery); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func (h *DBHandler) GetMonthlyTransactionsList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	firstDay, err := time.Parse("2006-01", mux.Vars(r)["month"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}
	lastDay := time.Date(firstDay.Year(), firstDay.Month()+1, 1, 0, 0, 0, 0, firstDay.Location()).Add(-1 * time.Second)

	dbTransactionsList, err := h.DBRepo.GetMonthlyTransactionsList(userID, firstDay, lastDay)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if len(dbTransactionsList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&NoSearchContentMsg{"条件に一致する取引履歴は見つかりませんでした。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	transactionsList := model.NewTransactionsList(dbTransactionsList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&transactionsList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PostTransaction(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var transactionReceiver model.TransactionReceiver
	if err := json.NewDecoder(r.Body).Decode(&transactionReceiver); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateTransaction(&transactionReceiver); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	result, err := h.DBRepo.PostTransaction(&transactionReceiver, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var transactionSender model.TransactionSender
	dbTransactionSender, err := h.DBRepo.GetTransaction(&transactionSender, int(lastInsertId))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dbTransactionSender); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PutTransaction(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var transactionReceiver model.TransactionReceiver
	if err := json.NewDecoder(r.Body).Decode(&transactionReceiver); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	transactionID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := h.DBRepo.PutTransaction(&transactionReceiver, transactionID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var transactionSender model.TransactionSender
	dbTransactionSender, err := h.DBRepo.GetTransaction(&transactionSender, transactionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dbTransactionSender); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	transactionID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := h.DBRepo.DeleteTransaction(transactionID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&DeleteTransactionMsg{"トランザクションを削除しました。"}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) SearchTransactionsList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, nil))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	urlQuery := r.URL.Query()

	searchQuery := NewSearchQuery(urlQuery, userID)

	query, err := generateSqlQuery(searchQuery)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbTransactionsList, err := h.DBRepo.SearchTransactionsList(query)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if len(dbTransactionsList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&NoSearchContentMsg{"条件に一致する取引履歴は見つかりませんでした。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	transactionsList := model.NewTransactionsList(dbTransactionsList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&transactionsList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}