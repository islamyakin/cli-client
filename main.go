package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-tty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "cli-client/auth"
)

const (
	address   = "localhost:50051"
	tokenFile = "go-client-token"
)

var tokenFilePath = filepath.Join(os.TempDir(), tokenFile)

func readPassword() (string, error) {
	tty, err := tty.Open()
	if err != nil {
		return "", err
	}
	defer tty.Close()

	var password strings.Builder
	for {
		r, err := tty.ReadRune()
		if err != nil {
			return "", err
		}
		if r == '\n' || r == '\r' {
			break
		}
		password.WriteRune(r)
	}
	return password.String(), nil
}

func saveTokenToFile(token string) error {
	file, err := os.Create(tokenFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	result := []byte(token + "\n")

	_, err = file.WriteString(string(result))
	return err
}

func readTokenFromFile() (string, error) {
	file, err := os.Open(tokenFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(token), nil
}

func login(client pb.AuthServiceClient, username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Login(ctx, &pb.LoginRequest{Username: username, Password: password})
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", resp.Message)
	return saveTokenToFile(resp.GetToken())
}

func sendMessage(client pb.AuthServiceClient, token, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.SendMessage(ctx, &pb.MessageRequest{Token: token, Message: message})
	if err != nil {
		st, _ := status.FromError(err)
		return fmt.Errorf("error: %v", st.Message())
	}
	fmt.Println(resp.GetResponse())
	return nil
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)

	var rootCmd = &cobra.Command{Use: "go-client"}

	var username, password, message string

	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login with username and password",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print("Enter password: ")
			pass, err := readPassword()
			if err != nil {
				fmt.Println("Error reading password:", err)
				return
			}
			password = strings.TrimSpace(pass)
			err = login(client, username, password)
			if err != nil {
				fmt.Println("Login failed:", err)
			}
		},
	}

	loginCmd.Flags().StringVarP(&username, "username", "u", "", "Username")
	loginCmd.MarkFlagRequired("username")

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Send message to server",
		Run: func(cmd *cobra.Command, args []string) {
			token, err := readTokenFromFile()
			if err != nil {
				fmt.Println("You must login first:", err)
				return
			}
			err = sendMessage(client, token, message)
			if err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	runCmd.Flags().StringVarP(&message, "name", "n", "", "Message to send")
	runCmd.MarkFlagRequired("name")

	rootCmd.AddCommand(loginCmd, runCmd)
	rootCmd.Execute()
}
