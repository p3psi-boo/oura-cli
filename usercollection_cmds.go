package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type MultiDocumentResponse[T any] struct {
	Data      []T    `json:"data"`
	NextToken string `json:"next_token"`
}

type PersonalInfoResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Age           any    `json:"age"`
	BiologicalSex string `json:"biological_sex"`
	Height        any    `json:"height"`
	Weight        any    `json:"weight"`
}

type TagModel struct {
	ID        string   `json:"id"`
	Day       string   `json:"day"`
	Timestamp string   `json:"timestamp"`
	Text      string   `json:"text"`
	Tags      []string `json:"tags"`
}

type EnhancedTagModel struct {
	ID          string `json:"id"`
	TagTypeCode string `json:"tag_type_code"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	StartDay    string `json:"start_day"`
	EndDay      string `json:"end_day"`
	Comment     string `json:"comment"`
	CustomName  string `json:"custom_name"`
}

type SessionModel struct {
	ID                   string          `json:"id"`
	Day                  string          `json:"day"`
	StartDatetime        string          `json:"start_datetime"`
	EndDatetime          string          `json:"end_datetime"`
	Type                 string          `json:"type"`
	HeartRate            json.RawMessage `json:"heart_rate"`
	HeartRateVariability json.RawMessage `json:"heart_rate_variability"`
	Mood                 string          `json:"mood"`
	MotionCount          json.RawMessage `json:"motion_count"`
}

func printPersonalInfoUsage() {
	fmt.Print(`Personal info

Usage:
  oura personal-info [--json|-j]
  oura personal-info get [--json|-j]
`)
}

func printTagUsage() {
	fmt.Print(`Tags

Usage:
  oura tag [list] [--start-date <date>] [--end-date <date>] [--next-token <token>] [--json|-j]
  oura tag get <document_id> [--json|-j]
`)
}

func printEnhancedTagUsage() {
	fmt.Print(`Enhanced tags

Usage:
  oura enhanced-tag [list] [--start-date <date>] [--end-date <date>] [--next-token <token>] [--json|-j]
  oura enhanced-tag get <document_id> [--json|-j]
`)
}

func printSessionUsage() {
	fmt.Print(`Sessions

Usage:
  oura session [list] [--start-date <date>] [--end-date <date>] [--next-token <token>] [--json|-j]
  oura session get <document_id> [--json|-j]
`)
}

func handlePersonalInfo(args []string, opts Options) {
	if opts.Help {
		printPersonalInfoUsage()
		return
	}
	if len(args) > 1 {
		printPersonalInfoUsage()
		os.Exit(1)
	}
	if len(args) == 1 && args[0] != "get" {
		printPersonalInfoUsage()
		os.Exit(1)
	}

	body, err := apiGet("/personal_info", nil)
	if err != nil {
		exitErr(err)
	}

	if opts.JSON {
		writeJSON(body)
		return
	}

	var pi PersonalInfoResponse
	if err := json.Unmarshal(body, &pi); err != nil {
		exitErr(fmt.Errorf("failed to parse response: %w", err))
	}

	fmt.Println("Personal info")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:       %s\n", pi.ID)
	if pi.Email != "" {
		fmt.Printf("Email:    %s\n", pi.Email)
	}
	if pi.BiologicalSex != "" {
		fmt.Printf("Sex:      %s\n", pi.BiologicalSex)
	}
	if pi.Age != nil {
		fmt.Printf("Age:      %v\n", pi.Age)
	}
	if pi.Height != nil {
		fmt.Printf("Height:   %v\n", pi.Height)
	}
	if pi.Weight != nil {
		fmt.Printf("Weight:   %v\n", pi.Weight)
	}
}

func handleTag(args []string, opts Options) {
	handleListGet("tag", args, opts)
}

func handleEnhancedTag(args []string, opts Options) {
	handleListGet("enhanced_tag", args, opts)
}

func handleSession(args []string, opts Options) {
	handleListGet("session", args, opts)
}

func handleListGet(kind string, args []string, opts Options) {
	if opts.Help {
		switch kind {
		case "tag":
			printTagUsage()
		case "enhanced_tag":
			printEnhancedTagUsage()
		case "session":
			printSessionUsage()
		default:
			printUsage()
		}
		return
	}

	// Allow: `oura tag --start-date ...` (implicit list).
	sub := "list"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		sub = args[0]
		rest = args[1:]
	}

	switch sub {
	case "list":
		params, extra, err := parseRangeQueryFlags(rest)
		if err != nil {
			exitErr(err)
		}
		if len(extra) != 0 {
			exitErr(fmt.Errorf("unexpected args: %s", strings.Join(extra, " ")))
		}
		switch kind {
		case "tag":
			listAndPrint[TagModel]("/tag", params, opts, printTagList)
		case "enhanced_tag":
			listAndPrint[EnhancedTagModel]("/enhanced_tag", params, opts, printEnhancedTagList)
		case "session":
			listAndPrint[SessionModel]("/session", params, opts, printSessionList)
		}
	case "get":
		if len(rest) != 1 {
			exitErr(fmt.Errorf("missing document_id"))
		}
		id := rest[0]
		switch kind {
		case "tag":
			getAndPrint[TagModel]("/tag/"+url.PathEscape(id), opts, printTag)
		case "enhanced_tag":
			getAndPrint[EnhancedTagModel]("/enhanced_tag/"+url.PathEscape(id), opts, printEnhancedTag)
		case "session":
			getAndPrint[SessionModel]("/session/"+url.PathEscape(id), opts, printSession)
		}
	default:
		switch kind {
		case "tag":
			printTagUsage()
		case "enhanced_tag":
			printEnhancedTagUsage()
		case "session":
			printSessionUsage()
		}
		os.Exit(1)
	}
}

func parseRangeQueryFlags(args []string) (params url.Values, rest []string, err error) {
	flags, pos, err := parseLongFlags(args)
	if err != nil {
		return nil, nil, err
	}
	params = url.Values{}
	if v := firstFlag(flags, "start-date", "start_date"); v != "" {
		params.Set("start_date", v)
	}
	if v := firstFlag(flags, "end-date", "end_date"); v != "" {
		params.Set("end_date", v)
	}
	if v := firstFlag(flags, "next-token", "next_token"); v != "" {
		params.Set("next_token", v)
	}
	return params, pos, nil
}

func listAndPrint[T any](endpoint string, params url.Values, opts Options, printer func(MultiDocumentResponse[T])) {
	body, err := apiGet(endpoint, params)
	if err != nil {
		exitErr(err)
	}
	if opts.JSON {
		writeJSON(body)
		return
	}
	var resp MultiDocumentResponse[T]
	if err := json.Unmarshal(body, &resp); err != nil {
		exitErr(fmt.Errorf("failed to parse response: %w", err))
	}
	printer(resp)
}

func getAndPrint[T any](endpoint string, opts Options, printer func(T)) {
	body, err := apiGet(endpoint, nil)
	if err != nil {
		exitErr(err)
	}
	if opts.JSON {
		writeJSON(body)
		return
	}
	var doc T
	if err := json.Unmarshal(body, &doc); err != nil {
		exitErr(fmt.Errorf("failed to parse response: %w", err))
	}
	printer(doc)
}

func writeJSON(body []byte) {
	os.Stdout.Write(body)
	if len(body) == 0 || body[len(body)-1] != '\n' {
		fmt.Println()
	}
}

func printNextToken(next string) {
	if next == "" {
		return
	}
	fmt.Println(strings.Repeat("-", 72))
	fmt.Printf("next_token: %s\n", next)
}

func printTagList(resp MultiDocumentResponse[TagModel]) {
	if len(resp.Data) == 0 {
		fmt.Println("No tags")
		return
	}
	fmt.Printf("Tags (%d)\n", len(resp.Data))
	fmt.Println(strings.Repeat("-", 72))
	for _, t := range resp.Data {
		label := t.Text
		if label == "" {
			label = strings.Join(t.Tags, ",")
		}
		fmt.Printf("%s  %s  %s\n", t.ID, t.Day, truncate(label, 60))
	}
	printNextToken(resp.NextToken)
}

func printTag(t TagModel) {
	fmt.Println("Tag")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", t.ID)
	fmt.Printf("Day:       %s\n", t.Day)
	if t.Timestamp != "" {
		fmt.Printf("Timestamp: %s\n", t.Timestamp)
	}
	if t.Text != "" {
		fmt.Printf("Text:      %s\n", t.Text)
	}
	if len(t.Tags) > 0 {
		fmt.Printf("Tags:      %s\n", strings.Join(t.Tags, ", "))
	}
}

func printEnhancedTagList(resp MultiDocumentResponse[EnhancedTagModel]) {
	if len(resp.Data) == 0 {
		fmt.Println("No enhanced tags")
		return
	}
	fmt.Printf("Enhanced tags (%d)\n", len(resp.Data))
	fmt.Println(strings.Repeat("-", 72))
	for _, t := range resp.Data {
		days := t.StartDay
		if t.EndDay != "" && t.EndDay != t.StartDay {
			days = t.StartDay + ".." + t.EndDay
		}
		label := firstNonEmpty(t.CustomName, t.Comment, t.TagTypeCode)
		fmt.Printf("%s  %s  %s\n", t.ID, days, truncate(label, 60))
	}
	printNextToken(resp.NextToken)
}

func printEnhancedTag(t EnhancedTagModel) {
	fmt.Println("Enhanced tag")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:        %s\n", t.ID)
	if t.TagTypeCode != "" {
		fmt.Printf("Type:      %s\n", t.TagTypeCode)
	}
	if t.StartDay != "" {
		fmt.Printf("Start day: %s\n", t.StartDay)
	}
	if t.EndDay != "" {
		fmt.Printf("End day:   %s\n", t.EndDay)
	}
	if t.StartTime != "" {
		fmt.Printf("Start:     %s\n", t.StartTime)
	}
	if t.EndTime != "" {
		fmt.Printf("End:       %s\n", t.EndTime)
	}
	if t.CustomName != "" {
		fmt.Printf("Name:      %s\n", t.CustomName)
	}
	if t.Comment != "" {
		fmt.Printf("Comment:   %s\n", t.Comment)
	}
}

func printSessionList(resp MultiDocumentResponse[SessionModel]) {
	if len(resp.Data) == 0 {
		fmt.Println("No sessions")
		return
	}
	fmt.Printf("Sessions (%d)\n", len(resp.Data))
	fmt.Println(strings.Repeat("-", 72))
	for _, s := range resp.Data {
		label := firstNonEmpty(s.Type, s.Mood)
		fmt.Printf("%s  %s  %s\n", s.ID, s.Day, truncate(label, 60))
	}
	printNextToken(resp.NextToken)
}

func printSession(s SessionModel) {
	fmt.Println("Session")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("ID:    %s\n", s.ID)
	fmt.Printf("Day:   %s\n", s.Day)
	if s.Type != "" {
		fmt.Printf("Type:  %s\n", s.Type)
	}
	if s.Mood != "" {
		fmt.Printf("Mood:  %s\n", s.Mood)
	}
	if s.StartDatetime != "" {
		fmt.Printf("Start: %s\n", s.StartDatetime)
	}
	if s.EndDatetime != "" {
		fmt.Printf("End:   %s\n", s.EndDatetime)
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
