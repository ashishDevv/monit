package middle

// func Metrics(recorder MetricsRecorder) func(http.Handler) http.Handler {
//     return func(next http.Handler) http.Handler {
//         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//             start := time.Now()

//             next.ServeHTTP(w, r)

//             recorder.Observe(
//                 r.Method,
//                 r.URL.Path,
//                 time.Since(start),
//             )
//         })
//     }
// }