package testutil

import (
	"movieexample.com/gen"
	"movieexample.com/rating/internal/controller/rating"
	"movieexample.com/rating/internal/repository/memory"

	grpchandler "movieexample.com/rating/internal/handler/grpc"
)

func NewTestRatingGRPCServer() gen.RatingServiceServer {
	r := memory.New()
	ctrl := rating.New(r, nil)

	return grpchandler.New(ctrl)
}
