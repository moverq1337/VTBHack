package utils

import (
	"context"
	"github.com/moverq1337/VTBHack/internal/pb"
	"google.golang.org/grpc"
	"os"
)

// CallNLPParse вызывает NLP-сервис для парсинга текста
func CallNLPParse(text string) (string, error) {
	grpcHost := os.Getenv("GRPC_HOST")
	grpcPort := os.Getenv("GRPC_PORT")

	conn, err := grpc.Dial(grpcHost+":"+grpcPort, grpc.WithInsecure())
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
