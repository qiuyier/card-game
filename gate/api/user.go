package api

import (
	"common/rpc"
	"github.com/gin-gonic/gin"
	"user/pb"
)

type UserHandler struct {
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (u *UserHandler) Register(ctx *gin.Context) {
	_, err := rpc.UserClient.Register(ctx, &pb.RegisterParams{})
	if err != nil {

	}

}
