package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	// "flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	// "github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/pkg/discovery"
	"movieexample.com/pkg/tracing"

	"movieexample.com/pkg/discovery/consul"
	// "movieexample.com/pkg/discovery/discmemory"
	"movieexample.com/rating/internal/controller/rating"
	"movieexample.com/rating/internal/repository/memory"

	// "movieexample.com/rating/internal/repository/mysql"

	// httphandler "movieexample.com/rating/internal/handler/http"
	grpchandler "movieexample.com/rating/internal/handler/grpc"
)

const serviceName = "rating"

func main() {
	// var port int

	// flag.IntVar(&port, "port", 8082, "API handler port")
	// flag.Parse()
	// log.Printf("Starting the rating service on port %d", port)

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
	log.Printf("Starting the rating service at %d.\n", port)

	// err := godotenv.Load("./.env")
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		panic(err)
	}

	// registry := discmemory.NewRegistry()

	ctx, cancel := context.WithCancel(context.Background())

	tp, err := tracing.NewJaegerProvider(cfg.Jaeger.URL, serviceName)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

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

	// repo, err := mysql.New()
	// if err != nil {
	// 	panic(err)
	// }

	repo := memory.New()

	// ingester, err := kafka.NewIngester(os.Getenv("RATING_SERVER_URL"), "kafka-grpc", "ratings")
	// if err != nil {
	// 	log.Fatalf("Failed to create kafak ingester: %v", err)
	// }

	// ctrl := rating.New(repo, ingester)
	ctrl := rating.New(repo, nil)
	h := grpchandler.New(ctrl)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()))
	reflection.Register(srv)
	gen.RegisterRatingServiceServer(srv, h)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		s := <-sigChan

		cancel()

		log.Printf("Received signal %v, attempting graceful shutdown.", s)
		srv.GracefulStop()
		log.Println("Gracefully stopped gRPC server.")
	}()

	if err := srv.Serve(lis); err != nil {
		panic(err)
	}

	wg.Wait()
}
