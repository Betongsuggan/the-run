package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/BirgerRydback/the-run/backend/internal/api"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/store"
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

	mux := http.NewServeMux()
	api.Register(mux, dynStore, authCfg)

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") == "" {
		addr := ":8080"
		log.Printf("the-run api listening on %s (local dev)", addr)
		log.Fatal(http.ListenAndServe(addr, withDevCORS(mux)))
		return
	}

	adapter := httpadapter.NewV2(mux)
	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return adapter.ProxyWithContext(ctx, req)
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
