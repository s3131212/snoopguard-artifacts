package main

import (
	"chatbot-poc-go/pkg/client"
	"flag"
)

func main() {
	var address, userID string
	flag.StringVar(&address, "addr", "localhost:50051", "the address to connect to")
	flag.StringVar(&userID, "u", "alice", "user ID")
	flag.Parse()

	user := client.NewClient(userID)
	closeClient := user.SetupChatServiceClient(address)
	defer closeClient()
	user.RegisterUserToServer()

	user.ListenToStreams()
}
