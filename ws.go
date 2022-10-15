package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type Book struct {
	price  float64
	title  string
	stock  bool
	rating int
}

func (b Book) String() string {
	return fmt.Sprintf("Title: %v\nPrice:%.2f\nIs in stock:%v\nRating:%v stars", b.title, b.price, b.stock, b.rating)
}

var wg sync.WaitGroup

func main() {
	base_url := "http://books.toscrape.com/"
	res, err := http.Get(base_url)

	if err != nil {
		fmt.Println("Error on request to", base_url)
		return
	}
	defer res.Body.Close()
	body := res.Body
	doc := html.NewTokenizer(body)

	books := make([]Book, 0)

	scrap(doc, &books)

	for _, book := range books {
		fmt.Println(book.title)
	}
}
func hasClassName(Attributes []html.Attribute, attr_val string) bool {
	for _, attr := range Attributes {
		if attr.Key == "class" && strings.Contains(attr.Val, attr_val) {
			return true
		}
	}
	return false
}

func hasAttribute(Attributes []html.Attribute, attr_name string) bool {
	for _, attr := range Attributes {
		if attr.Key == attr_name {
			return true
		}
	}
	return false
}

func scrap(doc *html.Tokenizer, books *[]Book) {
	urls := make([]string, 0)

loop:
	for {
		token := doc.Next()

		switch token {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			tag := doc.Token()
			if tag.Data == "a" && hasAttribute(tag.Attr, "title") {
				for _, attr := range tag.Attr {
					if attr.Key == "href" {
						urls = append(urls, attr.Val)
					}
				}
			}
		}
	}
	r := make(chan Book, len(urls))

	for _, url := range urls {
		go getData(url, r)
	}
	wg.Wait()
	close(r)
	fmt.Println(len(r))
	for value := range r {
		*books = append(*books, value)
	}
}

func getData(url string, rchan chan Book) {
	wg.Add(1)
	defer wg.Done()

	initBook := Book{
		price:  0.0,
		title:  "",
		stock:  false,
		rating: -1,
	}
	response, err := http.Get("http://books.toscrape.com/" + url)
	if err != nil {
		panic("Error on reqeust to " + url)
	}
	defer response.Body.Close()

	doc := html.NewTokenizer(response.Body)

loop:
	for {
		token := doc.Next()
		switch token {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			tag := doc.Token()
			if tag.Data == "h1" {
				doc.Next()
				initBook.title = doc.Token().Data
			} else if tag.Data == "p" && hasClassName(tag.Attr, "price_color") {
				doc.Next()
				val, err := strconv.ParseFloat(doc.Token().Data[2:], 32)
				if err != nil {
					panic("Error on parse")
				}
				initBook.price = val
			} else if tag.Data == "p" && hasClassName(tag.Attr, "instock availability") {
				initBook.stock = true
			} else if tag.Data == "p" && hasClassName(tag.Attr, "star-rating") {
				for _, attr := range tag.Attr {
					if attr.Key == "class" {
						rating_text := strings.Fields(attr.Val)[1]
						var rating_val int
						switch rating_text {
						case "One":
							rating_val = 1
						case "Two":
							rating_val = 2
						case "Three":
							rating_val = 3
						case "Four":
							rating_val = 4
						case "Five":
							rating_val = 5
						}
						initBook.rating = rating_val
					}
				}
			}
		}
	}
	rchan <- initBook
}
