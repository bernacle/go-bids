package api

import (
	"fmt"
	"net/http"

	"gobid/internal/jsonutils"

	"github.com/gorilla/csrf"
)

func (api *Api) HandleGetCSRFtoken(w http.ResponseWriter, r *http.Request) {
	token := csrf.Token(r)
	jsonutils.EncodeJson(w, r, http.StatusOK, map[string]any{
		"csrf_token": token,
	})
}

func (api *Api) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add debug logging
		fmt.Printf("Checking auth, session exists: %v\n",
			api.Sessions.Exists(r.Context(), "AuthenticatedUserId"))

		if !api.Sessions.Exists(r.Context(), "AuthenticatedUserId") {
			jsonutils.EncodeJson(w, r, http.StatusUnauthorized, map[string]any{
				"message": "must be logged in",
			})
			return
		}

		// Add debug logging
		userId := api.Sessions.Get(r.Context(), "AuthenticatedUserId")
		fmt.Printf("Auth successful for user: %v\n", userId)

		next.ServeHTTP(w, r)
	})
}
