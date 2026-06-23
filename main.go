package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- Trade model ---

type Trade struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	UserID       int64  `gorm:"index;not null"`
	Pair         string `gorm:"not null"`
	Direction    string `gorm:"not null"`
	Entry        string `gorm:"not null"`
	SL           string `gorm:"not null"`
	TP           string `gorm:"not null"`
	Date         string `gorm:"not null"`
	Result       string `gorm:"not null"`
	Confirmation string
	Notes        string
}

// --- Database ---

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("journal.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	db.AutoMigrate(&Trade{})
	log.Println("Database ready.")
}

// --- Storage ---

func loadTrades(userID int64) ([]Trade, error) {
	var trades []Trade
	result := db.Where("user_id = ?", userID).Order("id desc").Find(&trades)
	return trades, result.Error
}

func addTrade(t Trade) error {
	return db.Create(&t).Error
}

func updateTrade(t Trade) error {
	return db.Where("id = ? AND user_id = ?", t.ID, t.UserID).Save(&t).Error
}

func deleteTrade(userID int64, id uint) error {
	return db.Where("id = ? AND user_id = ?", id, userID).Delete(&Trade{}).Error
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
	EditingID uint
}

var (
	sessions   = make(map[int64]*UserSession)
	sessionsMu sync.Mutex
)

func getSession(userID int64) *UserSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if sessions[userID] == nil {
		sessions[userID] = &UserSession{State: StateIdle}
	}
	return sessions[userID]
}

// --- Helpers ---

func tradeDetail(t Trade) string {
	return fmt.Sprintf(
		"📊 Pair: *%s*\n📈 Direction: *%s*\n💰 Entry: *%s*\n🛑 SL: *%s*\n🎯 TP: *%s*\n🕐 Date: *%s*\n🏆 Result: *%s*\n🤔 Confirmation: %s\n📝 Notes: %s",
		t.Pair, t.Direction, t.Entry, t.SL, t.TP, t.Date, t.Result, t.Confirmation, t.Notes,
	)
}

// --- Main ---

func main() {
	initDB()

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
		if sess.EditingID != 0 {
			sess.Trade.ID = sess.EditingID
			sess.Trade.UserID = c.Sender().ID
			saveErr = updateTrade(sess.Trade)
			sess.EditingID = 0
		} else {
			sess.Trade.UserID = c.Sender().ID
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
		trades, err := loadTrades(c.Sender().ID)
		if err != nil || len(trades) == 0 {
			if c.Callback() != nil {
				c.Respond()
			}
			return c.Send(
				"📒 *My Journal*\n\nNo trades yet. Add your first trade!",
				&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: mainMenu},
			)
		}
		menu := &tele.ReplyMarkup{}
		var rows []tele.Row
		limit := 10
		if len(trades) < limit {
			limit = len(trades)
		}
		for i := 0; i < limit; i++ {
			t := trades[i]
			label := fmt.Sprintf("%s | %s %s | %s", t.Date, t.Pair, t.Direction, t.Result)
			btn := menu.Data(label, "view:"+strconv.Itoa(int(t.ID)))
			rows = append(rows, menu.Row(btn))
		}
		rows = append(rows, menu.Row(menu.Data("🔙 Back", "back_main")))
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
		sess.EditingID = 0
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
			fmt.Sprintf("%s Result: *%s*\n\n🤔 *Step 8/9 — Confirmation*\n\nWhy did you take this trade?", emoji, result),
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

	// --- Dynamic callback router ---

	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		data := strings.TrimPrefix(c.Callback().Data, "\f")
		parts := strings.SplitN(data, ":", 2)
		action := parts[0]
		idStr := ""
		if len(parts) == 2 {
			idStr = parts[1]
		}

		switch action {

		case "view":
			idInt, _ := strconv.Atoi(idStr)
			var t Trade
			result := db.Where("id = ? AND user_id = ?", idInt, c.Sender().ID).First(&t)
			if result.Error != nil {
				return c.Respond(&tele.CallbackResponse{Text: "⚠️ Trade not found."})
			}
			menu := &tele.ReplyMarkup{}
			editBtn := menu.Data("✏️ Edit", "edit:"+idStr)
			deleteBtn := menu.Data("🗑 Delete", "delete:"+idStr)
			backBtn := menu.Data("🔙 Journal", "back_journal")
			menu.Inline(menu.Row(editBtn, deleteBtn), menu.Row(backBtn))
			c.Respond()
			return c.Send(
				"📋 *Trade Detail*\n\n"+tradeDetail(t),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: menu},
			)

		case "edit":
			idInt, _ := strconv.Atoi(idStr)
			var t Trade
			result := db.Where("id = ? AND user_id = ?", idInt, c.Sender().ID).First(&t)
			if result.Error != nil {
				return c.Respond(&tele.CallbackResponse{Text: "⚠️ Trade not found."})
			}
			sess := getSession(c.Sender().ID)
			sess.EditingID = t.ID
			sess.Trade = t
			sess.State = StateWaitingPair
			c.Respond()
			return c.Send(
				fmt.Sprintf("✏️ *Edit Trade*\n\n📊 *Step 1/9 — Pair*\n\nCurrent: *%s*\n\nType new value or same:", t.Pair),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			)

		case "delete":
			menu := &tele.ReplyMarkup{}
			confirmBtn := menu.Data("✅ Yes, delete", "confirm_delete:"+idStr)
			cancelBtn := menu.Data("❌ Cancel", "view:"+idStr)
			menu.Inline(menu.Row(confirmBtn, cancelBtn))
			c.Respond()
			return c.Send("🗑 Are you sure you want to delete this trade?", &tele.SendOptions{ReplyMarkup: menu})

		case "confirm_delete":
			idInt, _ := strconv.Atoi(idStr)
			if err := deleteTrade(c.Sender().ID, uint(idInt)); err != nil {
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
				"✅ Confirmation saved.\n\n📝 *Step 9/9 — Notes*\n\nAny additional notes?",
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
		c.Respond()
		trades, err := loadTrades(c.Sender().ID)
		if err != nil || len(trades) == 0 {
			return c.Send("⚠️ *My Weakness*\n\nNo trades yet. Add some trades first!", &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		}

		// --- Count trades per symbol ---
		pairCount := make(map[string]int)
		pairWins := make(map[string]int)
		buyTotal, buyWins := 0, 0
		sellTotal, sellWins := 0, 0

		for _, t := range trades {
			pairCount[t.Pair]++
			if t.Result == "Win" {
				pairWins[t.Pair]++
			}
			if t.Direction == "Buy" {
				buyTotal++
				if t.Result == "Win" {
					buyWins++
				}
			} else if t.Direction == "Sell" {
				sellTotal++
				if t.Result == "Win" {
					sellWins++
				}
			}
		}

		// --- Most traded symbol ---
		mostTradedPair := ""
		mostTradedCount := 0
		for pair, count := range pairCount {
			if count > mostTradedCount {
				mostTradedCount = count
				mostTradedPair = pair
			}
		}

		// --- Most winning symbol ---
		mostWinningPair := ""
		mostWinningCount := 0
		for pair, wins := range pairWins {
			if wins > mostWinningCount {
				mostWinningCount = wins
				mostWinningPair = pair
			}
		}

		// --- Buy vs Sell win rate ---
		buyWinRate := 0.0
		sellWinRate := 0.0
		if buyTotal > 0 {
			buyWinRate = float64(buyWins) / float64(buyTotal) * 100
		}
		if sellTotal > 0 {
			sellWinRate = float64(sellWins) / float64(sellTotal) * 100
		}

		directionLine := ""
		switch {
		case buyWinRate > sellWinRate:
			directionLine = fmt.Sprintf("📈 You perform better on *Buy* trades (%.0f%% win rate vs %.0f%% on Sells). Consider being more selective with Sell setups.", buyWinRate, sellWinRate)
		case sellWinRate > buyWinRate:
			directionLine = fmt.Sprintf("📉 You perform better on *Sell* trades (%.0f%% win rate vs %.0f%% on Buys). Consider being more selective with Buy setups.", sellWinRate, buyWinRate)
		default:
			directionLine = fmt.Sprintf("⚖️ Your Buy and Sell win rates are equal (%.0f%%). Balanced performance across both directions.", buyWinRate)
		}

		winRateLine := ""
		if mostWinningPair == "" {
			winRateLine = "You have no winning trades yet."
		} else {
			total := pairCount[mostWinningPair]
			winRateLine = fmt.Sprintf("🏆 Your best performing symbol is *%s* with %d win(s) out of %d trade(s).", mostWinningPair, mostWinningCount, total)
		}

		report := fmt.Sprintf(
			"⚠️ *My Weakness Report*\n\n"+
				"📊 You trade *%s* the most (%d trades). Ask yourself: is this your strongest setup, or a habit?\n\n"+
				"%s\n\n"+
				"%s\n\n"+
				"📝 Total trades analyzed: *%d*",
			mostTradedPair, mostTradedCount,
			winRateLine,
			directionLine,
			len(trades),
		)

		return c.Send(report, &tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: mainMenu})
	})
	bot.Handle(&btnAdvantage, func(c tele.Context) error {
		return c.Respond(&tele.CallbackResponse{Text: "✅ My Advantage — coming soon!"})
	})

	log.Println("Trading Journal Bot is running...")
	bot.Start()
}
