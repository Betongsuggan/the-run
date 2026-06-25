package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/BirgerRydback/the-run/backend/internal/api"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/store"
	"github.com/BirgerRydback/the-run/backend/internal/turnstile"
)

func main() {
	ctx := context.Background()

	dynStore, err := store.NewDynamoStoreFromEnv(ctx)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	authCfg, err := auth.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("init auth config: %v", err)
	}

	turnstileCfg, err := turnstile.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("init turnstile config: %v", err)
	}

	mux := http.NewServeMux()
	api.Register(mux, dynStore, authCfg, turnstileCfg)

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") == "" {
		addr := ":8080"
		log.Printf("the-run api listening on %s (local dev)", addr)
		log.Fatal(http.ListenAndServe(addr, withDevCORS(accessLog(mux))))
		return
	}

	adapter := httpadapter.NewV2(shortCircuitOptions(accessLog(mux)))
	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return adapter.ProxyWithContext(ctx, req)
	})
}

// accessLog writes one line per request to the default logger:
//
//	METHOD path -> status (duration)
//
// Goes to stdout locally and to CloudWatch Logs in Lambda. The status is
// captured via a thin ResponseWriter wrapper so we record what the mux wrote,
// not just whatever happens to have been set by middleware before us.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}

// shortCircuitOptions returns 204 for any OPTIONS request before it reaches
// the mux. Huma registers routes via Go 1.22 method-aware patterns ("POST
// /auth/login"), so an OPTIONS preflight to those paths would otherwise 405
// — API Gateway's CORS config adds Access-Control-* headers on top, but the
// browser still rejects the preflight because the status isn't 2xx.
func shortCircuitOptions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withDevCORS allows the SvelteKit dev server (:5173) to call this API with
// cookies. Production CORS is configured on API Gateway; this is dev-only.
func withDevCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allow any localhost origin in dev — browsers reject "*" with credentials.
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
