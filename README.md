# 📒 Trading Journal Bot

A Telegram bot for forex traders to log, review, and manage their trades — right inside Telegram. Each user has a fully private, isolated journal.

---

## ✨ Features

- **Add a Trade** — 9-step guided flow to log every detail
- **My Journal** — Browse your trades, newest first
- **View Trade** — See full details of any trade
- **Edit Trade** — Update any trade through the same guided flow
- **Delete Trade** — Remove a trade with a confirmation step
- **Private by design** — Every user's data is fully isolated
- **SQLite storage** — All trades stored in a local `journal.db` file via GORM

---

## 📋 What Gets Logged Per Trade

| Step | Field | Description |
|---|---|---|
| 1 | Pair | e.g. XAUUSD, EURUSD, GBPUSD |
| 2 | Direction | Buy or Sell |
| 3 | Entry Price | Your entry point |
| 4 | Stop Loss | Your SL level |
| 5 | Take Profit | Your TP level |
| 6 | Date & Time | When you took the trade |
| 7 | Result | Win / Loss / Breakeven |
| 8 | Confirmation | Why you took this trade |
| 9 | Notes | Any additional observations (skippable) |

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.21+
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- GCC (required for SQLite). On Windows install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

### Installation

```bash
git clone https://github.com/majid-alikhani/trading-journal-bot.git
cd trading-journal-bot
go mod tidy
```

### Running the Bot

**Windows (PowerShell):**
```powershell
$env:TELEGRAM_TOKEN = "your_token_here"
go run main.go
```

**Linux / macOS:**
```bash
export TELEGRAM_TOKEN="your_token_here"
go run main.go
```

You should see:
```
Database ready.
Trading Journal Bot is running...
```

A `journal.db` file is created automatically on first run.

---

## 🤖 Bot Usage

### Main Menu
Send `/start` to get the main menu:

| Button | Description |
|---|---|
| 📒 My Journal | View all your logged trades |
| ➕ Add a Trade | Start the 9-step trade logging flow |
| ⚠️ My Weakness | Coming soon |
| ✅ My Advantage | Coming soon |

### Adding a Trade

Tap **➕ Add a Trade** and follow the 9 steps. Direction and Result are selected via inline buttons; everything else is typed.

### Managing Trades

**📒 My Journal** → tap any trade → choose:
- **✏️ Edit** — re-run the flow with new values
- **🗑 Delete** — remove with confirmation prompt

---

## 📁 Project Structure

```
trading-journal-bot/
├── main.go        # Bot logic, state machine, GORM setup, all handlers
├── journal.db     # SQLite database (auto-created, not committed)
├── go.mod
├── go.sum
└── .gitignore
```

---

## 🛠 Built With

- [Go](https://golang.org/) — Backend language
- [telebot.v3](https://github.com/tucnak/telebot) — Telegram bot framework
- [GORM](https://gorm.io/) — ORM for Go
- [SQLite](https://www.sqlite.org/) — Embedded database via gorm/driver/sqlite

---

## 🔒 Privacy Model

All trades are stored in a single `journal.db` file. Every database query filters by `user_id` (Telegram user ID), so users can only ever read, edit, or delete their own trades — even if they somehow know another user's trade ID.

---

## 🗺 Roadmap

- [ ] My Weakness — log recurring trading mistakes
- [ ] My Advantage — log what you do well
- [ ] Trade statistics (win rate, average RR)
- [ ] Filter journal by pair, result, or date range
- [ ] PostgreSQL support for cloud deployment

---

## 📄 License

MIT
