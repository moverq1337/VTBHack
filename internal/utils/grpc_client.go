package utils

import (
	"context"
	"github.com/moverq1337/VTBHack/internal/pb"
	//"log"
	//
	//"github.com/moverq1337/VTBHack/internal/db"
	"google.golang.org/grpc"
)

func CallNLPParse(text string) (string, error) {
	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := pb.NewNLPServiceClient(conn)
	resp, err := client.ParseResume(context.Background(), &pb.ParseRequest{Text: text})
	if err != nil {
		return "", err
	}

	return resp.ParsedData, nil
}
