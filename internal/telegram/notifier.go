// internal/telegram/notifier.go
package telegram

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/override/volato/internal/api"
)

// Notifier sends messages via Telegram.
type Notifier struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

// NewNotifier creates a new Telegram notifier.
func NewNotifier(token string, chatID int64) (*Notifier, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	return &Notifier{bot: bot, chatID: chatID}, nil
}

// SendDeal sends a deal notification.
func (n *Notifier) SendDeal(flight api.Flight) error {
	msg := tgbotapi.NewMessage(n.chatID, FormatDealMessage(flight))
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := n.bot.Send(msg)
	return err
}

// SendText sends a plain text message.
func (n *Notifier) SendText(text string) error {
	msg := tgbotapi.NewMessage(n.chatID, text)
	_, err := n.bot.Send(msg)
	return err
}

// FormatDealMessage formats a flight as a deal notification.
func FormatDealMessage(f api.Flight) string {
	duration := int(f.ReturnDate.Sub(f.DepartureDate).Hours() / 24)

	var sb strings.Builder
	sb.WriteString("✈️ <b>Deal Found!</b>\n\n")
	sb.WriteString(fmt.Sprintf("🛫 %s → %s\n", f.Origin, f.Destination))
	sb.WriteString(fmt.Sprintf("📅 %s - %s (%d days)\n",
		f.DepartureDate.Format("Jan 2"),
		f.ReturnDate.Format("Jan 2, 2006"),
		duration))
	sb.WriteString(fmt.Sprintf("💰 $%.0f %s\n", f.Price, f.Currency))

	if f.BookingLink != "" {
		sb.WriteString(fmt.Sprintf("\n🔗 <a href=\"%s\">Book now</a>", f.BookingLink))
	}

	return sb.String()
}

// FormatPriceDropMessage adds drop info to the deal message.
func FormatPriceDropMessage(f api.Flight, dropPct, avgPrice float64) string {
	base := FormatDealMessage(f)
	dropInfo := fmt.Sprintf("\n\n📉 <i>%.0f%% below average ($%.0f)</i>", dropPct, avgPrice)
	return base + dropInfo
}
