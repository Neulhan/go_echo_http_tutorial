package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type extractedJob struct {
	id       string
	title    string
	location string
	summary  string
	salary   string
}

//Scrape 검색어에 대한 채용 정보를 스크래핑 해오는 함수
func Scrape(term string) {
	c := make(chan []extractedJob)
	var indeedURL string = "https://kr.indeed.com/jobs?q=" + term + "&limit=50"

	pages := getPages(indeedURL)
	var jobs []extractedJob
	for i := 0; i < pages; i++ {
		go getPage(i, indeedURL, c)
	}

	for i := 0; i < pages; i++ {
		extractedJobs := <-c
		jobs = append(jobs, extractedJobs...)
	}

	writeJobs(jobs)
	fmt.Println("Done, extracted", len(jobs))
}

func getPage(page int, indeedURL string, cm chan []extractedJob) {
	c := make(chan extractedJob)
	var jobs []extractedJob
	start := page * 50
	pageURL := indeedURL + "&start=" + strconv.Itoa(start)
	fmt.Println("Requesting", pageURL)

	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	defer res.Body.Close()
	checkErr(err)
	searchCards := doc.Find(".jobsearch-SerpJobCard")

	searchCards.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c)
	})

	for i := 0; i < searchCards.Length(); i++ {
		jobs = append(jobs, <-c)
	}
	cm <- jobs
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs1.csv")
	checkErr(err)
	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"id", "title", "location", "summary", "salary"}

	wErr := writer.Write(headers)

	for _, job := range jobs {
		jobSlice := []string{job.id, job.title, job.location, job.summary, job.salary}
		jwErr := writer.Write(jobSlice)
		checkErr(jwErr)
	}

	checkErr(wErr)
}
func extractJob(card *goquery.Selection, c chan<- extractedJob) {
	id, _ := card.Attr("data-jk")
	title := CleanString(card.Find(".title > a").Text())
	location := CleanString(card.Find("span.location").Text())
	summary := CleanString(card.Find(".summary").Text())
	salary := CleanString(card.Find("span.salaryText").Text())

	c <- extractedJob{
		id:       id,
		title:    title,
		location: location,
		summary:  summary,
		salary:   salary,
	}
}

//CleanString string을 clean
func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPages(indeedURL string) (pages int) {
	res, err := http.Get(indeedURL)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})

	return
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request Failed With Status: ", res.StatusCode)
	}
}
