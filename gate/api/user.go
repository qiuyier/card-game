package api

import (
	"common/logs"
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
	response, err := rpc.UserClient.Register(ctx, &pb.RegisterParams{})
	if err != nil {

	}
	uid := response.Uid
	logs.Info("uid:%s", uid)

}
