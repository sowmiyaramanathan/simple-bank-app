package gapi

import (
	"fmt"

	db "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/db/sqlc"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/db/util"
	pb "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/pb"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/token"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/worker"
)

type Server struct {
	pb.UnimplementedSimpleBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}
	server := &Server{config: config, store: store, tokenMaker: tokenMaker, taskDistributor: taskDistributor}

	return server, nil
}
