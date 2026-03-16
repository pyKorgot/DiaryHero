package telegram

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"diaryhero/internal/config"
	"diaryhero/internal/domain"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	logger    *slog.Logger
	b         *tgbot.Bot
	chatRepo  domain.TelegramChatRepository
	enabled   bool
	startOnce sync.Once
	startErr  error
}

func New(cfg config.TelegramConfig, logger *slog.Logger, chatRepo domain.TelegramChatRepository) (*Bot, error) {
	client := &Bot{
		logger:   logger,
		chatRepo: chatRepo,
		enabled:  cfg.BotToken != "",
	}

	if !client.enabled {
		return client, nil
	}

	allowedUpdates := tgbot.AllowedUpdates{
		models.AllowedUpdateChannelPost,
		models.AllowedUpdateMyChatMember,
	}

	b, err := tgbot.New(cfg.BotToken,
		tgbot.WithDefaultHandler(client.handleDefault),
		tgbot.WithAllowedUpdates(allowedUpdates),
	)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	client.b = b

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

	chatID, err := b.resolveTargetChatID(ctx)
	if err != nil {
		return err
	}

	_, err = b.b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: normalizeChatID(chatID),
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}

func (b *Bot) handleDefault(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	if update == nil {
		return
	}

	if update.ChannelPost != nil {
		if _, err := b.registerChat(ctx, update.ChannelPost.Chat, "channel_post", false); err != nil {
			b.logger.Warn("failed to register telegram channel chat", "error", err)
		}
		return
	}

	if update.MyChatMember != nil {
		setAsDefault, err := b.registerChat(ctx, update.MyChatMember.Chat, "membership", update.MyChatMember.Chat.Type == models.ChatTypeChannel)
		if err != nil {
			b.logger.Warn("failed to register telegram membership chat", "error", err)
			return
		}
		b.logger.Info(
			"telegram chat membership updated",
			"chat_id", update.MyChatMember.Chat.ID,
			"chat_type", update.MyChatMember.Chat.Type,
			"new_status", update.MyChatMember.NewChatMember.Type,
			"default_selected", setAsDefault,
		)
	}
}

func (b *Bot) registerChat(ctx context.Context, chat models.Chat, source string, allowAutoDefault bool) (bool, error) {
	if b.chatRepo == nil {
		return false, nil
	}
	if chat.Type != models.ChatTypeChannel {
		return false, nil
	}

	chatID := strconv.FormatInt(chat.ID, 10)
	telegramChat := domain.TelegramChat{
		ChatID:   chatID,
		ChatType: string(chat.Type),
		Title:    chat.Title,
		Username: chat.Username,
		Source:   source,
	}
	if telegramChat.Title == "" {
		telegramChat.Title = strings.TrimSpace(strings.Join([]string{chat.FirstName, chat.LastName}, " "))
	}

	if err := b.chatRepo.UpsertChat(ctx, telegramChat); err != nil {
		return false, err
	}

	if !allowAutoDefault {
		return false, nil
	}

	if _, err := b.chatRepo.GetDefaultChatID(ctx); err == nil {
		return false, nil
	} else if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if err := b.chatRepo.SetDefaultChat(ctx, chatID); err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bot) resolveTargetChatID(ctx context.Context) (string, error) {
	if b.chatRepo != nil {
		chatID, err := b.chatRepo.GetDefaultChatID(ctx)
		if err == nil && chatID != "" {
			return chatID, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("resolve default telegram chat: %w", err)
		}

		chatID, err = b.chatRepo.GetLatestPublishableChannelID(ctx)
		if err == nil && chatID != "" {
			return chatID, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("resolve latest publishable telegram channel: %w", err)
		}
	}

	return "", fmt.Errorf("telegram channel target is not configured; add the bot to a channel as admin")
}

func normalizeChatID(chatID string) any {
	if parsed, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		return parsed
	}
	return chatID
}
