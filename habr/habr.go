package habr

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func ParseHabr() []string {
	var zhdem sync.WaitGroup
	pages := make(chan int)
	results := make(chan string, 10000)
	for w := 0; w < 20; w++ {
		zhdem.Add(1)
		go func() {
			zhdem.Done()
			for page := range pages {
				url := "https://career.habr.com/vacancies?page=" + strconv.Itoa(page) + "&type=all"
				fmt.Println("Парсим страницу:", page)

				resp, err := http.Get(url)
				if err != nil {
					log.Println(err)
					continue
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Println(err)
					continue
				}
				resp.Body.Close()

				doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
				if err != nil {
					log.Println(err)
					continue
				}

				keywords := []string{
					"junior", "intern", "джуниор", "интерн", "стажер", "младший", "специалист",
					"developer", "разработчик", "grpc", "golang", "docker", "git", "linux",
					"sql", "postgresql", "алгоритмы", "ci/cd", "ооп", "базы данных", "go",
				}

				doc.Find(".vacancy-card__info").Each(func(i int, s *goquery.Selection) {
					keywordsText := strings.ToLower(s.Text())

					mustHave := strings.Contains(keywordsText, "golang") && (strings.Contains(keywordsText, "junior") || strings.Contains(keywordsText, "стажер") || strings.Contains(keywordsText, "младший"))
					count := 0
					for _, word := range keywords {
						if strings.Contains(keywordsText, word) {
							count++
						}
					}
					if mustHave && count >= 5 {
						link, _ := s.Find(".vacancy-card__title a").Attr("href")
						if link != "" {
							results <- "https://career.habr.com" + link
						}
					}
				})
			}
		}()
	}

	for i := 1; i <= 100; i++ {
		pages <- i
		//time.Sleep(1 * time.Second)
	}
	close(pages)
	zhdem.Wait()
	close(results)

	var output []string
	for r := range results {
		output = append(output, r)
	}
	return output
}
