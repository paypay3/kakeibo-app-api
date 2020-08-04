package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/paypay3/kakeibo-app-api/user-rest-service/injector"
	"github.com/rs/cors"
)

func Run() error {
	h := injector.InjectDBHandler()

	router := mux.NewRouter()
	router.Handle("/signup", http.HandlerFunc(h.SignUp)).Methods("POST")
	router.Handle("/login", http.HandlerFunc(h.Login)).Methods("POST")
	router.Handle("/logout", http.HandlerFunc(h.Logout)).Methods("DELETE")
	router.Handle("/groups", http.HandlerFunc(h.GetGroupList)).Methods("GET")
	router.Handle("/groups", http.HandlerFunc(h.PostGroup)).Methods("POST")
	router.Handle("/groups/{group_id:[0-9]+}", http.HandlerFunc(h.PutGroup)).Methods("PUT")
	router.Handle("/groups/{group_id:[0-9]+}/users", http.HandlerFunc(h.PostGroupUnapprovedUser)).Methods("POST")
	router.Handle("/groups/{group_id:[0-9]+}/users/approved", http.HandlerFunc(h.PostGroupApprovedUser)).Methods("POST")
	router.Handle("/groups/{group_id:[0-9]+}/users/unapproved", http.HandlerFunc(h.DeleteGroupUnapprovedUser)).Methods("DELETE")

	corsWrapper := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Accept-Language"},
		AllowCredentials: true,
	})

	if err := http.ListenAndServe(":8080", corsWrapper.Handler(router)); err != nil {
		return err
	}

	return nil
}
