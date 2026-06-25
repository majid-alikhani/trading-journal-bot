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

func analyzeWeakness(trades []Trade) string {
	pairCount := make(map[string]int)
	pairLosses := make(map[string]int)
	buyTotal, buyWins, buyLosses := 0, 0, 0
	sellTotal, sellWins, sellLosses := 0, 0, 0

	for _, t := range trades {
		pairCount[t.Pair]++
		if t.Result == "Loss" {
			pairLosses[t.Pair]++
		}
		if t.Direction == "Buy" {
			buyTotal++
			if t.Result == "Win" {
				buyWins++
			}
			if t.Result == "Loss" {
				buyLosses++
			}
		} else if t.Direction == "Sell" {
			sellTotal++
			if t.Result == "Win" {
				sellWins++
			}
			if t.Result == "Loss" {
				sellLosses++
			}
		}
	}

	// Least traded pair
	leastPair, leastCount := "", int(^uint(0)>>1)
	for pair, count := range pairCount {
		if count < leastCount {
			leastCount = count
			leastPair = pair
		}
	}

	// Most losing pair
	worstPair, worstLosses := "", 0
	for pair, losses := range pairLosses {
		if losses > worstLosses {
			worstLosses = losses
			worstPair = pair
		}
	}

	// Loss rates
	buyLossRate, sellLossRate := 0.0, 0.0
	if buyTotal > 0 {
		buyLossRate = float64(buyLosses) / float64(buyTotal) * 100
	}
	if sellTotal > 0 {
		sellLossRate = float64(sellLosses) / float64(sellTotal) * 100
	}

	var sb strings.Builder
	sb.WriteString("⚠️ MY WEAKNESS REPORT\n\n")
	sb.WriteString(fmt.Sprintf("📊 Total trades analyzed: %d\n\n", len(trades)))

	// Least traded
	sb.WriteString(fmt.Sprintf(
		"📉 You trade %s the least (%d trade(s)). This could mean you lack confidence in this pair or simply avoid it — worth reviewing if it has potential in your strategy.\n\n",
		leastPair, leastCount,
	))

	// Worst pair
	if worstPair == "" {
		sb.WriteString("🎉 You have no losing trades yet — keep it up!\n\n")
	} else {
		total := pairCount[worstPair]
		lossRate := float64(worstLosses) / float64(total) * 100
		sb.WriteString(fmt.Sprintf(
			"❌ Your worst pair is %s with %d loss(es) out of %d trade(s) (%.0f%% loss rate). Consider pausing this pair and reviewing what goes wrong in your setups.\n\n",
			worstPair, worstLosses, total, lossRate,
		))
	}

	// Direction weakness
	switch {
	case buyLossRate > sellLossRate:
		sb.WriteString(fmt.Sprintf(
			"📈 Your Buy trades fail more often (%.0f%% loss rate vs %.0f%% on Sells). You may be entering longs without enough confirmation or against the trend. Be more patient with Buy setups.\n",
			buyLossRate, sellLossRate,
		))
	case sellLossRate > buyLossRate:
		sb.WriteString(fmt.Sprintf(
			"📉 Your Sell trades fail more often (%.0f%% loss rate vs %.0f%% on Buys). You may be shorting into strong bullish momentum. Review your Sell confirmations more carefully.\n",
			sellLossRate, buyLossRate,
		))
	default:
		sb.WriteString(fmt.Sprintf(
			"⚖️ Your Buy and Sell loss rates are equal (%.0f%%). Losses are evenly spread — focus on improving your overall entry confirmation.\n",
			buyLossRate,
		))
	}

	return sb.String()
}

func analyzeAdvantage(trades []Trade) string {
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

	// Most traded pair
	mostPair, mostCount := "", 0
	for pair, count := range pairCount {
		if count > mostCount {
			mostCount = count
			mostPair = pair
		}
	}

	// Best winning pair
	bestPair, bestWins := "", 0
	for pair, wins := range pairWins {
		if wins > bestWins {
			bestWins = wins
			bestPair = pair
		}
	}

	// Win rates
	buyWinRate, sellWinRate := 0.0, 0.0
	if buyTotal > 0 {
		buyWinRate = float64(buyWins) / float64(buyTotal) * 100
	}
	if sellTotal > 0 {
		sellWinRate = float64(sellWins) / float64(sellTotal) * 100
	}

	var sb strings.Builder
	sb.WriteString("✅ MY ADVANTAGE REPORT\n\n")
	sb.WriteString(fmt.Sprintf("📊 Total trades analyzed: %d\n\n", len(trades)))

	// Most traded
	sb.WriteString(fmt.Sprintf(
		"📊 You trade %s the most (%d trade(s)). This is your most familiar pair — make sure you are exploiting it to its full potential.\n\n",
		mostPair, mostCount,
	))

	// Best pair
	if bestPair == "" {
		sb.WriteString("📈 No winning trades yet — keep logging and the patterns will emerge.\n\n")
	} else {
		total := pairCount[bestPair]
		winRate := float64(bestWins) / float64(total) * 100
		sb.WriteString(fmt.Sprintf(
			"🏆 Your best pair is %s with %d win(s) out of %d trade(s) (%.0f%% win rate). This is where your edge is strongest — prioritize this pair in your sessions.\n\n",
			bestPair, bestWins, total, winRate,
		))
	}

	// Direction strength
	switch {
	case buyWinRate > sellWinRate:
		sb.WriteString(fmt.Sprintf(
			"📈 You are stronger on Buy trades (%.0f%% win rate vs %.0f%% on Sells). Lean into long setups and be more selective with Sells.\n",
			buyWinRate, sellWinRate,
		))
	case sellWinRate > buyWinRate:
		sb.WriteString(fmt.Sprintf(
			"📉 You are stronger on Sell trades (%.0f%% win rate vs %.0f%% on Buys). Lean into short setups and be more selective with Buys.\n",
			sellWinRate, buyWinRate,
		))
	default:
		sb.WriteString(fmt.Sprintf(
			"⚖️ Your Buy and Sell win rates are equal (%.0f%%). You have a balanced edge — focus on increasing trade frequency on your best pairs.\n",
			buyWinRate,
		))
	}

	return sb.String()
}

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
	btnWeakness := mainMenu.Data("✅ My Advantage", "my_weakness")
	btnAdvantage := mainMenu.Data("⚠️ My Weakness", "my_advantage")
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

	bot.Handle(&btnAdvantage, func(c tele.Context) error {
		c.Respond()
		trades, err := loadTrades(c.Sender().ID)
		if err != nil || len(trades) == 0 {
			return c.Send("⚠️ My Weakness\n\nNo trades yet. Add some trades first!")
		}
		return c.Send(analyzeWeakness(trades), mainMenu)
	})

	bot.Handle(&btnWeakness, func(c tele.Context) error {
		c.Respond()
		trades, err := loadTrades(c.Sender().ID)
		if err != nil || len(trades) == 0 {
			return c.Send("✅ My Advantage\n\nNo trades yet. Add some trades first!")
		}
		return c.Send(analyzeAdvantage(trades), mainMenu)
	})

	log.Println("Trading Journal Bot is running...")
	bot.Start()
}
