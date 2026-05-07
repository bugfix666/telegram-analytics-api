package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type interactiveAuthenticator struct{}

func (a interactiveAuthenticator) Phone(ctx context.Context) (string, error) {
	fmt.Print("Phone (international): ")
	return readLine(), nil
}

func (a interactiveAuthenticator) Password(ctx context.Context) (string, error) {
	fmt.Print("2FA password (if any): ")
	return readLine(), nil
}

func (a interactiveAuthenticator) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Code: ")
	return readLine(), nil
}

func (a interactiveAuthenticator) AcceptTermsOfService(ctx context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (a interactiveAuthenticator) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, nil
}

func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
