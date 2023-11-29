package main

import (
	"context"
	"os"

	// "flag"
	"fmt"
	"log"
	"net"
	"time"

	// "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	// "go.opentelemetry.io/otel"
	// "go.opentelemetry.io/otel/propagation"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/movie/internal/controller/movie"
	metadatagateway "movieexample.com/movie/internal/gateway/metadata/grpc"
	ratinggateway "movieexample.com/movie/internal/gateway/rating/grpc"
	grpchandler "movieexample.com/movie/internal/handler/grpc"
	"movieexample.com/pkg/discovery"

	// "movieexample.com/pkg/tracing"

	"movieexample.com/pkg/discovery/consul"
	// "movieexample.com/pkg/discovery/discmemory"
)

const serviceName = "movie"

type limiter struct {
	l *rate.Limiter
}

func main() {
	// var port int

	// flag.IntVar(&port, "port", 8083, "API handler port")
	// flag.Parse()
	// log.Printf("Starting the movie service on port %d", port)

	f, err := os.Open("../configs/base.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var cfg serviceConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		panic(err)
	}

	port := cfg.APIConfig.Port
	log.Printf("Starting the movie service at %d.\n", port)

	ctx := context.Background()

	// tp, err := tracing.NewJaegerProvider(cfg.Jaeger.URL, serviceName)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer func() {
	// 	if err := tp.Shutdown(ctx); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	// otel.SetTracerProvider(tp)
	// otel.SetTextMapPropagator(propagation.TraceContext{})

	// registry := discmemory.NewRegistry()

	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		panic(err)
	}

	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("Failed to report healthy state: " + err.Error())
			}

			time.Sleep(1 * time.Second)
		}
	}()

	defer registry.Deregister(ctx, instanceID, serviceName)

	metadataGateway := metadatagateway.New(registry)
	ratingGateway := ratinggateway.New(registry)
	ctrl := movie.New(ratingGateway, metadataGateway)
	h := grpchandler.New(ctrl)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	const limit = 100
	const burst = 100

	l := newLimiter(limit, burst)
	srv := grpc.NewServer(grpc.UnaryInterceptor(ratelimit.UnaryServerInterceptor(l)))
	// srv := grpc.NewServer(grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()))

	reflection.Register(srv)
	gen.RegisterMovieServiceServer(srv, h)
	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}

func newLimiter(limit int, burst int) *limiter {
	return &limiter{rate.NewLimiter(rate.Limit(limit), burst)}
}

func (lim *limiter) Limit() bool {
	return lim.l.Allow()
}
