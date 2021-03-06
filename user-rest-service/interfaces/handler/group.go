package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"golang.org/x/xerrors"

	"github.com/paypay3/kakeibo-app-api/user-rest-service/apierrors"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/domain/model"
)

type NoContentMsg struct {
	Message string `json:"message"`
}

func validateGroupName(groupName string) error {
	if strings.HasPrefix(groupName, " ") || strings.HasPrefix(groupName, "　") {
		return &BadRequestErrorMsg{"文字列先頭に空白がないか確認してください。"}
	}

	if strings.HasSuffix(groupName, " ") || strings.HasSuffix(groupName, "　") {
		return &BadRequestErrorMsg{"文字列末尾に空白がないか確認してください。"}
	}

	if utf8.RuneCountInString(groupName) == 0 || utf8.RuneCountInString(groupName) > 20 {
		return &BadRequestErrorMsg{"グループ名は1文字以上、20文字以内で入力してください。"}
	}

	return nil
}

func postInitGroupStandardBudgets(groupID int) error {
	accountHost := os.Getenv("ACCOUNT_HOST")
	requestURL := fmt.Sprintf("http://%s:8081/groups/%d/standard-budgets", accountHost, groupID)

	request, err := http.NewRequest(
		"POST",
		requestURL,
		nil,
	)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          500,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 60 * time.Second,
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer func() {
		_, _ = io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}()

	if response.StatusCode == http.StatusCreated {
		return nil
	}

	return xerrors.New("couldn't create a group standard budget")
}

func checkForUniqueGroupUser(h *DBHandler, groupID int, userID string) error {
	if err := h.GroupRepo.FindApprovedUser(groupID, userID); err != sql.ErrNoRows {
		if err == nil {
			return &ConflictErrorMsg{"こちらのユーザーは既にグループに参加しています。"}
		}

		return err
	}

	if err := h.GroupRepo.FindUnapprovedUser(groupID, userID); err != sql.ErrNoRows {
		if err == nil {
			return &ConflictErrorMsg{"こちらのユーザーは既にグループに招待しています。"}
		}

		return err
	}

	return nil
}

func validateUserID(userID string) error {
	if strings.HasPrefix(userID, " ") || strings.HasPrefix(userID, "　") {
		return &BadRequestErrorMsg{"文字列先頭に空白がないか確認してください。"}
	}

	if strings.HasSuffix(userID, " ") || strings.HasSuffix(userID, "　") {
		return &BadRequestErrorMsg{"文字列末尾に空白がないか確認してください。"}
	}

	if len(userID) == 0 || len(userID) > 10 {
		return &BadRequestErrorMsg{"ユーザーIDは1文字以上、10文字以内で入力してください。"}
	}

	return nil
}

func generateGroupIDList(approvedGroupList []model.ApprovedGroup, unapprovedGroupList []model.UnapprovedGroup) []interface{} {
	var groupIDList []interface{}

	for _, approvedGroup := range approvedGroupList {
		groupIDList = append(groupIDList, approvedGroup.GroupID)
	}

	for _, unapprovedGroup := range unapprovedGroupList {
		groupIDList = append(groupIDList, unapprovedGroup.GroupID)
	}

	return groupIDList
}

func assignColorCodeToUser(groupUserIDList []string) string {
	const (
		red                  = "#FF0000"
		cyan                 = "#00FFFF"
		chartreuseGreen      = "#80FF00"
		violet               = "#8000FF"
		orange               = "#FF8000"
		azure                = "#0080FF"
		emeraldGreen         = "#00FF80"
		rubyRed              = "#FF0080"
		yellow               = "#FFFF00"
		blue                 = "#0000FF"
		green                = "#00FF00"
		magenta              = "#FF00FF"
		goldenYellow         = "#FFBF00"
		cobaltBlue           = "#0040FF"
		brightYellowishGreen = "#BFFF00"
		hyacinth             = "#4000FF"
		cobaltGreen          = "#00FF40"
		reddishPurple        = "#FF00BF"
		leafGreen            = "#40FF00"
		purple               = "#BF00FF"
		vermilion            = "#FF4000"
		ceruleanBlue         = "#00BFFF"
		turquoiseGreen       = "#00FFBF"
		carmine              = "#FF0040"
	)

	colorCodeList := [24]string{
		red,
		cyan,
		chartreuseGreen,
		violet,
		orange,
		azure,
		emeraldGreen,
		rubyRed,
		yellow,
		blue,
		green,
		magenta,
		goldenYellow,
		cobaltBlue,
		brightYellowishGreen,
		hyacinth,
		cobaltGreen,
		reddishPurple,
		leafGreen,
		purple,
		vermilion,
		ceruleanBlue,
		turquoiseGreen,
		carmine,
	}

	idx := len(groupUserIDList) % len(colorCodeList)

	colorCode := colorCodeList[idx]

	return colorCode
}

func (h *DBHandler) GetGroupList(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	approvedGroupList, err := h.GroupRepo.GetApprovedGroupList(userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	unapprovedGroupList, err := h.GroupRepo.GetUnApprovedGroupList(userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if len(approvedGroupList) == 0 && len(unapprovedGroupList) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(&NoContentMsg{"参加しているグループ、招待されているグループはありません。"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	groupIDList := generateGroupIDList(approvedGroupList, unapprovedGroupList)

	approvedUsersList, err := h.GroupRepo.GetApprovedUsersList(groupIDList)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	unapprovedUsersList, err := h.GroupRepo.GetUnapprovedUsersList(groupIDList)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	for i := 0; i < len(approvedGroupList); i++ {
		for _, approvedUser := range approvedUsersList {
			if approvedGroupList[i].GroupID == approvedUser.GroupID {
				approvedGroupList[i].ApprovedUsersList = append(approvedGroupList[i].ApprovedUsersList, approvedUser)
			}
		}

		for _, unapprovedUser := range unapprovedUsersList {
			if approvedGroupList[i].GroupID == unapprovedUser.GroupID {
				approvedGroupList[i].UnapprovedUsersList = append(approvedGroupList[i].UnapprovedUsersList, unapprovedUser)
			}
		}
	}

	for i := 0; i < len(unapprovedGroupList); i++ {
		for _, approvedUser := range approvedUsersList {
			if unapprovedGroupList[i].GroupID == approvedUser.GroupID {
				unapprovedGroupList[i].ApprovedUsersList = append(unapprovedGroupList[i].ApprovedUsersList, approvedUser)
			}
		}

		for _, unapprovedUser := range unapprovedUsersList {
			if unapprovedGroupList[i].GroupID == unapprovedUser.GroupID {
				unapprovedGroupList[i].UnapprovedUsersList = append(unapprovedGroupList[i].UnapprovedUsersList, unapprovedUser)
			}
		}
	}

	groupList := model.NewGroupList(approvedGroupList, unapprovedGroupList)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&groupList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PostGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var group model.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateGroupName(group.GroupName); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	result, err := h.GroupRepo.PostGroupAndApprovedUser(&group, userID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupLastInsertId, err := result.LastInsertId()
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := postInitGroupStandardBudgets(int(groupLastInsertId)); err != nil {
		if err := h.GroupRepo.DeleteGroupAndApprovedUser(int(groupLastInsertId), userID); err != nil {
			errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbGroup, err := h.GroupRepo.GetGroup(int(groupLastInsertId))
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&dbGroup); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PutGroup(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var group model.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateGroupName(group.GroupName); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"group ID を正しく指定してください。"}))
		return
	}

	if err := h.GroupRepo.PutGroup(&group, groupID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	dbGroup, err := h.GroupRepo.GetGroup(groupID)
	if err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"グループ名を取得できませんでした"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&dbGroup); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PostGroupUnapprovedUser(w http.ResponseWriter, r *http.Request) {
	_, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	var groupUnapprovedUser model.UnapprovedUser
	if err := json.NewDecoder(r.Body).Decode(&groupUnapprovedUser); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := validateUserID(groupUnapprovedUser.UserID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, err))
		return
	}

	if _, err := h.UserRepo.FindSignUpUserByUserID(groupUnapprovedUser.UserID); err != nil {
		var notFoundError *apierrors.NotFoundError
		if xerrors.As(err, &notFoundError) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &NotFoundErrorMsg{"該当するユーザーが見つかりませんでした。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"group ID を正しく指定してください。"}))
		return
	}

	if err := checkForUniqueGroupUser(h, groupID, groupUnapprovedUser.UserID); err != nil {
		groupUserConflictErrorMsg, ok := err.(*ConflictErrorMsg)
		if !ok {
			errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusConflict, groupUserConflictErrorMsg))
		return
	}

	result, err := h.GroupRepo.PostUnapprovedUser(&groupUnapprovedUser, groupID)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	unapprovedUser, err := h.GroupRepo.GetUnapprovedUser(int(lastInsertId))
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&unapprovedUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) DeleteGroupApprovedUser(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"group ID を正しく指定してください。"}))
		return
	}

	if err := h.GroupRepo.FindApprovedUser(groupID, userID); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"こちらのグループには参加していません。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := h.GroupRepo.DeleteGroupApprovedUser(groupID, userID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&DeleteContentMsg{"グループを退会しました。"}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) PostGroupApprovedUser(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"group ID を正しく指定してください。"}))
		return
	}

	if err := h.GroupRepo.FindUnapprovedUser(groupID, userID); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"グループに招待されていないため、参加できませんでした。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupUserIDList, err := h.GroupRepo.GetGroupUsersList(groupID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	colorCode := assignColorCodeToUser(groupUserIDList)

	result, err := h.GroupRepo.PostGroupApprovedUserAndDeleteGroupUnapprovedUser(groupID, userID, colorCode)
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	approvedUser, err := h.GroupRepo.GetApprovedUser(int(lastInsertId))
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&approvedUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) DeleteGroupUnapprovedUser(w http.ResponseWriter, r *http.Request) {
	userID, err := verifySessionID(h, w, r)
	if err != nil {
		if err == http.ErrNoCookie || err == redis.ErrNil {
			errorResponseByJSON(w, NewHTTPError(http.StatusUnauthorized, &AuthenticationErrorMsg{"このページを表示するにはログインが必要です。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"group ID を正しく指定してください。"}))
		return
	}

	if err := h.GroupRepo.FindUnapprovedUser(groupID, userID); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			errorResponseByJSON(w, NewHTTPError(http.StatusBadRequest, &BadRequestErrorMsg{"こちらのグループには招待されていません。"}))
			return
		}

		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	if err := h.GroupRepo.DeleteGroupUnapprovedUser(groupID, userID); err != nil {
		errorResponseByJSON(w, NewHTTPError(http.StatusInternalServerError, nil))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&DeleteContentMsg{"グループ招待を拒否しました。"}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) VerifyGroupAffiliation(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := mux.Vars(r)["user_id"]

	if err := h.GroupRepo.FindApprovedUser(groupID, userID); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DBHandler) VerifyGroupAffiliationOfUsersList(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var groupUsersList model.GroupTasksUsersListReceiver
	if err := json.NewDecoder(r.Body).Decode(&groupUsersList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbGroupUsersList, err := h.GroupRepo.FindApprovedUsersList(groupID, groupUsersList.GroupUsersList)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(groupUsersList.GroupUsersList) != len(dbGroupUsersList.GroupUsersList) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DBHandler) GetGroupUserIDList(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.Atoi(mux.Vars(r)["group_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupUserIDList, err := h.GroupRepo.GetGroupUsersList(groupID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(groupUserIDList) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&groupUserIDList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
