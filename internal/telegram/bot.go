// internal/telegram/bot.go
package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CheckFunc is called when /check command is received.
type CheckFunc func(ctx context.Context) error

// StatusInfo contains bot status information.
type StatusInfo struct {
	LastCheck  time.Time
	DealsFound int
	NextCheck  string
}

// Bot handles Telegram bot commands.
type Bot struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	checkFunc  CheckFunc
	statusFunc func() StatusInfo
	dealsFunc  func() []string

	mu            sync.Mutex
	lastCheckTime time.Time
}

// NewBot creates a new Telegram bot handler.
func NewBot(token string, chatID int64) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	return &Bot{bot: bot, chatID: chatID}, nil
}

// SetCheckFunc sets the function to call for /check command.
func (b *Bot) SetCheckFunc(f CheckFunc) {
	b.checkFunc = f
}

// SetStatusFunc sets the function to call for /status command.
func (b *Bot) SetStatusFunc(f func() StatusInfo) {
	b.statusFunc = f
}

// SetDealsFunc sets the function to call for /deals command.
func (b *Bot) SetDealsFunc(f func() []string) {
	b.dealsFunc = f
}

// Run starts the bot and listens for commands.
func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}

			// Only respond to configured chat
			if update.Message.Chat.ID != b.chatID {
				continue
			}

			switch update.Message.Command() {
			case "status":
				b.handleStatus(update.Message.Chat.ID)
			case "check":
				b.handleCheck(ctx, update.Message.Chat.ID)
			case "deals":
				b.handleDeals(update.Message.Chat.ID)
			case "help":
				b.handleHelp(update.Message.Chat.ID)
			}
		}
	}
}

func (b *Bot) handleStatus(chatID int64) {
	var text string
	if b.statusFunc != nil {
		info := b.statusFunc()
		text = fmt.Sprintf("📊 <b>Status</b>\n\n"+
			"Last check: %s\n"+
			"Deals found: %d\n"+
			"Next check: %s",
			info.LastCheck.Format("Jan 2, 15:04"),
			info.DealsFound,
			info.NextCheck)
	} else {
		text = "Status information unavailable"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}

func (b *Bot) handleCheck(ctx context.Context, chatID int64) {
	// Rate limit: 1 hour cooldown
	b.mu.Lock()
	if time.Since(b.lastCheckTime) < time.Hour {
		remaining := time.Hour - time.Since(b.lastCheckTime)
		b.mu.Unlock()
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Please wait %d minutes before checking again", int(remaining.Minutes())))
		b.bot.Send(msg)
		return
	}
	b.lastCheckTime = time.Now()
	b.mu.Unlock()

	msg := tgbotapi.NewMessage(chatID, "🔍 Starting flight check...")
	b.bot.Send(msg)

	if b.checkFunc != nil {
		if err := b.checkFunc(ctx); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Check failed: %v", err))
			b.bot.Send(msg)
			return
		}
	}

	msg = tgbotapi.NewMessage(chatID, "✅ Check complete!")
	b.bot.Send(msg)
}

func (b *Bot) handleDeals(chatID int64) {
	var text string
	if b.dealsFunc != nil {
		deals := b.dealsFunc()
		if len(deals) == 0 {
			text = "No recent deals found"
		} else {
			text = "📋 <b>Recent Deals</b>\n\n" + strings.Join(deals, "\n\n")
		}
	} else {
		text = "Deals information unavailable"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}

func (b *Bot) handleHelp(chatID int64) {
	text := `🤖 <b>Volato Bot Commands</b>

/status - Show bot status and last check time
/check - Trigger a manual flight check (1hr cooldown)
/deals - Show recent deals found
/help - Show this help message`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}
