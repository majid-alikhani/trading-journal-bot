package main

import (
	"log"
	"os"
	"time"

	tele "gopkg.in/telebot.v3"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	// Main menu keyboard
	mainMenu := &tele.ReplyMarkup{}
	btnMyJournal := mainMenu.Data("📒 My Journal", "my_journal")
	btnAddTrade := mainMenu.Data("➕ Add a Trade", "add_trade")
	btnWeakness := mainMenu.Data("⚠️ My Weakness", "my_weakness")
	btnAdvantage := mainMenu.Data("✅ My Advantage", "my_advantage")

	mainMenu.Inline(
		mainMenu.Row(btnMyJournal, btnAddTrade),
		mainMenu.Row(btnWeakness, btnAdvantage),
	)

	// /start handler
	bot.Handle("/start", func(c tele.Context) error {
		name := c.Sender().FirstName
		return c.Send(
			"👋 Hi "+name+", welcome to Trading Journal Bot!\nWhen you write your problem, half of it gonna solves\n ",
			mainMenu,
		)
	})

	// Button handlers
	bot.Handle(&btnMyJournal, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "My Journal coming soon..."})
	})

	bot.Handle(&btnAddTrade, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "Add a Trade coming soon..."})
	})

	bot.Handle(&btnWeakness, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "My Weakness coming soon..."})
	})

	bot.Handle(&btnAdvantage, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "My Advantage coming soon..."})
	})

	log.Println("Trading Journal Bot is running...")
	bot.Start()
}
