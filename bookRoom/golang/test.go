package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("missing required environment variable %s", key)
	}
	return val
}

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		log.Fatalf("enable network: %v", err)
	}

	cookies := []struct {
		envKey string
		name   string
	}{
		{envKey: "LC_EA_PO", name: "lc_ea_po"},
		{envKey: "LC_EBCART", name: "lc_ebcart"},
	}

	for _, c := range cookies {
		value := requireEnv(c.envKey)
		if err := chromedp.Run(ctx, network.SetCookie(c.name, value).
			WithDomain("libcal.library.gatech.edu").
			WithPath("/").
			WithHTTPOnly(true),
		); err != nil {
			log.Fatalf("set cookie %s: %v", c.name, err)
		}
	}

	var resultHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://libcal.library.gatech.edu/reserve/study-rooms"),
		chromedp.Sleep(2*time.Second),

		chromedp.Click(`#eid_160969 button`, chromedp.NodeVisible),
		chromedp.Sleep(1*time.Second),

		chromedp.SendKeys(`#nick`, "My Study Session", chromedp.NodeVisible),
		chromedp.SetValue(`#q22886`, "No", chromedp.NodeVisible),

		chromedp.Click(`#btn-form-submit`, chromedp.NodeVisible),

		chromedp.Sleep(2*time.Second),
		chromedp.InnerHTML(`#s-lc-public-page-content`, &resultHTML, chromedp.NodeVisible),
	)
	if err != nil {
		log.Fatalf("booking flow: %v", err)
	}

	if strings.Contains(resultHTML, "Your booking is confirmed") ||
		strings.Contains(resultHTML, "Thank you for your booking") {
		fmt.Println("Booking successful!")
	} else if strings.Contains(resultHTML, "already booked") ||
		strings.Contains(resultHTML, "cannot be booked") {
		fmt.Println("Booking failed: room unavailable or already booked.")
	} else {
		fmt.Println("Booking result unclear, check manually.")
	}

	previewLen := 500
	if len(resultHTML) < previewLen {
		previewLen = len(resultHTML)
	}
	fmt.Println("Result preview:\n", resultHTML[:previewLen])
}
