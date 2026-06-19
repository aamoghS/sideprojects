package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	// --- 1. Create headless Chrome ---
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Enable network for cookies
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		panic(err)
	}

	// --- 2. Set session cookies ---
	cookies := []struct {
		Name, Value string
	}{
		{
			Name:  "lc_ea_po",
			Value: "0019ff6b70f438a9ab56eb016eec06725ce4d13782c6f21cce6f36c23671abfe43f629669e7a12431c3f52aa2a87dba9a287b97df09aa20adcd782726356dde3c7f1daea25059cba1ba4aa41831b6dab5b6feb70faac622fc90ffd222cfcc43ae5dec5083e4c9cf2f393ba034b28dd851a431b635b072eab3e6dac03e90b5932cf2266ce469c4e40bba0a9a3f5a8d0dcdf9265b7888d670545b7e98c6a03f65d14c89eb3700a031291bbb891ebe9259210a3dd348feecc6f4346e",
		},
		{
			Name:  "lc_ebcart",
			Value: "00107a769f1ef615a50ef2be9706a51b266205dcdab7d5e289eba728ad0a08a1c98279e572c1b631f501b2a4e240b08f48e4136d4f88b6b216e7a716135b57eb9e597dc965baff1a461af154b3152ce3d24720dc3888b36e382792453e5af015c554e6c953045676d254304dbe8652fa5b76d333ad1892774677ccd47947847bb4d6b10a6829295c6dbf76f569ddfd2c5ca26e100426599e8aae85c5d48e97b2dd099629c45bad5498698d7cf501b5cd2760d46a47d4e8919d2ea04ebb03960477597eddd8c44cbf7088c0c63fc68",
		},
	}

	for _, c := range cookies {
		if err := chromedp.Run(ctx, network.SetCookie(c.Name, c.Value).
			WithDomain("libcal.library.gatech.edu").
			WithPath("/").
			WithHTTPOnly(true),
		); err != nil {
			panic(err)
		}
	}

	// --- 3. Navigate and book ---
	var resultHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://libcal.library.gatech.edu/reserve/study-rooms"),
		chromedp.Sleep(2*time.Second), // wait for JS session to initialize

		// Click your room by eid
		chromedp.Click(`#eid_160969 button`, chromedp.NodeVisible),
		chromedp.Sleep(1*time.Second),

		// Fill required fields
		chromedp.SendKeys(`#nick`, "My Study Session", chromedp.NodeVisible),
		chromedp.SetValue(`#q22886`, "No", chromedp.NodeVisible),

		// Submit form
		chromedp.Click(`#btn-form-submit`, chromedp.NodeVisible),

		// Wait a second and grab the resulting HTML
		chromedp.Sleep(2*time.Second),
		chromedp.InnerHTML(`#s-lc-public-page-content`, &resultHTML, chromedp.NodeVisible),
	)
	if err != nil {
		panic(err)
	}

	// --- 4. Check booking result ---
	if strings.Contains(resultHTML, "Your booking is confirmed") ||
		strings.Contains(resultHTML, "Thank you for your booking") {
		fmt.Println("✅ Booking successful!")
	} else if strings.Contains(resultHTML, "already booked") ||
		strings.Contains(resultHTML, "cannot be booked") {
		fmt.Println("⚠️ Booking failed: room unavailable or already booked.")
	} else {
		fmt.Println("⚠️ Booking result unclear, check manually.")
	}

	// Optional: preview HTML snippet
	fmt.Println("Result preview:\n", resultHTML[:500])
}
