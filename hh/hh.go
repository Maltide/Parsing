package hh

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var Count_pages int64

func parseZp(zp *[]string) int {
	if len(*zp) == 0 {
		return 0
	}
	cleaned := strings.ReplaceAll((*zp)[0], " ", "")
	cleaned = strings.ReplaceAll(cleaned, "₽", "")
	value, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0
	}
	return value
}

func ContainsAny(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, word := range keywords {
		if strings.Contains(text, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

type hh_Response struct {
	Pages int `json:"pages"`
	Items []struct {
		Name          string `json:"name"`
		Alternate_url string `json:"alternate_url"`
		Salary        struct {
			From     int    `json:"from"`
			To       int    `json:"to"`
			Currency string `json:"currency"`
		} `json:"salary"`
		Area struct {
			Name string `json:"name"`
		} `json:"area"`
		Hours []struct {
			Name string `json:"name"`
		} `json:"working_hours"`
		Graphic []struct {
			Name string `json:"name"`
		} `json:"work_schedule_by_days"`
		WorkFormat []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"work_format"`
		Snippet struct {
			Requirment     string `json:"requirement"`
			Responsibility string `json:"responsibility"`
		} `json:"snippet"`
	} `json:"items"`
}

func ParseHH(mainWords, addWords, noWords, format, hours, graphic, zp, region *[]string, stroguy_poisk bool) []string {
	hh_keywords_main := url.QueryEscape(strings.Join(*mainWords, " "))

	var zhdem sync.WaitGroup
	results := make(chan string, 10000)

	url := "https://api.hh.ru/vacancies?text=" + hh_keywords_main + "&page=0&per_page=100&currency=RUR"
	if len(*zp) > 0 && (*zp)[0] != "" {
		url += "&salary" + (*zp)[0]
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var AllPages hh_Response
	err = json.Unmarshal(body, &AllPages)
	if err != nil {
		log.Println(err)
		return nil
	}

	gorutines_channel := make(chan struct{}, 5)

	for i := 0; i < AllPages.Pages; i++ {
		zhdem.Add(1)

		go func(page int) {
			gorutines_channel <- struct{}{}
			defer func() {

				<-gorutines_channel
				zhdem.Done()
				fmt.Println("Закончили парсить страницу:", page)
				atomic.AddInt64(&Count_pages, 1)
			}()

			var result hh_Response

			url := "https://api.hh.ru/vacancies?text=" + hh_keywords_main + "&page=" + strconv.Itoa(page) + "&per_page=100"
			if len(*zp) > 0 {
				url += "&salary=" + (*zp)[0]
			}

			time.Sleep(time.Second * 2) // пауза 2 секунды между запросами к HH

			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Ошибка http.Get на странице %d: %v\n", page, err)
				return
			}
			defer func() {
				if cerr := resp.Body.Close(); cerr != nil {
					log.Printf("Ошибка закрытия body на странице %d: %v\n", page, cerr)
				}
			}()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Ошибка чтения тела на странице %d: %v\n", page, err)
				return
			}

			err = json.Unmarshal(body, &result)
			if err != nil {
				log.Printf("Ошибка json.Unmarshal на странице %d: %v\n", page, err)
				return
			}

			for _, vacancy := range result.Items {
				title_main := strings.ToLower(vacancy.Name)
				title_add := strings.ToLower(vacancy.Snippet.Requirment + " " + vacancy.Snippet.Responsibility)

				title_region := strings.ToLower(vacancy.Area.Name)
				if vacancy.Salary.Currency != "RUR" {
					continue
				}

				if parseZp(zp) > 0 {
					if (vacancy.Salary.From == 0 && vacancy.Salary.To < parseZp(zp)) ||
						(vacancy.Salary.From > 0 && vacancy.Salary.To > 0 && vacancy.Salary.To < parseZp(zp)) ||
						(vacancy.Salary.From == 0 && vacancy.Salary.To == 0) ||
						(vacancy.Salary.To == 0 && vacancy.Salary.From < parseZp(zp)-50000) {
						fmt.Printf("ОТБРОСИЛ: %s | From: %d | To: %d | minSalary: %d\n", vacancy.Name, vacancy.Salary.From, vacancy.Salary.To, parseZp(zp))
						continue
					}
				}

				fmt.Println(mainWords)
				if stroguy_poisk {
					matched := false
					for _, word := range *mainWords {
						if strings.EqualFold(strings.ToLower(strings.TrimSpace(title_main)), strings.ToLower(strings.TrimSpace(word))) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
				}

				if len(*addWords) > 0 && !((ContainsAny(title_add, *addWords)) && (!ContainsAny(title_main+title_add, *noWords)) &&
					(len(*region) == 0 || (title_region != "" && ContainsAny(title_region, *region)))) {
					continue
				}

				found_format := false
				if len(*format) > 0 {
					for _, wf := range vacancy.WorkFormat {
						if ContainsAny(wf.Name, *format) {
							found_format = true
							break
						}
					}
					if !found_format {
						continue
					}
				}

				found_hours := false
				if len(*hours) > 0 {
					for _, hf := range vacancy.Hours {
						if ContainsAny(hf.Name, *hours) {
							found_hours = true
							break
						}
					}
					if !found_hours {
						continue
					}
				}

				fmt.Printf("ДОБАВИЛ: %s | From: %d | To: %d | minSalary: %d\n", vacancy.Name, vacancy.Salary.From, vacancy.Salary.To, parseZp(zp))
				results <- vacancy.Alternate_url

			}
		}(i)

	}
	go func() {
		zhdem.Wait()
		close(results)
	}()

	unique := make(map[string]struct{})
	var alllinks []string
	for current_link := range results {
		if _, exists := unique[current_link]; exists {
			fmt.Println("Дубликат: \n", current_link)
			continue
		}
		unique[current_link] = struct{}{}
		alllinks = append(alllinks, current_link)
	}
	fmt.Println("MainWords:", *mainWords)

	return alllinks
}
