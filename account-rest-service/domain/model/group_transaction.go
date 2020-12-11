package model

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"
)

type GroupTransactionsList struct {
	GroupTransactionsList []GroupTransactionSender `json:"transactions_list"`
}

type GroupTransactionSender struct {
	ID                 int        `json:"id"                   db:"id"`
	TransactionType    string     `json:"transaction_type"     db:"transaction_type"`
	PostedDate         time.Time  `json:"posted_date"          db:"posted_date"`
	UpdatedDate        time.Time  `json:"updated_date"         db:"updated_date"`
	TransactionDate    SenderDate `json:"transaction_date"     db:"transaction_date"`
	Shop               NullString `json:"shop"                 db:"shop"`
	Memo               NullString `json:"memo"                 db:"memo"`
	Amount             int        `json:"amount"               db:"amount"`
	PostedUserID       string     `json:"posted_user_id"       db:"posted_user_id"`
	UpdatedUserID      NullString `json:"updated_user_id"      db:"updated_user_id"`
	PaymentUserID      string     `json:"payment_user_id"      db:"payment_user_id"`
	BigCategoryID      int        `json:"big_category_id"      db:"big_category_id"`
	BigCategoryName    string     `json:"big_category_name"    db:"big_category_name"`
	MediumCategoryID   NullInt64  `json:"medium_category_id"   db:"medium_category_id"`
	MediumCategoryName NullString `json:"medium_category_name" db:"medium_category_name"`
	CustomCategoryID   NullInt64  `json:"custom_category_id"   db:"custom_category_id"`
	CustomCategoryName NullString `json:"custom_category_name" db:"custom_category_name"`
}

type GroupTransactionReceiver struct {
	TransactionType  string       `json:"transaction_type"   db:"transaction_type"   validate:"required,oneof=expense income"`
	TransactionDate  ReceiverDate `json:"transaction_date"   db:"transaction_date"   validate:"required,date"`
	Shop             NullString   `json:"shop"               db:"shop"               validate:"omitempty,max=20,blank"`
	Memo             NullString   `json:"memo"               db:"memo"               validate:"omitempty,max=50,blank"`
	Amount           int          `json:"amount"             db:"amount"             validate:"required,min=1"`
	PaymentUserID    string       `json:"payment_user_id"    db:"payment_user_id"`
	BigCategoryID    int          `json:"big_category_id"    db:"big_category_id"    validate:"required,min=1,max=17,either_id"`
	MediumCategoryID NullInt64    `json:"medium_category_id" db:"medium_category_id" validate:"omitempty,min=1,max=99"`
	CustomCategoryID NullInt64    `json:"custom_category_id" db:"custom_category_id" validate:"omitempty,min=1"`
}

type PayerList struct {
	PayerList []UserPaymentAmount
}

type RecipientList struct {
	RecipientList []UserPaymentAmount
}

type UserPaymentAmount struct {
	UserID              string `db:"user_id"`
	TotalPaymentAmount  int    `db:"total_payment_amount"`
	PaymentAmountToUser int
}

type GroupAccountsList struct {
	GroupID                   int            `json:"group_id"`
	Month                     time.Time      `json:"month"`
	GroupTotalPaymentAmount   int            `json:"group_total_payment_amount"`
	GroupAveragePaymentAmount int            `json:"group_average_payment_amount"`
	GroupRemainingAmount      int            `json:"group_remaining_amount"`
	GroupAccountsList         []GroupAccount `json:"group_accounts_list"`
}

type GroupAccount struct {
	ID                  int        `json:"id"                   db:"id"`
	GroupID             int        `json:"group_id"             db:"group_id"`
	Month               time.Time  `json:"month"                db:"years_months"`
	Payer               NullString `json:"payer_user_id"        db:"payer_user_id"`
	Recipient           NullString `json:"recipient_user_id"    db:"recipient_user_id"`
	PaymentAmount       NullInt    `json:"payment_amount"       db:"payment_amount"`
	PaymentConfirmation BitBool    `json:"payment_confirmation" db:"payment_confirmation"`
	ReceiptConfirmation BitBool    `json:"receipt_confirmation" db:"receipt_confirmation"`
}

type BitBool bool

type NullInt struct {
	Int   int
	Valid bool
}

func NewGroupTransactionsList(groupTransactionsList []GroupTransactionSender) GroupTransactionsList {
	return GroupTransactionsList{GroupTransactionsList: groupTransactionsList}
}

func (t GroupTransactionReceiver) ShowTransactionReceiver() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return string(b), err
	}

	return string(b), nil
}

func NewPayerList(userPaymentAmountList []UserPaymentAmount) PayerList {
	var payerList PayerList
	for _, userPaymentAmount := range userPaymentAmountList {
		if userPaymentAmount.PaymentAmountToUser < 0 {
			payerList.PayerList = append(payerList.PayerList, userPaymentAmount)
		}
	}

	sort.Slice(payerList.PayerList, func(i, j int) bool {
		return payerList.PayerList[i].PaymentAmountToUser < payerList.PayerList[j].PaymentAmountToUser
	})

	return payerList
}

func NewRecipientList(userPaymentAmountList []UserPaymentAmount) RecipientList {
	var recipientList RecipientList
	for _, userPaymentAmount := range userPaymentAmountList {
		if userPaymentAmount.PaymentAmountToUser > 0 {
			recipientList.RecipientList = append(recipientList.RecipientList, userPaymentAmount)
		}
	}

	sort.Slice(recipientList.RecipientList, func(i, j int) bool {
		return recipientList.RecipientList[i].PaymentAmountToUser > recipientList.RecipientList[j].PaymentAmountToUser
	})

	return recipientList
}

func NewGroupAccountsList(userPaymentAmountList []UserPaymentAmount, groupID int, month time.Time) GroupAccountsList {
	var totalPaymentAmount int
	for _, userPaymentAmount := range userPaymentAmountList {
		totalPaymentAmount += userPaymentAmount.TotalPaymentAmount
	}

	averagePaymentAmount := int(math.Round((float64(totalPaymentAmount)) / float64(len(userPaymentAmountList))))
	remainingAmount := totalPaymentAmount - averagePaymentAmount*len(userPaymentAmountList)

	return GroupAccountsList{
		GroupID:                   groupID,
		Month:                     month,
		GroupTotalPaymentAmount:   totalPaymentAmount,
		GroupAveragePaymentAmount: averagePaymentAmount,
		GroupRemainingAmount:      remainingAmount,
	}
}

func (b BitBool) Value() (driver.Value, error) {
	if b {
		return []byte{1}, nil
	}

	return []byte{0}, nil
}

func (b *BitBool) Scan(src interface{}) error {
	bitBool, ok := src.([]byte)
	if !ok {
		return errors.New("bad []byte type assertion")
	}

	*b = bitBool[0] == 1

	return nil
}

func (ni *NullInt) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(ni.Int)
}

func (ni *NullInt) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	if err := json.Unmarshal(b, &ni.Int); err != nil {
		return err
	}

	ni.Valid = true

	return nil
}

func (ni *NullInt) Scan(value interface{}) error {
	if value == nil {
		ni.Int, ni.Valid = 0, false
		return nil
	}

	intValue, ok := value.(int64)
	if !ok {
		return errors.New("type assertion error")
	}

	ni.Int, ni.Valid = int(intValue), true

	return nil
}

func (ni NullInt) Value() (driver.Value, error) {
	if !ni.Valid {
		return nil, nil
	}

	return int64(ni.Int), nil
}
