package main

//LUIZKA!!!
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

// type Vacancy struct {
// 	Link          string
// 	Title         string
// 	Date          string
// 	Salary        int
// 	Skills        []string
// 	City_schedule []string
// }

func main() {
	var zhdem sync.WaitGroup
	pages := make(chan int)
	results := make(chan string, 100)
	zhdem.Add(20)
	go func() {
		defer zhdem.Done()
		for i := 1; ; i++ {
			url := "https://career.habr.com/vacancies?page=" + strconv.Itoa(i)
			resp, err := http.Get(url)
			if err != nil {
				log.Println("Ошибка при получении страницы:", i)
				break
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
			if err != nil {
				log.Println("Ошибка парсинга страницы:", i)
				break
			}

			if doc.Find(".vacancy-card__info").Length() == 0 {
				fmt.Println("Страницы закончились на:", i)
				break
			}

			pages <- i
		}
		close(pages)
	}()

	for w := 0; w < 20; w++ {
		defer zhdem.Add(20)
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
	zhdem.Wait()
	close(results)
	for r := range results {
		fmt.Println(r)
	}
}
