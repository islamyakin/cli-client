package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mattn/go-tty"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var username, password, message string
var tokenFilePath = filepath.Join(os.TempDir(), "go-client-token")

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
func main() {
	var rootCmd = &cobra.Command{Use: "go-client"}

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
			err = login(username, password)
			if err != nil {
				fmt.Println("\nLogin failed:", err)
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
			err = sendMessageToServer(token, message)
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

func login(username, password string) error {
	url := "http://localhost:8080/login"
	loginReq := map[string]string{"username": username, "password": password}
	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	var loginResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return err
	}

	fmt.Printf("\n%s\n", loginResp["message"])

	token := loginResp["token"]

	return saveTokenToFile(token)
}

func sendMessageToServer(token, message string) error {
	url := "http://localhost:8080/log"
	msgReq := map[string]string{"token": token, "message": message}
	jsonData, err := json.Marshal(msgReq)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	return nil
}

func saveTokenToFile(token string) error {

	file, err := os.OpenFile(tokenFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	result := []byte(token + "\n")
	defer file.Close()

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
