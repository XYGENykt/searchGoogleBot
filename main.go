package main

import (
	"context"
	"strconv"

	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type Result struct {
	Position int64
	Result   *customsearch.Result
}

func main() {

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	fmt.Println("Server running")

	BOT_TOKEN := os.Getenv("BOT_TOKEN")
	CHAT_ID, _ := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	SEARCH_ENGINE_ID := os.Getenv("SEARCH_ENGINE_ID")
	fmt.Println(CHAT_ID)

	if BOT_TOKEN == "" {
		log.Fatal("BOT_TOKEN не установлен")
	}

	bot, err := tgbotapi.NewBotAPI(BOT_TOKEN)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msgToMyChannel := tgbotapi.NewMessage(CHAT_ID, update.Message.Text+" автор сообщения("+update.Message.From.UserName+")")
			bot.Send(msgToMyChannel)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			bot.Send(msg)
		}
	}

	// Load search key
	data, err := os.ReadFile("search-key.json")
	if err != nil {
		log.Fatal(err)
	}

	// Get the config from the json key file with the correct scope
	conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/cse")
	if err != nil {
		log.Fatal(err)
	}

	// Create context and client
	ctx := context.Background()
	client := conf.Client(ctx)

	// Create custom search service with the authenticated client
	cseService, err := customsearch.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatal(err)
	}

	search := cseService.Cse.List().Q("labis").Cx(SEARCH_ENGINE_ID)
	result := doSearch(search)

	if result.Position == 0 {
		log.Fatal("No results found in the top 10 pages.\n")
	}

	fmt.Printf("Result found for \"%s\"!\n", "google.com")
	fmt.Printf("Position: %d\n", result.Position)
	fmt.Printf("Url: %s\n", result.Result.Link)
	fmt.Printf("Title: %s\n", result.Result.Title)
	fmt.Printf("Snippet: %s\n", result.Result.Snippet)
}

func doSearch(search *customsearch.CseListCall) (result Result) {
	start := int64(1)

	// CSE Limits you to 10 pages of results with max 10 results per page
	for start < 100 {
		call, err := search.Start(start).Do()
		if err != nil {
			log.Fatal(err)
		}

		position, csResult := findDomain(call.Items, start)
		if csResult != nil {
			result = Result{
				Position: position,
				Result:   csResult,
			}
			return
		}

		// Проверяем TotalResults (который является строкой)
		totalResults, err := strconv.ParseInt(call.SearchInformation.TotalResults, 10, 64)
		if err != nil {
			log.Printf("Error parsing total results: %v", err)
			return
		}

		// No more search results?
		if totalResults < start {
			return
		}
		start += 10
	}
	return
}

func findDomain(results []*customsearch.Result, start int64) (position int64, result *customsearch.Result) {
	for index, r := range results {
		if strings.Contains(r.Link, "google.com") {
			return int64(index) + start, r
		}
	}
	return 0, nil
}
