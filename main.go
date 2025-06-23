package main

import (
	"fmt"
	"job_parser_bot/habr"
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
	habr_vacancy := habr.ParseHabr()
	fmt.Println("Вакансии на хабре: ")
	for _, v := range habr_vacancy {
		fmt.Println(v)
	}
}
