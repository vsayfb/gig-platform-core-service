package notification

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrEmptyToken = errors.New("fcm token must not be empty")

type NotificationService struct {
	repo NotificationRepository
}

func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) RegisterFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return ErrEmptyToken
	}

	return s.repo.UpsertFCMToken(ctx, userID, token)
}

func (s *NotificationService) RemoveFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return ErrEmptyToken
	}

	return s.repo.DeleteFCMToken(ctx, userID, token)
}

func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, p ListNotificationsParams) ([]Notification, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}

	if p.Offset < 0 {
		p.Offset = 0
	}

	return s.repo.ListNotifications(ctx, userID, p)
}
