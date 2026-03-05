package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/manimovassagh/rampart/internal/cli"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "login":
		err = cmdLogin(args)
	case "logout":
		err = cmdLogout()
	case "status":
		err = cmdStatus()
	case "whoami":
		err = cmdWhoami()
	case "users":
		err = cmdUsers(args)
	case "token":
		err = cmdToken()
	case "version":
		fmt.Printf("rampart-cli %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd) //nolint:gosec // cmd is from os.Args, not external input
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`rampart-cli — Rampart IAM management tool

Usage:
  rampart-cli <command> [options]

Commands:
  login     --issuer URL --email EMAIL --password PASS   Authenticate
  logout                                                  Clear stored credentials
  status                                                  Check server health
  whoami                                                  Show current user
  users     list | create | get <id>                      Manage users
  token                                                   Show current access token
  version                                                 Print version

Examples:
  rampart-cli login --issuer http://localhost:8080 --email admin@example.com --password secret
  rampart-cli status
  rampart-cli users list
  rampart-cli users create --email jane@example.com --username jane --password 'P@ss1234'
`)
}

// ── login ─────────────────────────────────────────────

func cmdLogin(args []string) error {
	var issuer, email, password string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--issuer":
			i++
			if i < len(args) {
				issuer = args[i]
			}
		case "--email", "--identifier":
			i++
			if i < len(args) {
				email = args[i]
			}
		case "--password":
			i++
			if i < len(args) {
				password = args[i]
			}
		}
	}

	if issuer == "" || email == "" || password == "" {
		return fmt.Errorf("missing required flags: --issuer, --email, --password")
	}

	issuer = strings.TrimRight(issuer, "/")

	cfg := &cli.Config{Issuer: issuer}
	client := cli.NewClient(cfg)

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		User         struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"user"`
	}

	err := client.Post("/login", map[string]string{
		"identifier": email,
		"password":   password,
	}, &loginResp)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	cfg.AccessToken = loginResp.AccessToken
	cfg.RefreshToken = loginResp.RefreshToken

	if err := cli.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Printf("Logged in as %s (%s)\n", loginResp.User.Username, loginResp.User.Email)
	fmt.Printf("Token expires in %ds\n", loginResp.ExpiresIn)
	return nil
}

// ── logout ────────────────────────────────────────────

func cmdLogout() error {
	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.AccessToken != "" {
		client := cli.NewClient(cfg)
		_ = client.Post("/logout", map[string]string{
			"refresh_token": cfg.RefreshToken,
		}, nil)
	}

	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	if err := cli.SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println("Logged out.")
	return nil
}

// ── status ────────────────────────────────────────────

func cmdStatus() error {
	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.Issuer == "" {
		return fmt.Errorf("not configured, run: rampart-cli login --issuer URL")
	}

	client := cli.NewClient(cfg)

	var health struct {
		Status string `json:"status"`
	}
	if err := client.Get("/healthz", &health); err != nil {
		fmt.Printf("Server: %s\n", cfg.Issuer)
		fmt.Printf("Status: unreachable (%v)\n", err)
		return nil
	}

	var ready struct {
		Status string `json:"status"`
	}
	_ = client.Get("/readyz", &ready)

	fmt.Printf("Server:    %s\n", cfg.Issuer)
	fmt.Printf("Health:    %s\n", health.Status)
	fmt.Printf("Ready:     %s\n", ready.Status)

	if cfg.AccessToken != "" {
		fmt.Printf("Auth:      logged in\n")
	} else {
		fmt.Printf("Auth:      not logged in\n")
	}
	return nil
}

// ── whoami ────────────────────────────────────────────

func cmdWhoami() error {
	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not logged in, run: rampart-cli login")
	}

	client := cli.NewClient(cfg)

	var me map[string]any
	if err := client.Get("/me", &me); err != nil {
		return err
	}

	fmt.Printf("User:      %s\n", me["preferred_username"])
	fmt.Printf("Email:     %s\n", me["email"])
	fmt.Printf("ID:        %s\n", me["id"])
	if orgID, ok := me["org_id"]; ok {
		fmt.Printf("Org:       %s\n", orgID)
	}
	if roles, ok := me["roles"]; ok {
		fmt.Printf("Roles:     %v\n", roles)
	}
	return nil
}

// ── users ─────────────────────────────────────────────

func cmdUsers(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand: users <list|create|get>")
	}

	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not logged in, run: rampart-cli login")
	}

	client := cli.NewClient(cfg)

	switch args[0] {
	case "list":
		return cmdUsersList(client)
	case "create":
		return cmdUsersCreate(client, args[1:])
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("missing user ID: users get <user-id>")
		}
		return cmdUsersGet(client, args[1])
	default:
		return fmt.Errorf("unknown users subcommand: %s", args[0])
	}
}

func cmdUsersList(client *cli.Client) error {
	var resp struct {
		Users []map[string]any `json:"users"`
	}
	if err := client.Get("/api/v1/admin/users", &resp); err != nil {
		return err
	}
	users := resp.Users

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tENABLED")
	for _, u := range users {
		id, _ := u["id"].(string)
		username, _ := u["username"].(string)
		email, _ := u["email"].(string)
		enabled, _ := u["enabled"].(bool)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", id, username, email, enabled)
	}
	return w.Flush()
}

func cmdUsersCreate(client *cli.Client, args []string) error {
	var email, username, password, givenName, familyName string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email":
			i++
			if i < len(args) {
				email = args[i]
			}
		case "--username":
			i++
			if i < len(args) {
				username = args[i]
			}
		case "--password":
			i++
			if i < len(args) {
				password = args[i]
			}
		case "--given-name":
			i++
			if i < len(args) {
				givenName = args[i]
			}
		case "--family-name":
			i++
			if i < len(args) {
				familyName = args[i]
			}
		}
	}

	if email == "" || username == "" || password == "" {
		return fmt.Errorf("missing required flags: --email, --username, --password")
	}

	body := map[string]string{
		"email":       email,
		"username":    username,
		"password":    password,
		"given_name":  givenName,
		"family_name": familyName,
	}

	var user map[string]any
	if err := client.Post("/api/v1/admin/users", body, &user); err != nil {
		return err
	}

	fmt.Printf("Created user %s (%s)\n", user["username"], user["id"])
	return nil
}

func cmdUsersGet(client *cli.Client, id string) error {
	var user map[string]any
	if err := client.Get("/api/v1/admin/users/"+id, &user); err != nil {
		return err
	}

	data, _ := json.MarshalIndent(user, "", "  ")
	fmt.Println(string(data))
	return nil
}

// ── token ─────────────────────────────────────────────

func cmdToken() error {
	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not logged in, run: rampart-cli login")
	}

	fmt.Println(cfg.AccessToken)
	return nil
}
