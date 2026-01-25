package middle

// type AuthorizationMiddleware struct {
//     Policy PolicyService
// }

// func (a *AuthorizationMiddleware) Handle(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         user := r.Context().Value(userKey).(*User)

//         if !a.Policy.CanAccess(user, r.URL.Path) {
//             http.Error(w, "forbidden", http.StatusForbidden)
//             return
//         }

//         next.ServeHTTP(w, r)
//     })
// }