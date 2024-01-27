package service

import (
	"common/logs"
	"context"
	"core/repo"
	"user/pb"
)

type AccountService struct {
	pb.UnimplementedUserServiceServer
}

func NewAccountService(manager *repo.Manager) *AccountService {
	return &AccountService{}
}

func (a *AccountService) Register(context.Context, *pb.RegisterParams) (*pb.RegisterResponse, error) {
	logs.Info("register server called...")
	return &pb.RegisterResponse{Uid: "1"}, nil
}
