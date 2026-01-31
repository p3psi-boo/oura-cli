package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const webhookBase = "https://api.ouraring.com/v2/webhook"

type WebhookSubscription struct {
	ID             string `json:"id"`
	CallbackURL    string `json:"callback_url"`
	EventType      string `json:"event_type"`
	DataType       string `json:"data_type"`
	ExpirationTime string `json:"expiration_time"`
}

type CreateWebhookSubscriptionRequest struct {
	CallbackURL       string `json:"callback_url"`
	VerificationToken string `json:"verification_token"`
	EventType         string `json:"event_type"`
	DataType          string `json:"data_type"`
}

type UpdateWebhookSubscriptionRequest struct {
	VerificationToken string  `json:"verification_token"`
	CallbackURL       *string `json:"callback_url,omitempty"`
	EventType         *string `json:"event_type,omitempty"`
	DataType          *string `json:"data_type,omitempty"`
}

var webhookOperations = []string{"create", "update", "delete"}

// Values from ExtApiV2DataType in refs/openapi-1.27.json.
var webhookDataTypes = []string{
	"tag",
	"enhanced_tag",
	"workout",
	"session",
	"sleep",
	"daily_sleep",
	"daily_readiness",
	"daily_activity",
	"daily_spo2",
	"sleep_time",
	"rest_mode_period",
	"ring_configuration",
	"daily_stress",
	"daily_cardiovascular_age",
	"daily_resilience",
	"vo2_max",
}

func handleWebhook(args []string, opts Options) {
	if len(args) < 1 {
		printWebhookUsage()
		os.Exit(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		if err := webhookList(opts); err != nil {
			exitErr(err)
		}
	case "get":
		if len(rest) != 1 {
			printWebhookUsage()
			os.Exit(1)
		}
		if err := webhookGet(rest[0], opts); err != nil {
			exitErr(err)
		}
	case "create":
		if err := webhookCreate(rest, opts); err != nil {
			exitErr(err)
		}
	case "update":
		if len(rest) < 1 {
			printWebhookUsage()
			os.Exit(1)
		}
		id := rest[0]
		if err := webhookUpdate(id, rest[1:], opts); err != nil {
			exitErr(err)
		}
	case "delete":
		if len(rest) != 1 {
			printWebhookUsage()
			os.Exit(1)
		}
		if err := webhookDelete(rest[0]); err != nil {
			exitErr(err)
		}
	case "renew":
		if len(rest) != 1 {
			printWebhookUsage()
			os.Exit(1)
		}
		if err := webhookRenew(rest[0], opts); err != nil {
			exitErr(err)
		}
	case "types":
		printWebhookTypes()
	default:
		printWebhookUsage()
		os.Exit(1)
	}
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func printWebhookUsage() {
	fmt.Print(`Webhook subscription management

Usage:
  oura webhook list [--json|-j]
  oura webhook get <id> [--json|-j]
  oura webhook create --callback-url <url> --verification-token <token> --event-type <create|update|delete> --data-type <type> [--json|-j]
  oura webhook update <id> --verification-token <token> [--callback-url <url>] [--event-type <create|update|delete>] [--data-type <type>] [--json|-j]
  oura webhook delete <id>
  oura webhook renew <id> [--json|-j]
  oura webhook types

Notes:
  - These endpoints use app credentials (x-client-id / x-client-secret), not the OAuth access token.
  - client_id/client_secret come from ~/.config/oura/config.json
`)
}

func printWebhookTypes() {
	fmt.Println("event_type:")
	for _, v := range webhookOperations {
		fmt.Printf("  %s\n", v)
	}
	fmt.Println("\ndata_type:")
	for _, v := range webhookDataTypes {
		fmt.Printf("  %s\n", v)
	}
}

func webhookList(opts Options) error {
	body, _, err := webhookDo("GET", "/subscription", nil)
	if err != nil {
		return err
	}

	if opts.JSON {
		// Preserve the original API payload.
		os.Stdout.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	var subs []WebhookSubscription
	if err := json.Unmarshal(body, &subs); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(subs) == 0 {
		fmt.Println("No webhook subscriptions")
		return nil
	}

	fmt.Printf("Webhook subscriptions (%d)\n", len(subs))
	fmt.Println(strings.Repeat("-", 72))
	for _, s := range subs {
		fmt.Printf("%s  %s/%s  expires=%s\n", s.ID, s.DataType, s.EventType, s.ExpirationTime)
		fmt.Printf("  %s\n", s.CallbackURL)
	}
	return nil
}

func webhookGet(id string, opts Options) error {
	body, _, err := webhookDo("GET", "/subscription/"+urlPathEscape(id), nil)
	if err != nil {
		return err
	}

	if opts.JSON {
		os.Stdout.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	var s WebhookSubscription
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Webhook subscription")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Type:      %s/%s\n", s.DataType, s.EventType)
	fmt.Printf("Expires:   %s\n", s.ExpirationTime)
	fmt.Printf("Callback:  %s\n", s.CallbackURL)
	return nil
}

func webhookCreate(args []string, opts Options) error {
	flags, pos, err := parseLongFlags(args)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return fmt.Errorf("unexpected args: %s", strings.Join(pos, " "))
	}

	callbackURL := firstFlag(flags, "callback-url", "callback_url")
	verificationToken := firstFlag(flags, "verification-token", "verification_token")
	eventType := firstFlag(flags, "event-type", "event_type")
	dataType := firstFlag(flags, "data-type", "data_type")

	if callbackURL == "" || verificationToken == "" || eventType == "" || dataType == "" {
		return fmt.Errorf("missing required flags; see: oura webhook create --help (or: oura webhook types)")
	}

	if err := validateEnum("event_type", eventType, webhookOperations); err != nil {
		return err
	}
	if err := validateEnum("data_type", dataType, webhookDataTypes); err != nil {
		return err
	}

	req := CreateWebhookSubscriptionRequest{
		CallbackURL:       callbackURL,
		VerificationToken: verificationToken,
		EventType:         eventType,
		DataType:          dataType,
	}

	body, _, err := webhookDo("POST", "/subscription", req)
	if err != nil {
		return err
	}

	if opts.JSON {
		os.Stdout.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	var s WebhookSubscription
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Created webhook subscription")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Type:      %s/%s\n", s.DataType, s.EventType)
	fmt.Printf("Expires:   %s\n", s.ExpirationTime)
	fmt.Printf("Callback:  %s\n", s.CallbackURL)
	return nil
}

func webhookUpdate(id string, args []string, opts Options) error {
	flags, pos, err := parseLongFlags(args)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return fmt.Errorf("unexpected args: %s", strings.Join(pos, " "))
	}

	verificationToken := firstFlag(flags, "verification-token", "verification_token")
	if verificationToken == "" {
		return fmt.Errorf("missing required flag: --verification-token")
	}

	var callbackURL *string
	if v := firstFlag(flags, "callback-url", "callback_url"); v != "" {
		callbackURL = &v
	}
	var eventType *string
	if v := firstFlag(flags, "event-type", "event_type"); v != "" {
		if err := validateEnum("event_type", v, webhookOperations); err != nil {
			return err
		}
		eventType = &v
	}
	var dataType *string
	if v := firstFlag(flags, "data-type", "data_type"); v != "" {
		if err := validateEnum("data_type", v, webhookDataTypes); err != nil {
			return err
		}
		dataType = &v
	}

	req := UpdateWebhookSubscriptionRequest{
		VerificationToken: verificationToken,
		CallbackURL:       callbackURL,
		EventType:         eventType,
		DataType:          dataType,
	}

	body, _, err := webhookDo("PUT", "/subscription/"+urlPathEscape(id), req)
	if err != nil {
		return err
	}

	if opts.JSON {
		os.Stdout.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	var s WebhookSubscription
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Updated webhook subscription")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Type:      %s/%s\n", s.DataType, s.EventType)
	fmt.Printf("Expires:   %s\n", s.ExpirationTime)
	fmt.Printf("Callback:  %s\n", s.CallbackURL)
	return nil
}

func webhookDelete(id string) error {
	_, status, err := webhookDo("DELETE", "/subscription/"+urlPathEscape(id), nil)
	if err != nil {
		return err
	}
	if status != 204 {
		return fmt.Errorf("unexpected status: %d", status)
	}
	fmt.Println("Deleted webhook subscription", id)
	return nil
}

func webhookRenew(id string, opts Options) error {
	body, _, err := webhookDo("PUT", "/subscription/renew/"+urlPathEscape(id), nil)
	if err != nil {
		return err
	}

	if opts.JSON {
		os.Stdout.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	var s WebhookSubscription
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Renewed webhook subscription")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Type:      %s/%s\n", s.DataType, s.EventType)
	fmt.Printf("Expires:   %s\n", s.ExpirationTime)
	fmt.Printf("Callback:  %s\n", s.CallbackURL)
	return nil
}

func webhookDo(method string, endpoint string, payload any) (respBody []byte, status int, err error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, webhookBase+endpoint, body)
	if err != nil {
		return nil, 0, err
	}

	// Webhook subscription endpoints use app credentials.
	req.Header.Set("x-client-id", config.ClientID)
	req.Header.Set("x-client-secret", config.ClientSecret)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	status = resp.StatusCode
	if status < 200 || status > 299 {
		return nil, status, fmt.Errorf("API error %d: %s", status, strings.TrimSpace(string(respBody)))
	}

	return respBody, status, nil
}

func parseLongFlags(args []string) (flags map[string]string, pos []string, err error) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}

		nameVal := strings.TrimPrefix(a, "--")
		if nameVal == "" {
			return nil, nil, fmt.Errorf("invalid flag: %q", a)
		}

		name, val, hasEq := strings.Cut(nameVal, "=")
		if !hasEq {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", a)
			}
			val = args[i+1]
			i++
		}
		flags[name] = val
	}
	return flags, pos, nil
}

func firstFlag(flags map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := flags[k]; ok {
			return v
		}
	}
	return ""
}

func validateEnum(name string, v string, allowed []string) error {
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("invalid %s: %q (try: oura webhook types)", name, v)
}

func urlPathEscape(s string) string {
	return url.PathEscape(s)
}
