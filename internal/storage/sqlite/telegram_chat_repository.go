package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"diaryhero/internal/domain"
)

type TelegramChatRepository struct {
	db *sql.DB
}

func NewTelegramChatRepository(db *sql.DB) *TelegramChatRepository {
	return &TelegramChatRepository{db: db}
}

func (r *TelegramChatRepository) UpsertChat(ctx context.Context, chat domain.TelegramChat) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO telegram_chats (chat_id, chat_type, title, username, is_default, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id) DO UPDATE SET
			chat_type = excluded.chat_type,
			title = excluded.title,
			username = excluded.username,
			source = excluded.source,
			updated_at = excluded.updated_at
	`, chat.ChatID, chat.ChatType, chat.Title, chat.Username, boolToInt(chat.IsDefault), chat.Source, now, now)
	if err != nil {
		return fmt.Errorf("upsert telegram chat: %w", err)
	}
	return nil
}

func (r *TelegramChatRepository) SetDefaultChat(ctx context.Context, chatID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE telegram_chats SET is_default = 0 WHERE is_default = 1`); err != nil {
		return fmt.Errorf("clear default telegram chat: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE telegram_chats
		SET is_default = 1, updated_at = ?
		WHERE chat_id = ?
	`, time.Now().UTC().Format(time.RFC3339), chatID)
	if err != nil {
		return fmt.Errorf("set default telegram chat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read default telegram chat update result: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit default telegram chat: %w", err)
	}

	return nil
}

func (r *TelegramChatRepository) GetDefaultChatID(ctx context.Context) (string, error) {
	var chatID string
	err := r.db.QueryRowContext(ctx, `
		SELECT chat_id
		FROM telegram_chats
		WHERE is_default = 1
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&chatID)
	if err != nil {
		return "", err
	}
	return chatID, nil
}

func (r *TelegramChatRepository) GetLatestPublishableChannelID(ctx context.Context) (string, error) {
	var chatID string
	err := r.db.QueryRowContext(ctx, `
		SELECT chat_id
		FROM telegram_chats
		WHERE chat_type = 'channel'
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&chatID)
	if err != nil {
		return "", err
	}
	return chatID, nil
}

func (r *TelegramChatRepository) GetChatByID(ctx context.Context, chatID string) (domain.TelegramChat, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT chat_id, chat_type, title, username, is_default, source, created_at, updated_at
		FROM telegram_chats
		WHERE chat_id = ?
	`, chatID)

	var chat domain.TelegramChat
	var isDefault int
	var createdAt string
	var updatedAt string
	if err := row.Scan(&chat.ChatID, &chat.ChatType, &chat.Title, &chat.Username, &isDefault, &chat.Source, &createdAt, &updatedAt); err != nil {
		return domain.TelegramChat{}, err
	}
	chat.IsDefault = isDefault == 1
	chat.CreatedAt = parseSQLiteTime(createdAt)
	chat.UpdatedAt = parseSQLiteTime(updatedAt)
	return chat, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
