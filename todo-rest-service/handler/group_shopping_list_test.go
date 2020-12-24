package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/paypay3/kakeibo-app-api/todo-rest-service/domain/model"
	"github.com/paypay3/kakeibo-app-api/todo-rest-service/testutil"
)

type MockGroupShoppingListRepository struct{}

func (m MockGroupShoppingListRepository) GetGroupShoppingItem(groupShoppingItemID int) (model.GroupShoppingItem, error) {
	if groupShoppingItemID == 2 {
		return model.GroupShoppingItem{
			ID:                    1,
			PostedDate:            time.Date(2020, 12, 14, 16, 0, 0, 0, time.UTC),
			UpdatedDate:           time.Date(2020, 12, 15, 16, 0, 0, 0, time.UTC),
			ExpectedPurchaseDate:  model.Date{Time: time.Date(2020, 12, 15, 0, 0, 0, 0, time.UTC)},
			CompleteFlag:          true,
			Purchase:              "鶏肉3kg",
			Shop:                  model.NullString{NullString: sql.NullString{String: "コストコ", Valid: true}},
			Amount:                model.NullInt64{NullInt64: sql.NullInt64{Int64: 1000, Valid: true}},
			BigCategoryID:         2,
			BigCategoryName:       "",
			MediumCategoryID:      model.NullInt64{NullInt64: sql.NullInt64{Int64: 6, Valid: true}},
			MediumCategoryName:    model.NullString{NullString: sql.NullString{String: "", Valid: false}},
			CustomCategoryID:      model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
			CustomCategoryName:    model.NullString{NullString: sql.NullString{String: "", Valid: false}},
			RegularShoppingListID: model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
			PaymentUserID:         model.NullString{NullString: sql.NullString{String: "userID1", Valid: true}},
			TransactionAutoAdd:    true,
			RelatedTransactionData: &model.GroupTransactionData{
				ID: model.NullInt64{NullInt64: sql.NullInt64{Int64: 1, Valid: true}},
			},
		}, nil
	}

	return model.GroupShoppingItem{
		ID:                     1,
		PostedDate:             time.Date(2020, 12, 24, 16, 0, 0, 0, time.UTC),
		UpdatedDate:            time.Date(2020, 12, 24, 16, 0, 0, 0, time.UTC),
		ExpectedPurchaseDate:   model.Date{Time: time.Date(2020, 12, 25, 0, 0, 0, 0, time.UTC)},
		CompleteFlag:           false,
		Purchase:               "鶏肉3kg",
		Shop:                   model.NullString{NullString: sql.NullString{String: "コストコ", Valid: true}},
		Amount:                 model.NullInt64{NullInt64: sql.NullInt64{Int64: 1000, Valid: true}},
		BigCategoryID:          2,
		BigCategoryName:        "",
		MediumCategoryID:       model.NullInt64{NullInt64: sql.NullInt64{Int64: 6, Valid: true}},
		MediumCategoryName:     model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		CustomCategoryID:       model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
		CustomCategoryName:     model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		RegularShoppingListID:  model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
		PaymentUserID:          model.NullString{NullString: sql.NullString{String: "userID1", Valid: true}},
		TransactionAutoAdd:     true,
		RelatedTransactionData: nil,
	}, nil
}

func (m MockGroupShoppingListRepository) PostGroupShoppingItem(groupShoppingItem *model.GroupShoppingItem, groupID int) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (m MockGroupShoppingListRepository) PutGroupShoppingItem(groupShoppingItem *model.GroupShoppingItem) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func TestDBHandler_PostGroupShoppingItem(t *testing.T) {
	if err := os.Setenv("ACCOUNT_HOST", "localhost"); err != nil {
		t.Fatalf("unexpected error by os.Setenv() '%#v'", err)
	}

	accountHost := os.Getenv("ACCOUNT_HOST")
	accountHostURL := fmt.Sprintf("%s:8081", accountHost)

	mockGetGroupShoppingItemCategoriesName := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockCategoriesName := MockCategoriesName{
			BigCategoryName:    model.NullString{NullString: sql.NullString{String: "食費", Valid: true}},
			MediumCategoryName: model.NullString{NullString: sql.NullString{String: "食料品", Valid: true}},
			CustomCategoryName: model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&mockCategoriesName); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	router := mux.NewRouter()
	router.HandleFunc("/groups/{group_id:[0-9]+}/categories/name", mockGetGroupShoppingItemCategoriesName).Methods("GET")

	listener, err := net.Listen("tcp", accountHostURL)
	if err != nil {
		t.Fatalf("unexpected error by net.Listen() '%#v'", err)
	}

	ts := httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: router},
	}

	ts.Start()
	defer ts.Close()

	h := DBHandler{
		AuthRepo:              MockAuthRepository{},
		GroupShoppingListRepo: MockGroupShoppingListRepository{},
	}

	r := httptest.NewRequest("POST", "/groups/1/shopping-list", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "1",
	})

	cookie := &http.Cookie{
		Name:  "session_id",
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PostGroupShoppingItem(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusCreated)
	testutil.AssertResponseBody(t, res, &model.GroupShoppingItem{}, &model.GroupShoppingItem{})
}

func TestDBHandler_PutGroupShoppingItem(t *testing.T) {
	if err := os.Setenv("ACCOUNT_HOST", "localhost"); err != nil {
		t.Fatalf("unexpected error by os.Setenv() '%#v'", err)
	}

	accountHost := os.Getenv("ACCOUNT_HOST")
	accountHostURL := fmt.Sprintf("%s:8081", accountHost)

	mockPostGroupTransaction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockGroupTransaction := model.GroupTransactionData{
			ID:                 model.NullInt64{NullInt64: sql.NullInt64{Int64: 1, Valid: true}},
			TransactionType:    "expense",
			PostedDate:         time.Date(2020, 12, 15, 16, 0, 0, 0, time.UTC),
			UpdatedDate:        time.Date(2020, 12, 15, 16, 0, 0, 0, time.UTC),
			TransactionDate:    "2020/12/15(火)",
			Shop:               model.NullString{NullString: sql.NullString{String: "コストコ", Valid: true}},
			Memo:               model.NullString{NullString: sql.NullString{String: "【買い物リスト】鶏肉3kg", Valid: true}},
			Amount:             1000,
			PostedUserID:       "userID1",
			UpdatedUserID:      model.NullString{NullString: sql.NullString{String: "", Valid: false}},
			PaymentUserID:      "userID1",
			BigCategoryID:      2,
			BigCategoryName:    "食費",
			MediumCategoryID:   model.NullInt64{NullInt64: sql.NullInt64{Int64: 6, Valid: true}},
			MediumCategoryName: model.NullString{NullString: sql.NullString{String: "食料品", Valid: true}},
			CustomCategoryID:   model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
			CustomCategoryName: model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&mockGroupTransaction); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	mockDeleteGroupTransaction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mockGetGroupShoppingItemCategoriesName := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockCategoriesName := MockCategoriesName{
			BigCategoryName:    model.NullString{NullString: sql.NullString{String: "食費", Valid: true}},
			MediumCategoryName: model.NullString{NullString: sql.NullString{String: "食料品", Valid: true}},
			CustomCategoryName: model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&mockCategoriesName); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	router := mux.NewRouter()
	router.HandleFunc("/groups/{group_id:[0-9]+}/transactions", mockPostGroupTransaction).Methods("POST")
	router.HandleFunc("/groups/{group_id:[0-9]+}/transactions/{id:[0-9]+}", mockDeleteGroupTransaction).Methods("DELETE")
	router.HandleFunc("/groups/{group_id:[0-9]+}/categories/name", mockGetGroupShoppingItemCategoriesName).Methods("GET")

	listener, err := net.Listen("tcp", accountHostURL)
	if err != nil {
		t.Fatalf("unexpected error by net.Listen() '%#v'", err)
	}

	ts := httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: router},
	}

	ts.Start()
	defer ts.Close()

	h := DBHandler{
		AuthRepo:              MockAuthRepository{},
		GroupShoppingListRepo: MockGroupShoppingListRepository{},
	}

	r := httptest.NewRequest("PUT", "/groups/1/shopping-list/2", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "1",
		"id":       "2",
	})

	cookie := &http.Cookie{
		Name:  "session_id",
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PutGroupShoppingItem(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &model.GroupShoppingItem{}, &model.GroupShoppingItem{})
}
