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

	"github.com/gorilla/mux"

	"github.com/google/uuid"
	"github.com/paypay3/kakeibo-app-api/todo-rest-service/domain/model"
	"github.com/paypay3/kakeibo-app-api/todo-rest-service/testutil"
)

type MockCategoriesName struct {
	BigCategoryName    model.NullString `json:"big_category_name"`
	MediumCategoryName model.NullString `json:"medium_category_name"`
	CustomCategoryName model.NullString `json:"custom_category_name"`
}

type MockShoppingListRepository struct{}

func (m MockShoppingListRepository) PostShoppingItem(shoppingItem *model.ShoppingItem, userID string) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (m MockShoppingListRepository) PutShoppingItem(shoppingItem *model.ShoppingItem) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (m MockShoppingListRepository) GetShoppingItem(shoppingItemID int) (model.ShoppingItem, error) {
	if shoppingItemID == 2 {
		return model.ShoppingItem{
			ID:                    2,
			PostedDate:            time.Date(2020, 12, 13, 16, 0, 0, 0, time.UTC),
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
			TransactionAutoAdd:    true,
			RelatedTransactionData: &model.TransactionData{
				ID: model.NullInt64{NullInt64: sql.NullInt64{Int64: 1, Valid: true}},
			},
		}, nil
	}

	return model.ShoppingItem{
		ID:                     1,
		PostedDate:             time.Date(2020, 12, 13, 16, 0, 0, 0, time.UTC),
		UpdatedDate:            time.Date(2020, 12, 13, 16, 0, 0, 0, time.UTC),
		ExpectedPurchaseDate:   model.Date{Time: time.Date(2020, 12, 15, 0, 0, 0, 0, time.UTC)},
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
		TransactionAutoAdd:     true,
		RelatedTransactionData: nil,
	}, nil
}

func TestDBHandler_PostShoppingItem(t *testing.T) {
	if err := os.Setenv("ACCOUNT_HOST", "localhost"); err != nil {
		t.Fatalf("unexpected error by os.Setenv() '%#v'", err)
	}

	accountHost := os.Getenv("ACCOUNT_HOST")
	accountHostURL := fmt.Sprintf("%s:8081", accountHost)

	mockGetCategoriesName := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	router.HandleFunc("/categories/names", mockGetCategoriesName).Methods("GET")

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
		AuthRepo:         MockAuthRepository{},
		ShoppingListRepo: MockShoppingListRepository{},
	}

	r := httptest.NewRequest("POST", "/shopping-list", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	cookie := &http.Cookie{
		Name:  "session_id",
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PostShoppingItem(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusCreated)
	testutil.AssertResponseBody(t, res, &model.ShoppingItem{}, &model.ShoppingItem{})
}

func TestDBHandler_PutShoppingItem(t *testing.T) {
	if err := os.Setenv("ACCOUNT_HOST", "localhost"); err != nil {
		t.Fatalf("unexpected error by os.Setenv() '%#v'", err)
	}

	accountHost := os.Getenv("ACCOUNT_HOST")
	accountHostURL := fmt.Sprintf("%s:8081", accountHost)

	mockPostTransaction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockTransaction := model.TransactionData{
			ID:                 model.NullInt64{NullInt64: sql.NullInt64{Int64: 1, Valid: true}},
			TransactionType:    "expense",
			PostedDate:         time.Date(2020, 12, 15, 16, 0, 0, 0, time.UTC),
			UpdatedDate:        time.Date(2020, 12, 15, 16, 0, 0, 0, time.UTC),
			TransactionDate:    "2020/12/15(火)",
			Shop:               model.NullString{NullString: sql.NullString{String: "コストコ", Valid: true}},
			Memo:               model.NullString{NullString: sql.NullString{String: "【買い物リスト】鶏肉3kg", Valid: true}},
			Amount:             1000,
			BigCategoryID:      2,
			BigCategoryName:    "食費",
			MediumCategoryID:   model.NullInt64{NullInt64: sql.NullInt64{Int64: 6, Valid: true}},
			MediumCategoryName: model.NullString{NullString: sql.NullString{String: "食料品", Valid: true}},
			CustomCategoryID:   model.NullInt64{NullInt64: sql.NullInt64{Int64: 0, Valid: false}},
			CustomCategoryName: model.NullString{NullString: sql.NullString{String: "", Valid: false}},
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&mockTransaction); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	mockDeleteTransaction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := mux.NewRouter()
	router.HandleFunc("/transactions", mockPostTransaction).Methods("POST")
	router.HandleFunc("/transactions/{id:[0-9]+}", mockDeleteTransaction).Methods("DELETE")

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
		AuthRepo:         MockAuthRepository{},
		ShoppingListRepo: MockShoppingListRepository{},
	}

	r := httptest.NewRequest("PUT", "/shopping-list/2", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"id": "2",
	})

	cookie := &http.Cookie{
		Name:  "session_id",
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PutShoppingItem(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &model.ShoppingItem{}, &model.ShoppingItem{})
}
