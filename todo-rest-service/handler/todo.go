package handler

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-playground/validator"

	"github.com/paypay3/kakeibo-app-api/todo-rest-service/domain/model"

	"github.com/gorilla/mux"

	"github.com/garyburd/redigo/redis"
)

type SearchQuery struct {
	DateType     string
	StartDate    string
	EndDate      string
	CompleteFlag string
	TodoContent  string
	Sort         string
	SortType     string
	Limit        string
	UserID       string
}

type NoContentMsg struct {
	Message string `json:"message"`
}

type DeleteTodoMsg struct {
	Message string `json:"message"`
}

type TodoValidationErrorMsg struct {
	Message []string `json:"message"`
}

func (e *TodoValidationErrorMsg) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	return string(b)
}

func validateTodo(todo *model.Todo) error {
	var todoValidationErrorMsg TodoValidationErrorMsg

	validate := validator.New()
	validate.RegisterCustomTypeFunc(validateValuer, model.Date{})
	validate.RegisterValidation("date", dateValidation)
	validate.RegisterValidation("blank", blankValidation)
	err := validate.Struct(todo)
	if err == nil {
		return nil
	}

	for _, err := range err.(validator.ValidationErrors) {
		var errorMessage string

		fieldName := err.Field()
		switch fieldName {
		case "ImplementationDate":
			errorMessage = "todo実施日を正しく選択してください。"
		case "DueDate":
			errorMessage = "todo期限日を正しく選択してください。"
		case "TodoContent":
			tagName := err.Tag()
			switch tagName {
			case "required":
				errorMessage = "内容が入力されていません。"
			case "max":
				errorMessage = "内容は100文字以内で入力してください"
			case "blank":
				errorMessage = "内容の文字列先頭か末尾に空白がないか確認してください。"
			}
		}
		todoValidationErrorMsg.Message = append(todoValidationErrorMsg.Message, errorMessage)
	}

	return &todoValidationErrorMsg
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

func dateValidation(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}

	stringDate := date.String()
	trimDate := strings.Trim(stringDate, "\"")[:10]

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

func blankValidation(fl validator.FieldLevel) bool {
	text := fl.Field().String()

	if strings.HasPrefix(text, " ") || strings.HasPrefix(text, "　") || strings.HasSuffix(text, " ") || strings.HasSuffix(text, "　") {
		return false
	}

	return true
}

func NewSearchQuery(urlQuery url.Values, userID string) (*SearchQuery, error) {
	startDate, err := generateStartDate(urlQuery.Get("start_date"))
	if err != nil {
		return nil, err
	}

	endDate, err := generateEndDate(urlQuery.Get("end_date"))
	if err != nil {
		return nil, err
	}

	return &SearchQuery{
		DateType:     urlQuery.Get("date_type"),
		StartDate:    startDate,
		EndDate:      endDate,
		CompleteFlag: urlQuery.Get("complete_flag"),
		TodoContent:  urlQuery.Get("todo_content"),
		Sort:         urlQuery.Get("sort"),
		SortType:     urlQuery.Get("sort_type"),
		Limit:        urlQuery.Get("limit"),
		UserID:       userID,
	}, nil
}

func generateStartDate(date string) (string, error) {
	if len(date) == 0 {
		return "", nil
	}

	startDate, err := time.Parse("2006-01-02", date[:10])
	if err != nil {
		return "", err
	}

	return startDate.String(), nil
}

func generateEndDate(date string) (string, error) {
	if len(date) == 0 {
		return "", nil
	}

	parseDate, err := time.Parse("2006-01-02", date[:10])
	if err != nil {
		return "", err
	}

	endDate := time.Date(parseDate.Year(), parseDate.Month(), parseDate.Day()+1, 0, 0, 0, 0, parseDate.Location()).Add(-1 * time.Second)

	return endDate.String(), nil
}

func generateSqlQuery(searchQuery *SearchQuery) (string, error) {
	query := `
        SELECT
            id,
            posted_date,
            implementation_date,
            due_date,
            todo_content,
            complete_flag
        FROM
            todo_list
        WHERE
            user_id = "{{ .UserID }}"

        {{ with $DateType := .DateType }}
        AND
            {{ $DateType }} >= "{{ $.StartDate }}"
        AND
            {{ $DateType }} <= "{{ $.EndDate }}"
        {{ else }}
        AND
            implementation_date >= "{{ .StartDate }}"
        AND
            implementation_date <= "{{ .EndDate }}"
        {{ end }}

        {{ with $CompleteFlag := .CompleteFlag }}
        AND
            complete_flag = {{ $CompleteFlag }}
        {{ end }}

        {{ with $TodoContent := .TodoContent }}
        AND
            todo_content
        LIKE
            "%{{ $TodoContent }}%"
        {{ end }}

        {{ with $Sort := .Sort}}
        ORDER BY
            {{ $Sort }}
        {{ else }}
        ORDER BY
            implementation_date
        {{ end }}

        {{ with $SortType := .SortType}}
        {{ $SortType }}
        {{ else }}
        ASC
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

func (h *DBHandler) GetDailyTodoList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	date, err := time.Parse("2006-01-02", mux.Vars(r)["date"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"日付を正しく指定してください。"}))
		return
	}

	implementationTodoList, err := h.DBRepo.GetDailyImplementationTodoList(date, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dueTodoList, err := h.DBRepo.GetDailyDueTodoList(date, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if len(implementationTodoList) == 0 && len(dueTodoList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&NoContentMsg{"今日実施予定todo、締切予定todoは登録されていません。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	todoList := model.NewTodoList(implementationTodoList, dueTodoList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&todoList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) GetMonthlyTodoList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	firstDay, err := time.Parse("2006-01", mux.Vars(r)["year_month"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"年月を正しく指定してください。"}))
		return
	}
	lastDay := time.Date(firstDay.Year(), firstDay.Month()+1, 1, 0, 0, 0, 0, firstDay.Location()).Add(-1 * time.Second)

	implementationTodoList, err := h.DBRepo.GetMonthlyImplementationTodoList(firstDay, lastDay, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dueTodoList, err := h.DBRepo.GetMonthlyDueTodoList(firstDay, lastDay, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if len(implementationTodoList) == 0 && len(dueTodoList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&NoContentMsg{"当月実施予定todoは登録されていません。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	todoList := model.NewTodoList(implementationTodoList, dueTodoList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&todoList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PostTodo(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var todo model.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateTodo(&todo); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	result, err := h.DBRepo.PostTodo(&todo, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbTodo, err := h.DBRepo.GetTodo(int(lastInsertId))
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dbTodo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PutTodo(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var todo model.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateTodo(&todo); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	todoID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"todo ID を正しく指定してください。"}))
		return
	}

	if err := h.DBRepo.PutTodo(&todo, todoID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbTodo, err := h.DBRepo.GetTodo(todoID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dbTodo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	todoID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"todo ID を正しく指定してください。"}))
		return
	}

	if err := h.DBRepo.DeleteTodo(todoID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&DeleteTodoMsg{"todoを削除しました。"}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) SearchTodoList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	urlQuery := r.URL.Query()

	searchQuery, err := NewSearchQuery(urlQuery, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"日付を正しく指定してください。"}))
		return
	}

	query, err := generateSqlQuery(searchQuery)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbSearchTodoList, err := h.DBRepo.SearchTodoList(query)

	if len(dbSearchTodoList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&NoContentMsg{"条件に一致するtodoは見つかりませんでした。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	searchTodoList := model.NewSearchTodoList(dbSearchTodoList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&searchTodoList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
