# 📒 Trading Journal Bot

A Telegram bot for forex traders to log, review, and manage their trades — right inside Telegram.

---

## ✨ Features

- **Add a Trade** — 9-step guided flow to log every detail of a trade
- **My Journal** — Browse all your trades, newest first
- **View Trade** — See full details of any trade
- **Edit Trade** — Update any existing trade through the same guided flow
- **Delete Trade** — Remove a trade with a confirmation step
- Trades are persisted locally in a `trades.json` file

---

## 📋 What Gets Logged Per Trade

| Field | Description |
|---|---|
| Pair | e.g. XAUUSD, EURUSD, GBPUSD |
| Direction | Buy or Sell |
| Entry Price | Your entry point |
| Stop Loss | Your SL level |
| Take Profit | Your TP level |
| Date & Time | When you took the trade |
| Result | Win / Loss / Breakeven |
| Confirmation | Why you took the trade |
| Notes | Any additional observations |

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.21+
- A Telegram bot token from [@BotFather](https://t.me/BotFather)

### Installation

```bash
git clone https://github.com/majid-alikhani/trading-journal-bot.git
cd trading-journal-bot
go mod tidy
```

### Running the Bot

Set your token and run:

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
Trading Journal Bot is running...
```

Open Telegram, find your bot, and send `/start`.

---

## 🤖 Bot Usage

### Main Menu
After `/start` you'll see 4 buttons:

| Button | Description |
|---|---|
| 📒 My Journal | View all logged trades |
| ➕ Add a Trade | Start the 9-step trade logging flow |
| ⚠️ My Weakness | Coming soon |
| ✅ My Advantage | Coming soon |

### Adding a Trade
Tap **➕ Add a Trade** and follow the steps:

```
Step 1 — Pair            (type it)
Step 2 — Direction       (Buy / Sell buttons)
Step 3 — Entry Price     (type it)
Step 4 — Stop Loss       (type it)
Step 5 — Take Profit     (type it)
Step 6 — Date & Time     (type it, e.g. 2024-06-22 14:30)
Step 7 — Result          (Win / Loss / Breakeven buttons)
Step 8 — Confirmation    (why did you take this trade?)
Step 9 — Notes           (type it or skip)
```

### Viewing & Managing Trades
Tap **📒 My Journal** → select a trade → choose:
- **✏️ Edit** — update the trade through the same flow
- **🗑 Delete** — remove with confirmation

---

## 📁 Project Structure

```
trading-journal-bot/
├── main.go        # Bot logic, state machine, handlers
├── trades.json    # Trade storage (auto-created on first save)
├── go.mod
├── go.sum
└── .gitignore
```

---

## 🛠 Built With

- [Go](https://golang.org/) — Backend language
- [telebot.v3](https://github.com/tucnak/telebot) — Telegram bot framework

---

## 🗺 Roadmap

- [ ] My Weakness — log recurring trading mistakes
- [ ] My Advantage — log what you do well
- [ ] Trade statistics (win rate, RR average)
- [ ] Filter journal by pair or result
- [ ] PostgreSQL storage for multi-user support

---

## 📄 License

MIT
