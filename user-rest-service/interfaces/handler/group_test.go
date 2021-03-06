package handler

import (
	"database/sql"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/config"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/model"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/repository"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/testutil"
)

type MockGroupRepository struct{}

type MockUserRepositoryForGroup struct {
	repository.UserRepository
}

type MockSqlResult struct {
	sql.Result
}

func (t MockUserRepositoryForGroup) FindSignUpUserByUserID(userID string) (*model.SignUpUser, error) {
	signUpUser := &model.SignUpUser{
		ID:       "testID",
		Name:     "testName",
		Email:    "test@icloud.com",
		Password: "$2a$10$teJL.9I0QfBESpaBIwlbl.VkivuHEOKhy674CW6J.4k3AnfEpcYLy",
	}

	return signUpUser, nil
}

func (r MockSqlResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (t MockGroupRepository) GetApprovedGroupList(userID string) ([]model.ApprovedGroup, error) {
	return []model.ApprovedGroup{
		{GroupID: 1, GroupName: "group1", ApprovedUsersList: make([]model.ApprovedUser, 0), UnapprovedUsersList: make([]model.UnapprovedUser, 0)},
		{GroupID: 2, GroupName: "group2", ApprovedUsersList: make([]model.ApprovedUser, 0), UnapprovedUsersList: make([]model.UnapprovedUser, 0)},
		{GroupID: 3, GroupName: "group3", ApprovedUsersList: make([]model.ApprovedUser, 0), UnapprovedUsersList: make([]model.UnapprovedUser, 0)},
	}, nil
}

func (t MockGroupRepository) GetUnApprovedGroupList(userID string) ([]model.UnapprovedGroup, error) {
	return []model.UnapprovedGroup{
		{GroupID: 4, GroupName: "group4", ApprovedUsersList: make([]model.ApprovedUser, 0), UnapprovedUsersList: make([]model.UnapprovedUser, 0)},
		{GroupID: 5, GroupName: "group5", ApprovedUsersList: make([]model.ApprovedUser, 0), UnapprovedUsersList: make([]model.UnapprovedUser, 0)},
	}, nil
}

func (t MockGroupRepository) GetApprovedUsersList(approvedGroupIDList []interface{}) ([]model.ApprovedUser, error) {
	return []model.ApprovedUser{
		{GroupID: 1, UserID: "userID1", UserName: "userName1", ColorCode: "#FF0000"},
		{GroupID: 1, UserID: "userID2", UserName: "userName2", ColorCode: "#00FFFF"},
		{GroupID: 2, UserID: "userID1", UserName: "userName1", ColorCode: "#FF0000"},
		{GroupID: 3, UserID: "userID1", UserName: "userName1", ColorCode: "#FF0000"},
		{GroupID: 3, UserID: "userID2", UserName: "userName2", ColorCode: "#00FFFF"},
		{GroupID: 4, UserID: "userID2", UserName: "userName2", ColorCode: "#FF0000"},
		{GroupID: 4, UserID: "userID4", UserName: "userName4", ColorCode: "#00FFFF"},
		{GroupID: 5, UserID: "userID4", UserName: "userName4", ColorCode: "#FF0000"},
	}, nil
}

func (t MockGroupRepository) GetUnapprovedUsersList(unapprovedGroupIDList []interface{}) ([]model.UnapprovedUser, error) {
	return []model.UnapprovedUser{
		{GroupID: 1, UserID: "userID3", UserName: "userName3"},
		{GroupID: 2, UserID: "userID3", UserName: "userName3"},
		{GroupID: 2, UserID: "userID4", UserName: "userName4"},
		{GroupID: 4, UserID: "userID1", UserName: "userName1"},
		{GroupID: 4, UserID: "userID3", UserName: "userName3"},
		{GroupID: 5, UserID: "userID1", UserName: "userName1"},
	}, nil
}

func (t MockGroupRepository) GetGroup(groupID int) (*model.Group, error) {
	return &model.Group{
		GroupID:   1,
		GroupName: "group1",
	}, nil
}

func (t MockGroupRepository) PutGroup(group *model.Group, groupID int) error {
	return nil
}

func (t MockGroupRepository) PostGroupAndApprovedUser(group *model.Group, userID string) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (t MockGroupRepository) DeleteGroupAndApprovedUser(groupID int, userID string) error {
	return nil
}

func (t MockGroupRepository) PostUnapprovedUser(unapprovedUser *model.UnapprovedUser, groupID int) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (t MockGroupRepository) GetUnapprovedUser(groupUnapprovedUsersID int) (*model.UnapprovedUser, error) {
	return &model.UnapprovedUser{
		GroupID:  1,
		UserID:   "userID2",
		UserName: "userName2",
	}, nil
}

func (t MockGroupRepository) FindApprovedUser(groupID int, userID string) error {
	if groupID == 1 {
		return sql.ErrNoRows
	}

	return nil
}

func (t MockGroupRepository) FindUnapprovedUser(groupID int, userID string) error {
	if groupID == 1 {
		return sql.ErrNoRows
	}

	return nil
}

func (t MockGroupRepository) PostGroupApprovedUserAndDeleteGroupUnapprovedUser(groupID int, userID string, colorCode string) (sql.Result, error) {
	return MockSqlResult{}, nil
}

func (t MockGroupRepository) GetApprovedUser(approvedUsersID int) (*model.ApprovedUser, error) {
	return &model.ApprovedUser{
		GroupID:   2,
		UserID:    "userID1",
		UserName:  "userName1",
		ColorCode: "#FF0000",
	}, nil
}

func (t MockGroupRepository) DeleteGroupApprovedUser(groupID int, userID string) error {
	return nil
}

func (t MockGroupRepository) DeleteGroupUnapprovedUser(groupID int, userID string) error {
	return nil
}

func (t MockGroupRepository) FindApprovedUsersList(groupID int, groupUsersList []string) (model.GroupTasksUsersListReceiver, error) {
	return model.GroupTasksUsersListReceiver{
		GroupUsersList: []string{
			"userID4",
			"userID5",
			"userID6",
		},
	}, nil
}

func (t MockGroupRepository) GetGroupUsersList(groupID int) ([]string, error) {
	return []string{"userID1", "userID4", "userID5", "userID3", "userID2"}, nil
}

func TestDBHandler_GetGroupList(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("GET", "/groups", nil)
	w := httptest.NewRecorder()

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.GetGroupList(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &model.GroupList{}, &model.GroupList{})
}

func TestDBHandler_PostGroup(t *testing.T) {
	if err := os.Setenv("ACCOUNT_HOST", "localhost"); err != nil {
		t.Fatalf("unexpected error by os.Setenv() '%#v'", err)
	}

	postInitGroupStandardBudgetsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		t.Fatalf("unexpected error by net.Listen() '%#v'", err)
	}

	ts := httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: postInitGroupStandardBudgetsHandler},
	}
	ts.Start()
	defer ts.Close()

	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("POST", "/groups", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PostGroup(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusCreated)
	testutil.AssertResponseBody(t, res, &model.Group{}, &model.Group{})
}

func TestDBHandler_PutGroup(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("PUT", "/groups/1", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "1",
	})

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PutGroup(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &model.Group{}, &model.Group{})
}

func TestDBHandler_PostGroupUnapprovedUser(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		UserRepo:  MockUserRepositoryForGroup{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("POST", "/groups/1/users", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "1",
	})

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PostGroupUnapprovedUser(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusCreated)
	testutil.AssertResponseBody(t, res, &model.UnapprovedUser{}, &model.UnapprovedUser{})
}

func TestDBHandler_DeleteGroupApprovedUser(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("DELETE", "/groups/2/users", nil)
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
	})

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.DeleteGroupApprovedUser(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &DeleteContentMsg{}, &DeleteContentMsg{})
}

func TestDBHandler_PostGroupApprovedUser(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("POST", "/groups/2/users/approved", nil)
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
	})

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.PostGroupApprovedUser(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusCreated)
	testutil.AssertResponseBody(t, res, &model.ApprovedUser{}, &model.ApprovedUser{})
}

func TestDBHandler_DeleteGroupUnapprovedUser(t *testing.T) {
	h := DBHandler{
		AuthRepo:  MockAuthRepository{},
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("DELETE", "/groups/2/users/unapproved", nil)
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
	})

	cookie := &http.Cookie{
		Name:  config.Env.Cookie.Name,
		Value: uuid.New().String(),
	}

	r.AddCookie(cookie)

	h.DeleteGroupUnapprovedUser(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &DeleteContentMsg{}, &DeleteContentMsg{})
}

func TestDBHandler_VerifyGroupAffiliation(t *testing.T) {
	h := DBHandler{
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("GET", "/groups/2/users/userID1/verify", nil)
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
		"user_id":  "userID1",
	})

	h.VerifyGroupAffiliation(w, r)

	res := w.Result()
	defer res.Body.Close()

	if diff := cmp.Diff(http.StatusOK, res.StatusCode); len(diff) != 0 {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

func TestDBHandler_VerifyGroupAffiliationOfUsersList(t *testing.T) {
	h := DBHandler{
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("GET", "/groups/2/users/verify", strings.NewReader(testutil.GetRequestJsonFromTestData(t)))
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
	})

	h.VerifyGroupAffiliationOfUsersList(w, r)

	res := w.Result()
	defer res.Body.Close()

	if diff := cmp.Diff(http.StatusOK, res.StatusCode); len(diff) != 0 {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

func TestDBHandler_GetGroupUserIDList(t *testing.T) {
	h := DBHandler{
		GroupRepo: MockGroupRepository{},
	}

	r := httptest.NewRequest("GET", "/groups/2/users", nil)
	w := httptest.NewRecorder()

	r = mux.SetURLVars(r, map[string]string{
		"group_id": "2",
	})

	h.GetGroupUserIDList(w, r)

	res := w.Result()
	defer res.Body.Close()

	testutil.AssertResponseHeader(t, res, http.StatusOK)
	testutil.AssertResponseBody(t, res, &[]string{}, &[]string{})
}
