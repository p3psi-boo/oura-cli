package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	redirectURI = "http://localhost:8081/callback"
	authURL     = "https://cloud.ouraring.com/oauth/authorize"
	tokenURL    = "https://api.ouraring.com/oauth/token"
	apiBase     = "https://api.ouraring.com/v2/usercollection"
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

var config Config

func loadConfig() error {
	configPath := filepath.Join(getConfigDir(), "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("missing config: %s\nCreate it with:\n{\n  \"client_id\": \"your-id\",\n  \"client_secret\": \"your-secret\"\n}", configPath)
	}
	return json.Unmarshal(data, &config)
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type StoredToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Options struct {
	JSON bool
	Help bool
}

type ParsedArgs struct {
	Command string
	Args    []string
	Opts    Options
}

type EndpointResult struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type JSONOutput struct {
	Command   string                    `json:"command"`
	Date      string                    `json:"date"`
	StartDate string                    `json:"start_date"`
	EndDate   string                    `json:"end_date"`
	Endpoints map[string]EndpointResult `json:"endpoints"`
}

func main() {
	pa, ok := parseArgs(os.Args)
	if !ok {
		printUsage()
		os.Exit(1)
	}

	if err := loadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Global help routing.
	if pa.Opts.Help {
		printHelp(pa.Command, pa.Args)
		return
	}

	switch pa.Command {
	case "auth":
		doAuth()
	case "help":
		printHelp("", pa.Args)
	case "completion", "completions":
		handleCompletion(pa.Args)
	case "personal-info", "personal_info", "personal":
		handlePersonalInfo(pa.Args, pa.Opts)
	case "tag":
		handleTag(pa.Args, pa.Opts)
	case "enhanced-tag", "enhanced_tag":
		handleEnhancedTag(pa.Args, pa.Opts)
	case "session":
		handleSession(pa.Args, pa.Opts)
	case "webhook":
		handleWebhook(pa.Args, pa.Opts)
	case "today":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchAllJSON(date)
			return
		}
		fetchAll(date)
	case "sleep":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchSleepJSON(date)
			return
		}
		fetchSleep(date)
	case "activity":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchActivityJSON(date)
			return
		}
		fetchActivity(date)
	case "readiness":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchReadinessJSON(date)
			return
		}
		fetchReadiness(date)
	case "heartrate":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchHeartRateJSON(date)
			return
		}
		fetchHeartRate(date)
	case "hrv":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchHRVJSON(date)
			return
		}
		fetchHRV(date)
	case "stress":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchStressJSON(date)
			return
		}
		fetchStress(date)
	case "spo2":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchSpO2JSON(date)
			return
		}
		fetchSpO2(date)
	case "resilience":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchResilienceJSON(date)
			return
		}
		fetchResilience(date)
	case "vo2":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchVO2MaxJSON(date)
			return
		}
		fetchVO2Max(date)
	case "workout":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchWorkoutsJSON(date)
			return
		}
		fetchWorkouts(date)
	case "all":
		date := parseDateArg(pa.Args)
		if pa.Opts.JSON {
			fetchAllJSON(date)
			return
		}
		fetchAll(date)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`oura - Oura Ring CLI

Usage:
  oura <command> [args] [--json|-j]
	  oura help [command]
	  oura completion <bash|zsh|fish>

Commands:
  auth              Authenticate with Oura (first time setup)
  personal-info     Fetch personal info
  today             Show today's summary
  all [date]        Show all metrics for date (default: today)
  sleep [date]      Show sleep data
  activity [date]   Show activity data  
	  readiness [date]  Show readiness data
	  heartrate [date]  Show heart rate data
	  hrv [date]        Show heart rate variability (from sleep)
	  stress [date]     Show daytime stress data
	  spo2 [date]       Show blood oxygen data
	  resilience [date] Show resilience data
	  vo2 [date]        Show VO2 max data
	  workout [date]    Show workouts
  json [date]       Raw JSON dump of all data (alias for: all --json)

  tag               Manage tags
  enhanced-tag      Manage enhanced tags
  session           Manage sessions

  webhook           Manage webhook subscriptions

Webhook subcommands:
  webhook list
  webhook get <id>
  webhook create --callback-url <url> --verification-token <token> --event-type <create|update|delete> --data-type <type>
  webhook update <id> --verification-token <token> [--callback-url <url>] [--event-type <create|update|delete>] [--data-type <type>]
  webhook delete <id>
  webhook renew <id>
  webhook types

Options:
  --help, -h        Show help for a command
  --json, -j         Output JSON to stdout (machine readable)

Date format: YYYY-MM-DD (defaults to today)`)
}

func parseArgs(argv []string) (pa ParsedArgs, ok bool) {
	if len(argv) < 2 {
		return ParsedArgs{}, false
	}

	cmd := argv[1]
	if cmd == "--help" || cmd == "-h" {
		return ParsedArgs{Command: "help", Opts: Options{Help: true}}, true
	}
	args := argv[2:]
	var opts Options

	// Back-compat: `oura json [date]`.
	if cmd == "json" {
		opts.JSON = true
		cmd = "all"
	}

	// Parse global flags in a permissive way: allow them anywhere.
	pos := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--help", "-h", "help":
			opts.Help = true
		case "--json", "-j":
			opts.JSON = true
		default:
			pos = append(pos, a)
		}
	}

	return ParsedArgs{Command: cmd, Args: pos, Opts: opts}, true
}

func parseDateArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return time.Now().Format("2006-01-02")
}

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "oura")
	os.MkdirAll(dir, 0700)
	return dir
}

func getTokenPath() string {
	return filepath.Join(getConfigDir(), "token.json")
}

func saveToken(token *StoredToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getTokenPath(), data, 0600)
}

func loadToken() (*StoredToken, error) {
	data, err := os.ReadFile(getTokenPath())
	if err != nil {
		return nil, err
	}
	var token StoredToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func getValidToken() (string, error) {
	token, err := loadToken()
	if err != nil {
		return "", fmt.Errorf("not authenticated - run 'oura auth' first")
	}

	if time.Now().Add(5 * time.Minute).After(token.ExpiresAt) {
		newToken, err := refreshToken(token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("token refresh failed - run 'oura auth' again: %v", err)
		}
		token = newToken
	}

	return token.AccessToken, nil
}

func refreshToken(refresh string) (*StoredToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refresh)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	stored := &StoredToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	if err := saveToken(stored); err != nil {
		return nil, err
	}

	return stored, nil
}

func doAuth() {
	state := fmt.Sprintf("%d", time.Now().UnixNano())

	authParams := url.Values{}
	authParams.Set("client_id", config.ClientID)
	authParams.Set("redirect_uri", redirectURI)
	authParams.Set("response_type", "code")
	// Keep scopes broad enough for all CLI endpoints.
	authParams.Set("scope", "daily heartrate personal workout spo2 stress heart_health tag session")
	authParams.Set("state", state)

	fullAuthURL := authURL + "?" + authParams.Encode()

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{Addr: ":8081"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errChan <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			http.Error(w, "No code", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>✓ Authenticated!</h1><p>You can close this tab.</p></body></html>`)
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	fmt.Println("Opening browser for authentication...")
	fmt.Println("If it doesn't open, visit:")
	fmt.Println(fullAuthURL)
	openBrowser(fullAuthURL)

	select {
	case code := <-codeChan:
		server.Close()
		exchangeCode(code)
	case err := <-errChan:
		server.Close()
		fmt.Fprintf(os.Stderr, "Auth error: %v\n", err)
		os.Exit(1)
	case <-time.After(2 * time.Minute):
		server.Close()
		fmt.Fprintln(os.Stderr, "Auth timeout")
		os.Exit(1)
	}
}

func exchangeCode(code string) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Token exchange failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Token exchange failed: %s\n", body)
		os.Exit(1)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse token: %v\n", err)
		os.Exit(1)
	}

	stored := &StoredToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	if err := saveToken(stored); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Authenticated successfully!")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func apiGet(endpoint string, params url.Values) ([]byte, error) {
	token, err := getValidToken()
	if err != nil {
		return nil, err
	}

	url := apiBase + endpoint
	if len(params) > 0 {
		url += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}

	return body, nil
}

// Data types

type SleepResponse struct {
	Data []SleepRecord `json:"data"`
}

type SleepRecord struct {
	Day                string  `json:"day"`
	Type               string  `json:"type"`
	BedtimeStart       string  `json:"bedtime_start"`
	BedtimeEnd         string  `json:"bedtime_end"`
	TotalSleepDuration int     `json:"total_sleep_duration"`
	TimeInBed          int     `json:"time_in_bed"`
	Efficiency         int     `json:"efficiency"`
	DeepSleepDuration  int     `json:"deep_sleep_duration"`
	LightSleepDuration int     `json:"light_sleep_duration"`
	RemSleepDuration   int     `json:"rem_sleep_duration"`
	AwakeTime          int     `json:"awake_time"`
	Latency            int     `json:"latency"`
	LowestHeartRate    int     `json:"lowest_heart_rate"`
	AverageHeartRate   float64 `json:"average_heart_rate"`
	AverageHRV         int     `json:"average_hrv"`
	AverageBreath      float64 `json:"average_breath"`
	RestlessPeriods    int     `json:"restless_periods"`
}

type DailySleepResponse struct {
	Data []DailySleepRecord `json:"data"`
}

type DailySleepRecord struct {
	Day          string `json:"day"`
	Score        int    `json:"score"`
	Contributors struct {
		DeepSleep   int `json:"deep_sleep"`
		Efficiency  int `json:"efficiency"`
		Latency     int `json:"latency"`
		RemSleep    int `json:"rem_sleep"`
		Restfulness int `json:"restfulness"`
		Timing      int `json:"timing"`
		TotalSleep  int `json:"total_sleep"`
	} `json:"contributors"`
}

type ReadinessResponse struct {
	Data []ReadinessRecord `json:"data"`
}

type ReadinessRecord struct {
	Day                       string   `json:"day"`
	Score                     int      `json:"score"`
	TemperatureDeviation      float64  `json:"temperature_deviation"`
	TemperatureTrendDeviation *float64 `json:"temperature_trend_deviation"`
	Contributors              struct {
		ActivityBalance     int  `json:"activity_balance"`
		BodyTemperature     int  `json:"body_temperature"`
		HRVBalance          *int `json:"hrv_balance"`
		PreviousDayActivity int  `json:"previous_day_activity"`
		PreviousNight       int  `json:"previous_night"`
		RecoveryIndex       int  `json:"recovery_index"`
		RestingHeartRate    int  `json:"resting_heart_rate"`
		SleepBalance        *int `json:"sleep_balance"`
		SleepRegularity     *int `json:"sleep_regularity"`
	} `json:"contributors"`
}

type ActivityResponse struct {
	Data []ActivityRecord `json:"data"`
}

type ActivityRecord struct {
	Day                   string `json:"day"`
	Score                 int    `json:"score"`
	Steps                 int    `json:"steps"`
	ActiveCalories        int    `json:"active_calories"`
	TotalCalories         int    `json:"total_calories"`
	TargetCalories        int    `json:"target_calories"`
	EquivalentWalkingDist int    `json:"equivalent_walking_distance"`
	HighActivityTime      int    `json:"high_activity_time"`
	MediumActivityTime    int    `json:"medium_activity_time"`
	LowActivityTime       int    `json:"low_activity_time"`
	SedentaryTime         int    `json:"sedentary_time"`
	RestingTime           int    `json:"resting_time"`
}

type HeartRateResponse struct {
	Data []HeartRateRecord `json:"data"`
}

type HeartRateRecord struct {
	Timestamp string `json:"timestamp"`
	BPM       int    `json:"bpm"`
	Source    string `json:"source"`
}

type StressResponse struct {
	Data []StressRecord `json:"data"`
}

type StressRecord struct {
	Day           string  `json:"day"`
	StressHigh    int     `json:"stress_high"`
	RecoveryHigh  int     `json:"recovery_high"`
	DaytimeStress float64 `json:"day_summary"`
}

type SpO2Response struct {
	Data []SpO2Record `json:"data"`
}

type SpO2Record struct {
	Day            string `json:"day"`
	SpO2Percentage struct {
		Average float64 `json:"average"`
	} `json:"spo2_percentage"`
	BreathingDisturbanceIndex float64 `json:"breathing_disturbance_index"`
}

type ResilienceResponse struct {
	Data []ResilienceRecord `json:"data"`
}

type ResilienceRecord struct {
	Day          string `json:"day"`
	Level        string `json:"level"`
	Contributors struct {
		SleepRecovery   float64 `json:"sleep_recovery"`
		DaytimeRecovery float64 `json:"daytime_recovery"`
	} `json:"contributors"`
}

type VO2MaxResponse struct {
	Data []VO2MaxRecord `json:"data"`
}

type VO2MaxRecord struct {
	Day    string  `json:"day"`
	VO2Max float64 `json:"vo2_max"`
}

type WorkoutResponse struct {
	Data []WorkoutRecord `json:"data"`
}

type WorkoutRecord struct {
	Day           string  `json:"day"`
	Activity      string  `json:"activity"`
	Calories      float64 `json:"calories"`
	Distance      float64 `json:"distance"`
	StartDatetime string  `json:"start_datetime"`
	EndDatetime   string  `json:"end_datetime"`
	Intensity     string  `json:"intensity"`
	Label         *string `json:"label"`
	Source        string  `json:"source"`
}

// Fetch functions

func fetchSleep(date string) {
	targetDate, _ := time.Parse("2006-01-02", date)
	startDate := targetDate.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := targetDate.AddDate(0, 0, 1).Format("2006-01-02")

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	// Try daily_sleep first for the score
	dailyBody, dailyErr := apiGet("/daily_sleep", params)
	var dailyData DailySleepResponse
	var dailySleep *DailySleepRecord
	if dailyErr == nil {
		json.Unmarshal(dailyBody, &dailyData)
		for i := range dailyData.Data {
			if dailyData.Data[i].Day == date {
				dailySleep = &dailyData.Data[i]
				break
			}
		}
	}

	// Get detailed sleep periods
	body, err := apiGet("/sleep", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data SleepResponse
	json.Unmarshal(body, &data)

	// Collect all sleep records for this date
	var sleepRecords []SleepRecord
	for i := range data.Data {
		if data.Data[i].Day == date {
			sleepRecords = append(sleepRecords, data.Data[i])
		}
	}

	if len(sleepRecords) == 0 && dailySleep == nil {
		fmt.Println("No sleep data for", date)
		return
	}

	fmt.Printf("🌙 Sleep - %s\n", date)
	fmt.Println(strings.Repeat("─", 40))

	if dailySleep != nil {
		fmt.Printf("Score:         %d\n", dailySleep.Score)
		fmt.Println()
		fmt.Println("Contributors:")
		fmt.Printf("  Total Sleep:   %d\n", dailySleep.Contributors.TotalSleep)
		fmt.Printf("  Efficiency:    %d\n", dailySleep.Contributors.Efficiency)
		fmt.Printf("  Restfulness:   %d\n", dailySleep.Contributors.Restfulness)
		fmt.Printf("  REM Sleep:     %d\n", dailySleep.Contributors.RemSleep)
		fmt.Printf("  Deep Sleep:    %d\n", dailySleep.Contributors.DeepSleep)
		fmt.Printf("  Latency:       %d\n", dailySleep.Contributors.Latency)
		fmt.Printf("  Timing:        %d\n", dailySleep.Contributors.Timing)
		fmt.Println()
	}

	for i, s := range sleepRecords {
		bedStart, _ := time.Parse(time.RFC3339, s.BedtimeStart)
		bedEnd, _ := time.Parse(time.RFC3339, s.BedtimeEnd)
		bedStart = bedStart.Local()
		bedEnd = bedEnd.Local()

		// Label the sleep type
		sleepLabel := "😴 Nap"
		if s.Type == "long_sleep" {
			sleepLabel = "🛏️  Main Sleep"
		}

		if i > 0 {
			fmt.Println()
			fmt.Println(strings.Repeat("─", 40))
		}
		fmt.Printf("%s\n", sleepLabel)
		fmt.Printf("Time:          %s → %s\n", bedStart.Format("3:04 PM"), bedEnd.Format("3:04 PM"))
		fmt.Printf("Total Sleep:   %s\n", formatDuration(s.TotalSleepDuration))
		fmt.Printf("Time in Bed:   %s\n", formatDuration(s.TimeInBed))
		fmt.Printf("Efficiency:    %d%%\n", s.Efficiency)
		fmt.Println()
		fmt.Printf("Deep Sleep:    %s\n", formatDuration(s.DeepSleepDuration))
		fmt.Printf("Light Sleep:   %s\n", formatDuration(s.LightSleepDuration))
		fmt.Printf("REM Sleep:     %s\n", formatDuration(s.RemSleepDuration))
		fmt.Printf("Awake:         %s\n", formatDuration(s.AwakeTime))
		fmt.Printf("Latency:       %s\n", formatDuration(s.Latency))
		fmt.Println()
		fmt.Printf("Lowest HR:     %d bpm\n", s.LowestHeartRate)
		fmt.Printf("Average HR:    %.0f bpm\n", s.AverageHeartRate)
		fmt.Printf("Average HRV:   %d ms\n", s.AverageHRV)
		fmt.Printf("Breath Rate:   %.1f /min\n", s.AverageBreath)
		fmt.Printf("Restlessness:  %d periods\n", s.RestlessPeriods)
	}
}

func fetchHRV(date string) {
	targetDate, err := time.Parse("2006-01-02", date)
	startDate := date
	endDate := date
	if err == nil {
		startDate = targetDate.AddDate(0, 0, -1).Format("2006-01-02")
		endDate = targetDate.AddDate(0, 0, 1).Format("2006-01-02")
	}

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	body, err := apiGet("/sleep", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data SleepResponse
	json.Unmarshal(body, &data)

	// Collect all sleep records for this date.
	var records []SleepRecord
	for i := range data.Data {
		if data.Data[i].Day == date {
			records = append(records, data.Data[i])
		}
	}

	if len(records) == 0 {
		fmt.Println("No HRV data for", date)
		return
	}

	fmt.Printf("💓 HRV - %s\n", date)
	fmt.Println(strings.Repeat("─", 40))

	for i, s := range records {
		bedStart, _ := time.Parse(time.RFC3339, s.BedtimeStart)
		bedEnd, _ := time.Parse(time.RFC3339, s.BedtimeEnd)
		bedStart = bedStart.Local()
		bedEnd = bedEnd.Local()

		label := "😴 Nap"
		if s.Type == "long_sleep" {
			label = "🛏️  Main Sleep"
		}

		if i > 0 {
			fmt.Println()
		}

		hrv := "n/a"
		if s.AverageHRV > 0 {
			hrv = fmt.Sprintf("%d ms", s.AverageHRV)
		}

		fmt.Printf("%s (%s → %s)\n", label, bedStart.Format("3:04 PM"), bedEnd.Format("3:04 PM"))
		fmt.Printf("Average HRV:   %s\n", hrv)
		fmt.Printf("Average HR:    %.0f bpm\n", s.AverageHeartRate)
		fmt.Printf("Lowest HR:     %d bpm\n", s.LowestHeartRate)
	}
}

func fetchReadiness(date string) {
	targetDate, _ := time.Parse("2006-01-02", date)
	startDate := targetDate.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := targetDate.AddDate(0, 0, 1).Format("2006-01-02")

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	body, err := apiGet("/daily_readiness", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data ReadinessResponse
	json.Unmarshal(body, &data)

	var r *ReadinessRecord
	for i := range data.Data {
		if data.Data[i].Day == date {
			r = &data.Data[i]
			break
		}
	}

	if r == nil {
		fmt.Println("No readiness data for", date)
		return
	}

	c := r.Contributors

	fmt.Printf("💪 Readiness - %s\n", r.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Score:              %d\n", r.Score)
	fmt.Printf("Temp Deviation:     %+.2f°C\n", r.TemperatureDeviation)
	fmt.Println()
	fmt.Println("Contributors:")
	fmt.Printf("  Resting HR:       %d\n", c.RestingHeartRate)
	if c.HRVBalance != nil {
		fmt.Printf("  HRV Balance:      %d\n", *c.HRVBalance)
	}
	fmt.Printf("  Body Temp:        %d\n", c.BodyTemperature)
	fmt.Printf("  Recovery Index:   %d\n", c.RecoveryIndex)
	fmt.Printf("  Previous Night:   %d\n", c.PreviousNight)
	fmt.Printf("  Prev Day Activity:%d\n", c.PreviousDayActivity)
	fmt.Printf("  Activity Balance: %d\n", c.ActivityBalance)
	if c.SleepBalance != nil {
		fmt.Printf("  Sleep Balance:    %d\n", *c.SleepBalance)
	}
	if c.SleepRegularity != nil {
		fmt.Printf("  Sleep Regularity: %d\n", *c.SleepRegularity)
	}
}

func fetchActivity(date string) {
	targetDate, _ := time.Parse("2006-01-02", date)
	startDate := targetDate.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := targetDate.AddDate(0, 0, 1).Format("2006-01-02")

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	body, err := apiGet("/daily_activity", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data ActivityResponse
	json.Unmarshal(body, &data)

	var a *ActivityRecord
	for i := range data.Data {
		if data.Data[i].Day == date {
			a = &data.Data[i]
			break
		}
	}

	if a == nil {
		fmt.Println("No activity data for", date)
		return
	}

	fmt.Printf("🏃 Activity - %s\n", a.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Score:         %d\n", a.Score)
	fmt.Printf("Steps:         %d\n", a.Steps)
	fmt.Printf("Distance:      %.1f km\n", float64(a.EquivalentWalkingDist)/1000)
	fmt.Println()
	fmt.Printf("Active Cal:    %d\n", a.ActiveCalories)
	fmt.Printf("Total Cal:     %d\n", a.TotalCalories)
	fmt.Printf("Target Cal:    %d\n", a.TargetCalories)
	fmt.Println()
	fmt.Printf("High Activity: %s\n", formatDuration(a.HighActivityTime))
	fmt.Printf("Med Activity:  %s\n", formatDuration(a.MediumActivityTime))
	fmt.Printf("Low Activity:  %s\n", formatDuration(a.LowActivityTime))
	fmt.Printf("Sedentary:     %s\n", formatDuration(a.SedentaryTime))
	fmt.Printf("Resting:       %s\n", formatDuration(a.RestingTime))
}

func fetchHeartRate(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/heartrate", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data HeartRateResponse
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No heart rate data for", date)
		return
	}

	var min, max, sum int
	min = 999
	for _, hr := range data.Data {
		if hr.BPM < min {
			min = hr.BPM
		}
		if hr.BPM > max {
			max = hr.BPM
		}
		sum += hr.BPM
	}
	avg := sum / len(data.Data)

	fmt.Printf("❤️  Heart Rate - %s\n", date)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Readings:  %d\n", len(data.Data))
	fmt.Printf("Min:       %d bpm\n", min)
	fmt.Printf("Max:       %d bpm\n", max)
	fmt.Printf("Average:   %d bpm\n", avg)
}

func fetchStress(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/daily_stress", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data StressResponse
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No stress data for", date)
		return
	}

	s := data.Data[0]

	fmt.Printf("😤 Stress - %s\n", s.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Stress High:     %d min\n", s.StressHigh)
	fmt.Printf("Recovery High:   %d min\n", s.RecoveryHigh)
}

func fetchSpO2(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/daily_spo2", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data SpO2Response
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No SpO2 data for", date)
		return
	}

	s := data.Data[0]

	fmt.Printf("🫁 Blood Oxygen - %s\n", s.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Average SpO2:    %.1f%%\n", s.SpO2Percentage.Average)
	fmt.Printf("Breathing Index: %.2f\n", s.BreathingDisturbanceIndex)
}

func fetchResilience(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/daily_resilience", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data ResilienceResponse
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No resilience data for", date)
		return
	}

	r := data.Data[0]

	fmt.Printf("🛡️  Resilience - %s\n", r.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Level:            %s\n", r.Level)
	fmt.Printf("Sleep Recovery:   %.0f%%\n", r.Contributors.SleepRecovery*100)
	fmt.Printf("Daytime Recovery: %.0f%%\n", r.Contributors.DaytimeRecovery*100)
}

func fetchVO2Max(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/vO2_max", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data VO2MaxResponse
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No VO2 max data for", date)
		return
	}

	v := data.Data[0]

	fmt.Printf("🏋️  VO2 Max - %s\n", v.Day)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("VO2 Max:  %.1f ml/kg/min\n", v.VO2Max)
}

func fetchWorkouts(date string) {
	params := url.Values{}
	params.Set("start_date", date)
	params.Set("end_date", date)

	body, err := apiGet("/workout", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var data WorkoutResponse
	json.Unmarshal(body, &data)

	if len(data.Data) == 0 {
		fmt.Println("No workout data for", date)
		return
	}

	fmt.Printf("🏋️  Workouts - %s\n", date)
	fmt.Println(strings.Repeat("─", 40))

	for i, w := range data.Data {
		if i > 0 {
			fmt.Println()
		}

		startTime, _ := time.Parse(time.RFC3339, w.StartDatetime)
		endTime, _ := time.Parse(time.RFC3339, w.EndDatetime)
		startTime = startTime.Local()
		endTime = endTime.Local()
		duration := endTime.Sub(startTime)

		label := w.Activity
		if w.Label != nil && *w.Label != "" {
			label = *w.Label
		}

		fmt.Printf("Activity:   %s\n", label)
		fmt.Printf("Time:       %s (%s)\n", startTime.Format("3:04 PM"), formatDuration(int(duration.Seconds())))
		fmt.Printf("Calories:   %.0f\n", w.Calories)
		if w.Distance > 0 {
			fmt.Printf("Distance:   %.2f km\n", w.Distance/1000)
		}
		fmt.Printf("Intensity:  %s\n", w.Intensity)
		fmt.Printf("Source:     %s\n", w.Source)
	}
}

func fetchAll(date string) {
	fmt.Printf("╔══════════════════════════════════════╗\n")
	fmt.Printf("║      OURA METRICS - %-10s       ║\n", date)
	fmt.Printf("╚══════════════════════════════════════╝\n\n")

	fetchReadiness(date)
	fmt.Println()
	fetchSleep(date)
	fmt.Println()
	fetchActivity(date)
	fmt.Println()
	fetchStress(date)
	fmt.Println()
	fetchHeartRate(date)
}

func writeJSONToStdout(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func paddedDateRange(date string, beforeDays int, afterDays int) (startDate string, endDate string) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		// Keep behavior predictable even on bad input.
		return date, date
	}
	startDate = t.AddDate(0, 0, -beforeDays).Format("2006-01-02")
	endDate = t.AddDate(0, 0, afterDays).Format("2006-01-02")
	return startDate, endDate
}

func fetchEndpointsJSON(command string, date string, startDate string, endDate string, endpoints []string) {
	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	out := JSONOutput{
		Command:   command,
		Date:      date,
		StartDate: startDate,
		EndDate:   endDate,
		Endpoints: make(map[string]EndpointResult, len(endpoints)),
	}

	for _, ep := range endpoints {
		name := strings.TrimPrefix(ep, "/")
		body, err := apiGet(ep, params)
		if err != nil {
			out.Endpoints[name] = EndpointResult{Error: err.Error()}
			continue
		}
		out.Endpoints[name] = EndpointResult{Data: json.RawMessage(body)}
	}

	writeJSONToStdout(out)
}

func fetchSleepJSON(date string) {
	startDate, endDate := paddedDateRange(date, 1, 1)
	fetchEndpointsJSON("sleep", date, startDate, endDate, []string{"/sleep", "/daily_sleep"})
}

func fetchActivityJSON(date string) {
	startDate, endDate := paddedDateRange(date, 1, 1)
	fetchEndpointsJSON("activity", date, startDate, endDate, []string{"/daily_activity"})
}

func fetchReadinessJSON(date string) {
	startDate, endDate := paddedDateRange(date, 1, 1)
	fetchEndpointsJSON("readiness", date, startDate, endDate, []string{"/daily_readiness"})
}

func fetchHeartRateJSON(date string) {
	fetchEndpointsJSON("heartrate", date, date, date, []string{"/heartrate"})
}

func fetchHRVJSON(date string) {
	startDate, endDate := paddedDateRange(date, 1, 1)
	// HRV is primarily exposed via sleep; readiness can include HRV-related contributors.
	fetchEndpointsJSON("hrv", date, startDate, endDate, []string{"/sleep", "/daily_sleep", "/daily_readiness"})
}

func fetchStressJSON(date string) {
	fetchEndpointsJSON("stress", date, date, date, []string{"/daily_stress"})
}

func fetchSpO2JSON(date string) {
	fetchEndpointsJSON("spo2", date, date, date, []string{"/daily_spo2"})
}

func fetchResilienceJSON(date string) {
	fetchEndpointsJSON("resilience", date, date, date, []string{"/daily_resilience"})
}

func fetchVO2MaxJSON(date string) {
	fetchEndpointsJSON("vo2", date, date, date, []string{"/vO2_max"})
}

func fetchWorkoutsJSON(date string) {
	fetchEndpointsJSON("workout", date, date, date, []string{"/workout"})
}

func fetchAllJSON(date string) {
	startDate, endDate := paddedDateRange(date, 1, 1)
	fetchEndpointsJSON("all", date, startDate, endDate, []string{
		"/sleep",
		"/daily_sleep",
		"/daily_activity",
		"/daily_readiness",
		"/heartrate",
		"/daily_stress",
		"/daily_spo2",
		"/daily_resilience",
		"/vO2_max",
		"/workout",
	})
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
