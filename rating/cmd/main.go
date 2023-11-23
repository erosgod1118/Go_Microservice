package main

import (
	"context"
	// "flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	// "github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/pkg/discovery"

	// "movieexample.com/pkg/discovery/consul"
	"movieexample.com/pkg/discovery/discmemory"
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

	// registry, err := consul.NewRegistry(os.Getenv("HASHCORP_CONSUL_URL"))
	// if err != nil {
	// 	panic(err)
	// }

	registry := discmemory.NewRegistry()

	ctx := context.Background()
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

	srv := grpc.NewServer()
	reflection.Register(srv)
	gen.RegisterRatingServiceServer(srv, h)
	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}
