package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

// --- Trade model ---

type Trade struct {
	ID           string `json:"id"`
	Pair         string `json:"pair"`
	Direction    string `json:"direction"`
	Entry        string `json:"entry"`
	SL           string `json:"sl"`
	TP           string `json:"tp"`
	Result       string `json:"result"`
	Confirmation string `json:"confirmation"`
	Notes        string `json:"notes"`
	Date         string `json:"date"`
}

// --- State machine ---

type TradeState int

const (
	StateIdle TradeState = iota
	StateWaitingPair
	StateWaitingDirection
	StateWaitingEntry
	StateWaitingSL
	StateWaitingTP
	StateWaitingDate
	StateWaitingResult
	StateWaitingConfirmation
	StateWaitingNotes
)

type UserSession struct {
	State     TradeState
	Trade     Trade
	EditingID string // non-empty when editing an existing trade
}

var (
	sessions   = make(map[int64]*UserSession)
	sessionsMu sync.Mutex
	tradesFile = "trades.json"
)

func getSession(userID int64) *UserSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if sessions[userID] == nil {
		sessions[userID] = &UserSession{State: StateIdle}
	}
	return sessions[userID]
}

// --- Storage ---

func loadTrades() ([]Trade, error) {
	data, err := os.ReadFile(tradesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Trade{}, nil
		}
		return nil, err
	}
	var trades []Trade
	return trades, json.Unmarshal(data, &trades)
}

func persistTrades(trades []Trade) error {
	out, _ := json.MarshalIndent(trades, "", "  ")
	return os.WriteFile(tradesFile, out, 0644)
}

func addTrade(t Trade) error {
	trades, err := loadTrades()
	if err != nil {
		return err
	}
	return persistTrades(append(trades, t))
}

func updateTrade(t Trade) error {
	trades, err := loadTrades()
	if err != nil {
		return err
	}
	for i, tr := range trades {
		if tr.ID == t.ID {
			trades[i] = t
			return persistTrades(trades)
		}
	}
	return fmt.Errorf("trade not found")
}

func deleteTrade(id string) error {
	trades, err := loadTrades()
	if err != nil {
		return err
	}
	var updated []Trade
	for _, t := range trades {
		if t.ID != id {
			updated = append(updated, t)
		}
	}
	return persistTrades(updated)
}

func genID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func tradeDetail(t Trade) string {
	return fmt.Sprintf(
		"📊 Pair: *%s*\n📈 Direction: *%s*\n💰 Entry: *%s*\n🛑 SL: *%s*\n🎯 TP: *%s*\n🏆 Result: *%s*\n🤔 Confirmation: %s\n📝 Notes: %s\n🕐 Date: %s",
		t.Pair, t.Direction, t.Entry, t.SL, t.TP, t.Result, t.Confirmation, t.Notes, t.Date,
	)
}

// --- Main ---

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	bot, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	// --- Static keyboards ---

	mainMenu := &tele.ReplyMarkup{}
	btnMyJournal := mainMenu.Data("📒 My Journal", "my_journal")
	btnAddTrade := mainMenu.Data("➕ Add a Trade", "add_trade")
	btnWeakness := mainMenu.Data("⚠️ My Weakness", "my_weakness")
	btnAdvantage := mainMenu.Data("✅ My Advantage", "my_advantage")
	mainMenu.Inline(
		mainMenu.Row(btnMyJournal, btnAddTrade),
		mainMenu.Row(btnWeakness, btnAdvantage),
	)

	dirMenu := &tele.ReplyMarkup{}
	btnBuy := dirMenu.Data("📈 Buy", "dir_buy")
	btnSell := dirMenu.Data("📉 Sell", "dir_sell")
	dirMenu.Inline(dirMenu.Row(btnBuy, btnSell))

	resultMenu := &tele.ReplyMarkup{}
	btnWin := resultMenu.Data("✅ Win", "result_win")
	btnLoss := resultMenu.Data("❌ Loss", "result_loss")
	btnBE := resultMenu.Data("➖ Breakeven", "result_be")
	resultMenu.Inline(resultMenu.Row(btnWin, btnLoss, btnBE))

	skipMenu := &tele.ReplyMarkup{}
	btnSkip := skipMenu.Data("⏭ Skip", "skip_notes")
	skipMenu.Inline(skipMenu.Row(btnSkip))

	// --- Helper: finish and save trade ---

	finishTrade := func(c tele.Context, sess *UserSession) error {
		sess.State = StateIdle
		var saveErr error
		if sess.EditingID != "" {
			sess.Trade.ID = sess.EditingID
			saveErr = updateTrade(sess.Trade)
			sess.EditingID = ""
		} else {
			sess.Trade.ID = genID()
			saveErr = addTrade(sess.Trade)
		}
		if saveErr != nil {
			return c.Send("❌ Failed to save trade. Please try again.")
		}
		return c.Send(
			"✅ *Trade Saved!*\n\n"+tradeDetail(sess.Trade),
			&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: mainMenu},
		)
	}

	// --- Helper: show journal list ---

	showJournal := func(c tele.Context) error {
		trades, err := loadTrades()
		if err != nil || len(trades) == 0 {
			msg := "📒 *My Journal*\n\nNo trades yet. Add your first trade!"
			if c.Callback() != nil {
				c.Respond()
			}
			return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: mainMenu})
		}
		menu := &tele.ReplyMarkup{}
		var rows []tele.Row
		// newest first, max 10
		for i := len(trades) - 1; i >= 0 && i >= len(trades)-10; i-- {
			t := trades[i]
			label := fmt.Sprintf("%s | %s %s | %s", t.Date, t.Pair, t.Direction, t.Result)
			btn := menu.Data(label, "view:"+t.ID)
			rows = append(rows, menu.Row(btn))
		}
		backBtn := menu.Data("🔙 Back", "back_main")
		rows = append(rows, menu.Row(backBtn))
		menu.Inline(rows...)
		if c.Callback() != nil {
			c.Respond()
		}
		return c.Send(
			"📒 *My Journal*\n\nSelect a trade to view:",
			&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: menu},
		)
	}

	// --- /start ---

	bot.Handle("/start", func(c tele.Context) error {
		return c.Send("👋 Hi "+c.Sender().FirstName+", welcome to Trading Journal Bot!", mainMenu)
	})

	// --- Add Trade: Step 1 ---

	bot.Handle(&btnAddTrade, func(c tele.Context) error {
		sess := getSession(c.Sender().ID)
		sess.State = StateWaitingPair
		sess.Trade = Trade{}
		sess.EditingID = ""
		c.Respond()
		return c.Send(
			"➕ *Add a Trade*\n\n📊 *Step 1/9 — Pair*\n\nWhich pair did you trade?\n_(e.g. XAUUSD, EURUSD, GBPUSD)_",
			&tele.SendOptions{ParseMode: tele.ModeMarkdown},
		)
	})

	// --- Direction buttons ---

	handleDir := func(c tele.Context, dir, emoji string) error {
		sess := getSession(c.Sender().ID)
		if sess.State != StateWaitingDirection {
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Please start from Add a Trade."})
		}
		sess.Trade.Direction = dir
		sess.State = StateWaitingEntry
		c.Respond()
		return c.Send(
			fmt.Sprintf("%s Direction: *%s*\n\n💰 *Step 3/9 — Entry Price*\n\nWhat was your entry price?", emoji, dir),
			&tele.SendOptions{ParseMode: tele.ModeMarkdown},
		)
	}
	bot.Handle(&btnBuy, func(c tele.Context) error { return handleDir(c, "Buy", "📈") })
	bot.Handle(&btnSell, func(c tele.Context) error { return handleDir(c, "Sell", "📉") })

	// --- Result buttons ---

	handleResult := func(c tele.Context, result, emoji string) error {
		sess := getSession(c.Sender().ID)
		if sess.State != StateWaitingResult {
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Please start from Add a Trade."})
		}
		sess.Trade.Result = result
		sess.State = StateWaitingConfirmation
		c.Respond()
		return c.Send(
			fmt.Sprintf("%s Result: *%s*\n\n🤔 *Step 7/9 — Confirmation*\n\nWhy did you take this trade?", emoji, result),
			&tele.SendOptions{ParseMode: tele.ModeMarkdown},
		)
	}
	bot.Handle(&btnWin, func(c tele.Context) error { return handleResult(c, "Win", "✅") })
	bot.Handle(&btnLoss, func(c tele.Context) error { return handleResult(c, "Loss", "❌") })
	bot.Handle(&btnBE, func(c tele.Context) error { return handleResult(c, "Breakeven", "➖") })

	// --- Skip notes ---

	bot.Handle(&btnSkip, func(c tele.Context) error {
		sess := getSession(c.Sender().ID)
		if sess.State != StateWaitingNotes {
			return c.Respond()
		}
		sess.Trade.Notes = "-"
		c.Respond()
		return finishTrade(c, sess)
	})

	// --- My Journal ---

	bot.Handle(&btnMyJournal, showJournal)

	// --- Dynamic callback router (view / edit / delete) ---
	// Handles buttons built at runtime with trade IDs embedded in callback data

	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		data := strings.TrimPrefix(c.Callback().Data, "\f")
		parts := strings.SplitN(data, ":", 2)
		action := parts[0]
		id := ""
		if len(parts) == 2 {
			id = parts[1]
		}

		switch action {

		case "view":
			trades, _ := loadTrades()
			for _, t := range trades {
				if t.ID == id {
					menu := &tele.ReplyMarkup{}
					editBtn := menu.Data("✏️ Edit", "edit:"+t.ID)
					deleteBtn := menu.Data("🗑 Delete", "delete:"+t.ID)
					backBtn := menu.Data("🔙 Journal", "back_journal")
					menu.Inline(menu.Row(editBtn, deleteBtn), menu.Row(backBtn))
					c.Respond()
					return c.Send(
						"📋 *Trade Detail*\n\n"+tradeDetail(t),
						&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: menu},
					)
				}
			}
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Trade not found."})

		case "edit":
			trades, _ := loadTrades()
			for _, t := range trades {
				if t.ID == id {
					sess := getSession(c.Sender().ID)
					sess.EditingID = t.ID
					sess.Trade = t
					sess.State = StateWaitingPair
					c.Respond()
					return c.Send(
						fmt.Sprintf("✏️ *Edit Trade*\n\n📊 *Step 1/9 — Pair*\n\nCurrent: *%s*\n\nType new value or same:", t.Pair),
						&tele.SendOptions{ParseMode: tele.ModeMarkdown},
					)
				}
			}
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Trade not found."})

		case "delete":
			menu := &tele.ReplyMarkup{}
			confirmBtn := menu.Data("✅ Yes, delete", "confirm_delete:"+id)
			cancelBtn := menu.Data("❌ Cancel", "view:"+id)
			menu.Inline(menu.Row(confirmBtn, cancelBtn))
			c.Respond()
			return c.Send("🗑 Are you sure you want to delete this trade?", &tele.SendOptions{ReplyMarkup: menu})

		case "confirm_delete":
			if err := deleteTrade(id); err != nil {
				return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to delete."})
			}
			c.Respond()
			return c.Send("✅ Trade deleted.", mainMenu)

		case "back_journal":
			return showJournal(c)

		case "back_main":
			c.Respond()
			return c.Send("📋 Main Menu", mainMenu)
		}

		return nil
	})

	// --- Text message router ---

	bot.Handle(tele.OnText, func(c tele.Context) error {
		sess := getSession(c.Sender().ID)
		text := c.Text()

		switch sess.State {
		case StateWaitingPair:
			sess.Trade.Pair = text
			sess.State = StateWaitingDirection
			return c.Send(
				fmt.Sprintf("✅ Pair: *%s*\n\n📈 *Step 2/9 — Direction*\n\nWhich direction?", text),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: dirMenu},
			)
		case StateWaitingEntry:
			sess.Trade.Entry = text
			sess.State = StateWaitingSL
			return c.Send(
				fmt.Sprintf("✅ Entry: *%s*\n\n🛑 *Step 4/9 — Stop Loss*\n\nWhat was your Stop Loss?", text),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			)
		case StateWaitingSL:
			sess.Trade.SL = text
			sess.State = StateWaitingTP
			return c.Send(
				fmt.Sprintf("✅ SL: *%s*\n\n🎯 *Step 5/9 — Take Profit*\n\nWhat was your Take Profit?", text),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			)
		case StateWaitingTP:
			sess.Trade.TP = text
			sess.State = StateWaitingDate
			return c.Send(
				fmt.Sprintf("✅ TP: *%s*\n\n📅 *Step 6/9 — Date & Time*\n\nWhen did you take this trade?\n_(e.g. 2024-06-22 14:30)_", text),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			)
		case StateWaitingDate:
			sess.Trade.Date = text
			sess.State = StateWaitingResult
			return c.Send(
				fmt.Sprintf("✅ Date: *%s*\n\n🏆 *Step 7/9 — Result*\n\nHow did the trade go?", text),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: resultMenu},
			)
		case StateWaitingConfirmation:
			sess.Trade.Confirmation = text
			sess.State = StateWaitingNotes
			return c.Send(
				"✅ Confirmation saved.\n\n📝 *Step 8/9 — Notes*\n\nAny additional notes?",
				&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: skipMenu},
			)
		case StateWaitingNotes:
			sess.Trade.Notes = text
			return finishTrade(c, sess)
		}
		return nil
	})

	// --- Placeholders ---

	bot.Handle(&btnWeakness, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ My Weakness — coming soon!"})
	})
	bot.Handle(&btnAdvantage, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "✅ My Advantage — coming soon!"})
	})

	log.Println("Trading Journal Bot is running...")
	bot.Start()
}
