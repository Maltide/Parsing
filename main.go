package main

import (

	// "job_parser_bot/habr"

	"fmt"
	"job_parser_bot/hh"
	"log"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//	func MemoryUsage() {
//		var m runtime.MemStats
//		runtime.ReadMemStats(&m)
//		fmt.Printf("Current alloc is %v\n", m.Alloc/1024/1024)
//	}
type SessionData struct {
	Vacancies []string
	Offset    int
}

var (
	stroguy_poisk     = make(map[int64]bool)
	userState         = map[int64]string{} //строка состояния(только одна единовременно)
	keywords_main     = map[int64][]string{}
	keywords_add      = map[int64][]string{}
	keywords_no       = map[int64][]string{}
	format_raboty     = map[int64][]string{}
	hours_raboty      = map[int64][]string{}
	graphic_raboty    = map[int64][]string{}
	zarplata          = map[int64][]string{}
	gorod             = map[int64][]string{}
	splitter          = regexp.MustCompile(`[\s,;\n]+`)
	splitter_for_main = regexp.MustCompile(`[,;]+`)
	shownVacancies    = make(map[string]time.Time)
	sessions          = make(map[int64]*SessionData)
)

func main() {
	// go func() {
	// 	for {
	// 		MemoryUsage()
	// 		time.Sleep(1 * time.Second)
	// 	}
	// }()

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			for link, t := range shownVacancies {
				if time.Since(t) > 8*24*time.Hour {
					delete(shownVacancies, link)
				}
			}
		}
	}()

	bot, err := tgbotapi.NewBotAPI("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	if err != nil {
		log.Println(err)
	}

	get_from_bot := tgbotapi.NewUpdate(0)
	get_from_bot.Timeout = 60

	mainkeywords_button := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Ввести заново"), tgbotapi.NewKeyboardButton("Дальше")))
	start_button := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Начать поиск вакансий"), tgbotapi.NewKeyboardButton("Очистить память")))
	skip_button := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Ввести заново"),
		tgbotapi.NewKeyboardButton("Дальше")))
	navigation_button := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Начать поиск вакансий"), tgbotapi.NewKeyboardButton("Показать еще")))
	filters_button := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("График работы"), tgbotapi.NewKeyboardButton("Формат"), tgbotapi.NewKeyboardButton("Регион"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Рабочие часы"), tgbotapi.NewKeyboardButton("Зарплата"), tgbotapi.NewKeyboardButton("Начать поиск")))

	updates := bot.GetUpdatesChan(get_from_bot)

	for update := range updates {
		if _, ok := stroguy_poisk[update.Message.Chat.ID]; !ok {
			text := strings.ToLower(update.Message.Text)
			if text == "да" {
				stroguy_poisk[update.Message.Chat.ID] = true
			} else if text == "нет" {
				stroguy_poisk[update.Message.Chat.ID] = false
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Включить строгий поиск по названию Вашей специальности? (да/нет)")
				bot.Send(msg)
				continue
			}
		}
		if update.Message != nil {
			if _, ok := userState[update.Message.Chat.ID]; !ok { //проверка на наличие чата с пользователем(был или нет. Если нет, то отсылается кнопка)
				startyem := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите опцию в меню чата\n")
				startyem.ReplyMarkup = start_button
				bot.Send(startyem)
				userState[update.Message.Chat.ID] = ""
				continue
			}
			switch update.Message.Text {

			case "Начать поиск вакансий":
				userState[update.Message.Chat.ID] = "main_words"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите ключевые слова - именно те, которые обозначают основную профессию.")
				msg.ReplyMarkup = mainkeywords_button
				bot.Send(msg)

			case "Показать еще":
				session, ok := sessions[update.Message.Chat.ID]
				if !ok || session.Offset >= len(session.Vacancies) {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Новых вакансий нет.")
					msg.ReplyMarkup = navigation_button
					bot.Send(msg)
					break
				}

				end := session.Offset + 10 // не вылезаем за пределы когда вакансии заканчиваются.
				if end > len(session.Vacancies) {
					end = len(session.Vacancies)
				}

				for _, link := range session.Vacancies[session.Offset:end] {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, link)
					bot.Send(msg)
				}

				session.Offset = end

			case "График работы":
				buttons := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("5/2"), tgbotapi.NewKeyboardButton("6/1"), tgbotapi.NewKeyboardButton("2/2"),
						tgbotapi.NewKeyboardButton("3/3"), tgbotapi.NewKeyboardButton("4/2"), tgbotapi.NewKeyboardButton("4/3"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("3/2"), tgbotapi.NewKeyboardButton("1/3"), tgbotapi.NewKeyboardButton("2/1"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Свободный"), tgbotapi.NewKeyboardButton("По выходным"), tgbotapi.NewKeyboardButton("Другое"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Вернуться к фильтрам"), tgbotapi.NewKeyboardButton("Ввести заново")))

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите нужные Вам графики работы")
				msg.ReplyMarkup = buttons
				bot.Send(msg)
				userState[update.Message.Chat.ID] = "filter_graphic"
				continue

			case "Формат":
				buttons := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("На месте работодателя"), tgbotapi.NewKeyboardButton("Удалённо"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Гибрид"), tgbotapi.NewKeyboardButton("Разъездной"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Вернуться к фильтрам"), tgbotapi.NewKeyboardButton("Ввести заново")))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите нужный Вам формат работы")
				msg.ReplyMarkup = buttons
				bot.Send(msg)
				userState[update.Message.Chat.ID] = "filter_format"
				continue

			case "Регион":
				buttons := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Вернуться к фильтрам"), tgbotapi.NewKeyboardButton("Ввести заново")))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Напишите город")
				msg.ReplyMarkup = buttons
				bot.Send(msg)
				userState[update.Message.Chat.ID] = "filter_gorod"
				continue

			case "Рабочие часы":
				buttons := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("2 часа"), tgbotapi.NewKeyboardButton("3 часа"),
						tgbotapi.NewKeyboardButton("4 часа"), tgbotapi.NewKeyboardButton("5 часов"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("6 часов"), tgbotapi.NewKeyboardButton("7 часов"),
						tgbotapi.NewKeyboardButton("8 часов"), tgbotapi.NewKeyboardButton("9 часов"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("10 часов"), tgbotapi.NewKeyboardButton("11 часов"),
						tgbotapi.NewKeyboardButton("12 часов"), tgbotapi.NewKeyboardButton("24 часа"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("По договоренности"), tgbotapi.NewKeyboardButton("Другое"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Вернуться к фильтрам"), tgbotapi.NewKeyboardButton("Ввести заново")))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите рабочие часы")
				msg.ReplyMarkup = buttons
				bot.Send(msg)
				userState[update.Message.Chat.ID] = "filter_hours"
				continue

			case "Зарплата":
				buttons := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("50000 ₽"), tgbotapi.NewKeyboardButton("80000 ₽"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("100000 ₽"), tgbotapi.NewKeyboardButton("150000 ₽"),
						tgbotapi.NewKeyboardButton("200000 ₽"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("300000 ₽"), tgbotapi.NewKeyboardButton("400000 ₽"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Вернуться к фильтрам"), tgbotapi.NewKeyboardButton("Ввести заново")))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите зарплату")
				msg.ReplyMarkup = buttons
				bot.Send(msg)
				userState[update.Message.Chat.ID] = "filter_zarplata"
				continue

			case "Начать поиск":
				mainWords := keywords_main[update.Message.Chat.ID]
				addWords := keywords_add[update.Message.Chat.ID]
				noWords := keywords_no[update.Message.Chat.ID]
				format := format_raboty[update.Message.Chat.ID]
				hours := hours_raboty[update.Message.Chat.ID]
				graphic := graphic_raboty[update.Message.Chat.ID]
				zp := zarplata[update.Message.Chat.ID]
				region := gorod[update.Message.Chat.ID]
				links := hh.ParseHH(&mainWords, &addWords, &noWords, &format, &hours, &graphic, &zp, &region, stroguy_poisk[update.Message.Chat.ID])
				var filtered []string
				for _, link := range links {
					if i, ok := shownVacancies[link]; ok && time.Since(i) < 7*24*time.Hour {
						continue
					}
					shownVacancies[link] = time.Now()
					filtered = append(filtered, link)
				}

				// создаём сессию
				sessions[update.Message.Chat.ID] = &SessionData{
					Vacancies: filtered,
					Offset:    0,
				}

				// выдаём первые 10
				end := 10
				if len(filtered) < 10 {
					end = len(filtered)
				}
				for _, link := range filtered[:end] {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, link)
					bot.Send(msg)
				}
				sessions[update.Message.Chat.ID].Offset = end

				// кнопка "Показать еще"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Всего обработано страниц: %d", hh.Count_pages))
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Показать еще"),
						tgbotapi.NewKeyboardButton("Начать поиск вакансий"),
					),
				)
				bot.Send(msg)

			default:
				state := userState[update.Message.Chat.ID]

				switch state {

				case "main_words":
					text := update.Message.Text
					if text == "Ввести заново" {
						keywords_main[update.Message.Chat.ID] = []string{}
						break
					}
					if text == "Дальше" {
						userState[update.Message.Chat.ID] = "add_words" //меняем флаг для следующего этапа поиска(т.е. доп слова)
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Принято, теперь введите вторичные ключевые слова(навыки, технологии - все что обычно пишут в описании и т.д.)")
						msg.ReplyMarkup = skip_button
						bot.Send(msg)
						break
					}
					keywords_main[update.Message.Chat.ID] = append(keywords_main[update.Message.Chat.ID], splitter_for_main.Split(text, -1)...)
					fmt.Printf("%#v\n", keywords_main[update.Message.Chat.ID])

				case "add_words":
					text := update.Message.Text
					if text == "Ввести заново" {
						keywords_add[update.Message.Chat.ID] = []string{}
						break
					}
					if text == "Дальше" {
						userState[update.Message.Chat.ID] = "no_words" // а это уже для исключения из поиска слов, которые пойдут в следующем if в массив исключающих слов
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Готово. Если Вы не хотите видеть в вакансиях какие-то ключевые слова, то напишите их здесь")
						msg.ReplyMarkup = skip_button
						bot.Send(msg)
						break
					}
					keywords_add[update.Message.Chat.ID] = append(keywords_add[update.Message.Chat.ID], splitter.Split(text, -1)...)
					fmt.Println(keywords_add)

				case "no_words":
					text := update.Message.Text
					if text == "Ввести заново" {
						keywords_no[update.Message.Chat.ID] = []string{}
						break
					}
					if text == "Дальше" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Перед началом поиска Вы можете выбрать фильтры для уточнения запроса.\n Если не нужны, выбирайте кнопку <Начать поиск>")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
					}
					keywords_no[update.Message.Chat.ID] = append(keywords_no[update.Message.Chat.ID], splitter.Split(text, -1)...)

					fmt.Println(keywords_no)

					log.Printf("Получено сообщение: '%s'", update.Message.Text)
					log.Printf("state: '%s', text: '%s'", state, update.Message.Text)

				case "filters":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.ReplyMarkup = filters_button
					bot.Send(msg)

				case "filter_graphic":
					text := update.Message.Text
					if text == "Вернуться к фильтрам" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите фильтр или начните поиск")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
						log.Printf("User %d state: '%s'", update.Message.Chat.ID, userState[update.Message.Chat.ID])
						continue
					} else if text == "Ввести заново" {
						graphic_raboty[update.Message.Chat.ID] = []string{}
					} else {
						graphic_raboty[update.Message.Chat.ID] = append(
							graphic_raboty[update.Message.Chat.ID],
							text)
					}
					fmt.Println(graphic_raboty)

				case "filter_format":
					text := update.Message.Text
					if text == "Вернуться к фильтрам" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите фильтр или начните поиск")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
						continue
					} else if text == "Ввести заново" {
						format_raboty[update.Message.Chat.ID] = []string{}
					} else if text == "Удалённо" {
						format_raboty[update.Message.Chat.ID] = append(format_raboty[update.Message.Chat.ID], []string{"Удалённая работа", "Удалённо"}...)
					} else {
						format_raboty[update.Message.Chat.ID] = append(
							format_raboty[update.Message.Chat.ID],
							text)
					}
					fmt.Println(format_raboty)
					fmt.Printf("%#v\n", format_raboty[update.Message.Chat.ID])

				case "filter_hours":
					text := update.Message.Text
					if text == "Вернуться к фильтрам" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите фильтр или начните поиск")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
						continue
					} else if text == "Ввести заново" {
						hours_raboty[update.Message.Chat.ID] = []string{}
					} else {
						hours_raboty[update.Message.Chat.ID] = append(
							hours_raboty[update.Message.Chat.ID],
							splitter.Split(text, -1)...)
					}
					fmt.Println(hours_raboty)

				case "filter_zarplata":
					text := update.Message.Text
					if text == "Вернуться к фильтрам" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите фильтр или начните поиск")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
						continue
					} else if text == "Ввести заново" {
						zarplata[update.Message.Chat.ID] = []string{}
					} else {
						zarplata[update.Message.Chat.ID] = append(
							zarplata[update.Message.Chat.ID],
							splitter.Split(text, -1)...)
					}
					fmt.Println(zarplata)

				case "filter_gorod":
					text := update.Message.Text
					if text == "Вернуться к фильтрам" {
						userState[update.Message.Chat.ID] = "filters"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите фильтр или начните поиск")
						msg.ReplyMarkup = filters_button
						bot.Send(msg)
						continue
					} else if text == "Ввести заново" {
						gorod[update.Message.Chat.ID] = []string{}
					} else {
						gorod[update.Message.Chat.ID] = append(
							gorod[update.Message.Chat.ID],
							splitter.Split(text, -1)...)
					}
					fmt.Println(gorod)
				}
			}
		}
	}
}
