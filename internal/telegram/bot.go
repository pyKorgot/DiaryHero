package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"diaryhero/internal/config"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	logger        *slog.Logger
	b             *tgbot.Bot
	defaultChatID string
	enabled       bool
	startOnce     sync.Once
	startErr      error
}

func New(cfg config.TelegramConfig, logger *slog.Logger) (*Bot, error) {
	client := &Bot{
		logger:        logger,
		defaultChatID: cfg.DefaultChatID,
		enabled:       cfg.BotToken != "",
	}

	if !client.enabled {
		return client, nil
	}

	b, err := tgbot.New(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	client.b = b
	client.registerHandlers()

	return client, nil
}

func (b *Bot) Enabled() bool {
	return b != nil && b.enabled && b.b != nil
}

func (b *Bot) Start(ctx context.Context) error {
	if !b.Enabled() {
		return nil
	}

	b.startOnce.Do(func() {
		go b.b.Start(ctx)
	})

	return b.startErr
}

func (b *Bot) PublishText(ctx context.Context, text string) error {
	if !b.Enabled() {
		return fmt.Errorf("telegram bot is not configured")
	}
	if b.defaultChatID == "" {
		return fmt.Errorf("telegram default chat id is not configured")
	}

	_, err := b.b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: b.defaultChatID,
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}

func (b *Bot) registerHandlers() {
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.handleStart)
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypeContains, b.handleAnyMessage)
}

func (b *Bot) handleStart(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	chatType := update.Message.Chat.Type
	text := fmt.Sprintf(
		"DiaryHero подключен. chat_id: %d\n\nЧтобы публиковать сюда записи, укажи TELEGRAM_DEFAULT_CHAT_ID=%d и перезапусти сервис.",
		chatID,
		chatID,
	)

	if chatType != "private" {
		text += "\n\nЕсли это группа или канал, убедись, что у бота есть права на отправку сообщений."
	}

	if _, err := bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		b.logger.Error("failed to reply to /start", "error", err)
		return
	}

	b.logger.Info("telegram /start received", "chat_id", chatID, "chat_type", chatType)
}

func (b *Bot) handleAnyMessage(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	b.logger.Info(
		"telegram message received",
		"chat_id", update.Message.Chat.ID,
		"chat_type", update.Message.Chat.Type,
		"text", update.Message.Text,
	)

	if update.Message.Text == "/chatid" {
		_, err := bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Текущий chat_id: %d", update.Message.Chat.ID),
		})
		if err != nil {
			b.logger.Error("failed to reply with chat id", "error", err)
		}
	}
}
