package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type Price struct {
	Date        string `json:"date"`
	K22         int    `json:"k22"`
	K21         int    `json:"k21"`
	K18         int    `json:"k18"`
	Traditional int    `json:"traditional"`
}

func writeRow(row *[]string, priceData *Price) {
	(*row)[0] = priceData.Date
	(*row)[1] = strconv.Itoa(priceData.K18)
	(*row)[2] = strconv.Itoa(priceData.K21)
	(*row)[3] = strconv.Itoa(priceData.K22)
	(*row)[4] = strconv.Itoa(priceData.Traditional)
}

func main() {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	todayPrice := Price{}
	todayPrice.Date = time.Now().Format("2006-01-02")

	fmt.Println("DEBUG: Starting scraper")
	fmt.Println("DEBUG: Server time is", time.Now().Format(time.RFC3339))
	fmt.Println("DEBUG: Today's date used:", todayPrice.Date)

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("DEBUG: Sending request to", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("ERROR: Request failed:", err)
		if r != nil {
			fmt.Println("ERROR: HTTP status code:", r.StatusCode)
			fmt.Println("ERROR: Response body length:", len(r.Body), "bytes")
		}
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("DEBUG: Got response. Status:", r.StatusCode, "Body length:", len(r.Body), "bytes")
	})

	getPrice := func(e *colly.HTMLElement) int {
		priceStr := strings.NewReplacer(",", "", " BDT/GRAM", "").Replace(e.Text)
		price, err := strconv.Atoi(priceStr)
		if err != nil {
			fmt.Println("DEBUG: Failed to parse price text:", e.Text, "error:", err)
		}
		return price
	}
	c.OnHTML(".gold-table tr:nth-child(1) .price", func(e *colly.HTMLElement) {
		fmt.Println("DEBUG: Found K22 element, text:", e.Text)
		todayPrice.K22 = getPrice(e)
	})
	c.OnHTML(".gold-table tr:nth-child(2) .price", func(e *colly.HTMLElement) {
		fmt.Println("DEBUG: Found K21 element, text:", e.Text)
		todayPrice.K21 = getPrice(e)
	})
	c.OnHTML(".gold-table tr:nth-child(3) .price", func(e *colly.HTMLElement) {
		fmt.Println("DEBUG: Found K18 element, text:", e.Text)
		todayPrice.K18 = getPrice(e)
	})
	c.OnHTML(".gold-table tr:nth-child(4) .price", func(e *colly.HTMLElement) {
		fmt.Println("DEBUG: Found Traditional element, text:", e.Text)
		todayPrice.Traditional = getPrice(e)
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("DEBUG: Scraping complete. Parsed price data:", todayPrice)

		if todayPrice.K22 == 0 && todayPrice.K21 == 0 && todayPrice.K18 == 0 && todayPrice.Traditional == 0 {
			fmt.Println("WARNING: All prices are zero. Page loaded but .gold-table not found in HTML. Not writing CSV.")
			return
		}

		f, err := os.OpenFile("./fe/src/prices.csv", os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		csvReader := csv.NewReader(f)

		records, err := csvReader.ReadAll()
		if err != nil {
			panic(err)
		}
		exists := false
		for i := 0; i < len(records); i++ {
			if records[i][0] == todayPrice.Date {
				writeRow(&records[i], &todayPrice)
				exists = true
				break
			}
		}
		if !exists {
			records = append(records, make([]string, 5))
			writeRow(&records[len(records)-1], &todayPrice)
		}
		f.Seek(0, 0)
		csv.NewWriter(f).WriteAll(records)
		fmt.Println("DEBUG: CSV updated successfully")
	})

	err := c.Visit("https://www.bajus.org/gold-price")
	if err != nil {
		fmt.Println("ERROR: Visit returned error:", err)
	}
	fmt.Println("DEBUG: Scraper finished.")
}
