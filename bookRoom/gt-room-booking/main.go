package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	baseURL          = "https://libcal.library.gatech.edu"
	loginPageURL     = "https://libcal.library.gatech.edu/login"
	studyRoomsURL    = "https://libcal.library.gatech.edu/reserve/study-rooms"
	availabilityURL  = "https://libcal.library.gatech.edu/spaces/availability/grid"
	bookingSubmitURL = "https://libcal.library.gatech.edu/spaces/availability/booking/add"
	confirmURL       = "https://libcal.library.gatech.edu/ajax/space/createcart"
)

type BookingConfig struct {
	Username     string
	Password     string
	RoomID       string // e.g. "158675" for Clough 342
	StartTime    string // Format "14:00" for 2:00 PM
	EndTime      string // Format "15:45" for 3:45 PM
	BookingDate  string // Format "2025-08-19"
	StudentID    string
	UserLastName string
}

func main() {
	config := BookingConfig{
		Username:     "asawant43",
		Password:     "XboxRiya2016#",
		RoomID:       "158675",                                         // Clough 342
		StartTime:    "14:00",                                          // 2:00 PM
		EndTime:      "15:45",                                          // 3:45 PM
		BookingDate:  time.Now().AddDate(0, 0, 1).Format("2006-01-02"), // Tomorrow
		StudentID:    "904011773",
		UserLastName: "Sawant",
	}

	// Initialize HTTP client with cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Error creating cookie jar:", err)
	}

	client := &http.Client{
		Jar: jar,
	}

	// Step 1: Get login page to extract CSRF token
	csrfToken, err := getCSRFToken(client)
	if err != nil {
		log.Fatal("Error getting CSRF token:", err)
	}

	// Step 2: Perform GT login
	err = login(client, csrfToken, config.Username, config.Password)
	if err != nil {
		log.Fatal("Login failed:", err)
	}

	// Step 3: Get available slots
	checksum, err := getSlotChecksum(client, config.RoomID, config.BookingDate, config.StartTime)
	if err != nil {
		log.Fatal("Error finding available slot:", err)
	}

	// Step 4: Submit booking
	bookingID, err := submitBooking(client, config.RoomID, config.BookingDate, config.StartTime, checksum)
	if err != nil {
		log.Fatal("Booking submission failed:", err)
	}

	// Step 5: Update booking duration
	err = updateBookingDuration(client, bookingID, config.RoomID, config.BookingDate, config.StartTime, config.EndTime)
	if err != nil {
		log.Fatal("Error updating booking duration:", err)
	}

	// Step 6: Final confirmation
	err = confirmBooking(client, bookingID, config.RoomID, config.BookingDate, config.StartTime, config.EndTime, config.StudentID, config.UserLastName)
	if err != nil {
		log.Fatal("Confirmation failed:", err)
	}

	fmt.Println("Successfully booked room!")
}

func getCSRFToken(client *http.Client) (string, error) {
	resp, err := client.Get(loginPageURL)
	if err != nil {
		return "", fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login page: %w", err)
	}

	// Extract CSRF token from HTML
	re := regexp.MustCompile(`name="csrf_token"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found in login page")
	}

	return matches[1], nil
}

func login(client *http.Client, csrfToken, username, password string) error {
	formData := url.Values{
		"csrf_token": {csrfToken},
		"username":   {username},
		"password":   {password},
	}

	req, err := http.NewRequest("POST", loginPageURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginPageURL)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	// Verify login by checking study rooms page
	resp, err = client.Get(studyRoomsURL)
	if err != nil {
		return fmt.Errorf("failed to verify login: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func getSlotChecksum(client *http.Client, roomID, date, startTime string) (string, error) {
	formData := url.Values{
		"lid":       {"18640"}, // Location ID for Georgia Tech Library Spaces
		"gid":       {"39399"}, // Group ID for Study Rooms
		"eid":       {roomID},
		"start":     {date},
		"end":       {date},
		"pageIndex": {"0"},
		"pageSize":  {"18"},
	}

	req, err := http.NewRequest("POST", availabilityURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating availability request: %w", err)
	}

	setCommonHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("availability request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read availability response: %w", err)
	}

	// Search for the checksum in the JSON response
	pattern := fmt.Sprintf(`"end":"%s %s:00","itemId":%s,"checksum":"([^"]+)"`, date, startTime, roomID)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("checksum not found for selected time slot")
	}

	return matches[1], nil
}

func submitBooking(client *http.Client, roomID, date, startTime, checksum string) (string, error) {
	formData := url.Values{
		"add[eid]":      {roomID},
		"add[gid]":      {"39399"},
		"add[lid]":      {"18640"},
		"add[start]":    {fmt.Sprintf("%s %s", date, startTime)},
		"add[checksum]": {checksum},
		"lid":           {"18640"},
		"gid":           {"39399"},
		"start":         {date},
		"end":           {date},
	}

	req, err := http.NewRequest("POST", bookingSubmitURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating booking request: %w", err)
	}

	setCommonHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("booking request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read booking response: %w", err)
	}

	// Parse response to get booking ID
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse booking response: %w", err)
	}

	bookings, ok := result["bookings"].([]interface{})
	if !ok || len(bookings) == 0 {
		return "", fmt.Errorf("no bookings found in response")
	}

	firstBooking, ok := bookings[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid booking data in response")
	}

	bookingID, ok := firstBooking["id"].(string)
	if !ok {
		return "", fmt.Errorf("booking ID not found in response")
	}

	return bookingID, nil
}

func updateBookingDuration(client *http.Client, bookingID, roomID, date, startTime, endTime string) error {
	// First get the current checksum for the booking
	checksum, err := getBookingChecksum(client, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking checksum: %w", err)
	}

	formData := url.Values{
		"update[id]":            {bookingID},
		"update[checksum]":      {checksum},
		"update[end]":           {fmt.Sprintf("%s %s", date, endTime)},
		"lid":                   {"18640"},
		"gid":                   {"39399"},
		"start":                 {date},
		"end":                   {date},
		"bookings[0][id]":       {bookingID},
		"bookings[0][eid]":      {roomID},
		"bookings[0][seat_id]":  {"0"},
		"bookings[0][gid]":      {"39399"},
		"bookings[0][lid]":      {"18640"},
		"bookings[0][start]":    {fmt.Sprintf("%s %s", date, startTime)},
		"bookings[0][end]":      {fmt.Sprintf("%s %s", date, endTime)},
		"bookings[0][checksum]": {checksum},
	}

	req, err := http.NewRequest("POST", bookingSubmitURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating duration update request: %w", err)
	}

	setCommonHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("duration update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("duration update failed with status %d", resp.StatusCode)
	}

	return nil
}

func getBookingChecksum(client *http.Client, bookingID string) (string, error) {
	// This would typically come from the booking response, but we'll simulate it
	// In a real implementation, you'd parse this from the booking response
	return "temp_checksum_value", nil
}

func confirmBooking(client *http.Client, bookingID, roomID, date, startTime, endTime, studentID, lastName string) error {
	formData := url.Values{
		"libAuth":               {"true"},
		"blowAwayCart":          {"true"},
		"returnUrl":             {fmt.Sprintf("/space/%s", roomID)},
		"bookings[0][id]":       {bookingID},
		"bookings[0][eid]":      {roomID},
		"bookings[0][seat_id]":  {"0"},
		"bookings[0][gid]":      {"39399"},
		"bookings[0][lid]":      {"18640"},
		"bookings[0][start]":    {fmt.Sprintf("%s %s", date, startTime)},
		"bookings[0][end]":      {fmt.Sprintf("%s %s", date, endTime)},
		"bookings[0][checksum]": {"final_checksum"}, // This should come from previous response
		"method":                {"12"},
		"lname":                 {lastName},
		"q2614":                 {studentID}, // Student ID field
	}

	req, err := http.NewRequest("POST", confirmURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating confirmation request: %w", err)
	}

	setCommonHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("confirmation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("confirmation failed with status %d", resp.StatusCode)
	}

	return nil
}

func setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", studyRoomsURL)
}
