package testutil

import (
	"movieexample.com/gen"
	"movieexample.com/movie/internal/controller/movie"
	"movieexample.com/pkg/discovery"

	metadatagateway "movieexample.com/movie/internal/gateway/metadata/grpc"
	ratinggateway "movieexample.com/movie/internal/gateway/rating/grpc"
	grpchandler "movieexample.com/movie/internal/handler/grpc"
)

func NewTestMovieGRPCServer(registry discovery.Registry) gen.MovieServiceServer {
	metadataGateway := metadatagateway.New(registry)
	ratingGateway := ratinggateway.New(registry)
	ctrl := movie.New(ratingGateway, metadataGateway)

	return grpchandler.New(ctrl)
}
