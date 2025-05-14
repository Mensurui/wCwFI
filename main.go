package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

func main() {
	log.Print("Main goroutine started")

	urls := []string{
		"http://info.cern.ch",
		"https://www.example.com",
		"https://www.iana.org/domains/reserved",
		"http://httpbin.org/html",
		"http://quotes.toscrape.com/",
	}

	pageCh := make(chan map[string]string, len(urls))

	var wg sync.WaitGroup
	for k, i := range urls {
		wg.Add(1)
		go FetcherGophers(i, pageCh, &wg, k)
	}

	for k := range urls {
		wg.Add(1)
		go ParserGopher(pageCh, &wg, k)
	}

	wg.Wait()

}

func FetcherGophers(url string, pageCh chan<- map[string]string, wg *sync.WaitGroup, i int) {
	defer wg.Done()
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		bodyMap := map[string]string{
			url: bodyString,
		}
		pageCh <- bodyMap
	}
}

func extractLinksAndText(n *html.Node, links *[]string, textBuilder *strings.Builder) {
	if n == nil {
		return
	}

	if n.Type == html.TextNode {
		// Trim whitespace and only add if there's actual content
		// This helps avoid lots of empty lines from formatting in HTML
		trimmedData := strings.TrimSpace(n.Data)
		if trimmedData != "" {
			textBuilder.WriteString(trimmedData)
			textBuilder.WriteString(" ") // Add a space between text elements
		}
	} else if n.Type == html.ElementNode {
		// Extract links from <a> tags
		if n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					*links = append(*links, attr.Val)
					break
				}
			}
		}
		// Skip content of script and style tags
		if n.Data == "script" || n.Data == "style" {
			// Do not recurse into children of script/style
			return
		}
	}

	// Recursively call for children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractLinksAndText(c, links, textBuilder)
	}
}

func ParserGopher(pageCh <-chan map[string]string, wg *sync.WaitGroup, i int) {
	defer wg.Done()
	body := <-pageCh
	for k, v := range body {
		doc, err := html.Parse(strings.NewReader(v))
		if err != nil {
			log.Printf("error parser: %v", err)
			return
		}

		var links []string
		var textBuilder strings.Builder
		extractLinksAndText(doc, &links, &textBuilder)
		log.Printf("\n------------------- PARSED DATA (Parser %d) -------------------\n", i)
		log.Printf("URL: %s\n", k)
		log.Printf("  Extracted Links (%d):\n", len(links))

		for _, link := range links {
			log.Printf("    - %s\n", link)
		}

		fullText := textBuilder.String()
		previewLength := 200
		if len(fullText) > previewLength {
			log.Printf("  Extracted Text (Preview - first %d chars): %s...\n", previewLength, fullText[:previewLength])
		} else {
			log.Printf("  Extracted Text: %s\n", fullText)
		}
		log.Printf("-----------------------------------------------------------\n")
	}

}
