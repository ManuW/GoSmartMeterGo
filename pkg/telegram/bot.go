package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"GoSmartMeterGo/pkg/config"
)

// BotWrapper wraps the Telegram bot client and configuration.
type BotWrapper struct {
	cfg *config.TelegramConfig
	bot *bot.Bot
}

// NewBotWrapper initializes the Telegram bot if enabled.
func NewBotWrapper(cfg *config.TelegramConfig) (*BotWrapper, error) {
	wrapper := &BotWrapper{cfg: cfg}

	if !cfg.Enabled {
		slog.Info("Telegram bot is disabled by configuration")
		return wrapper, nil
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is empty, but bot is enabled")
	}

	b, err := bot.New(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	wrapper.bot = b
	slog.Info("Telegram bot initialized successfully")
	return wrapper, nil
}

// Start runs the bot's update listener in the background.
// This is optional if the bot is only used to send messages and not receive commands.
// We still run it to keep connection alive or handle potential user interactions.
func (bw *BotWrapper) Start(ctx context.Context) {
	if bw.bot == nil {
		return
	}
	slog.Info("Telegram bot listener starting")
	bw.bot.Start(ctx)
}

// SendMessage sends a text message to all configured chat IDs.
func (bw *BotWrapper) SendMessage(ctx context.Context, text string) {
	if bw.bot == nil {
		return
	}

	for _, chatID := range bw.cfg.ChatIDs {
		_, err := bw.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      text,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			slog.Error("Failed to send Telegram message", "chat_id", chatID, "error", err)
		} else {
			slog.Debug("Telegram message sent successfully", "chat_id", chatID)
		}
	}
}
