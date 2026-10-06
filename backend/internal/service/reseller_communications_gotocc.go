package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
)

type ResellerCommunicationSettings struct {
	ContactEnabled        bool   `json:"contact_enabled"`
	ContactInfo           string `json:"contact_info"`
	AnnouncementsEnabled  bool   `json:"announcements_enabled"`
	SyncMainAnnouncements bool   `json:"sync_main_announcements"`
}

type ResellerAnnouncement struct {
	Announcement
	SourceAnnouncementID *int64
}

type ResellerAnnouncementReview struct {
	Announcement
	ReviewStatus string
	ReviewedAt   *time.Time
}

type ResellerAnnouncementInput struct {
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content" binding:"required"`
	Status     string `json:"status" binding:"required,oneof=draft active archived"`
	NotifyMode string `json:"notify_mode" binding:"required,oneof=silent popup"`
}

type ResellerCommunicationsRepository interface {
	SaveCommunicationSettings(context.Context, int64, ResellerCommunicationSettings) error
	ListResellerAnnouncements(context.Context, int64) ([]ResellerAnnouncement, error)
	CreateResellerAnnouncement(context.Context, int64, ResellerAnnouncementInput) (*ResellerAnnouncement, error)
	UpdateResellerAnnouncement(context.Context, int64, int64, ResellerAnnouncementInput) (*ResellerAnnouncement, error)
	SetResellerAnnouncementStatus(context.Context, int64, int64, string) error
	ListMainAnnouncementReviews(context.Context, int64) ([]ResellerAnnouncementReview, error)
	ReviewMainAnnouncement(context.Context, int64, int64, time.Time, string) error
	ListCustomerAnnouncements(context.Context, int64) ([]UserAnnouncement, error)
	MarkCustomerAnnouncementRead(context.Context, int64, int64) error
}

func (s *ResellerService) SaveCommunicationSettings(ctx context.Context, ownerID int64, settings ResellerCommunicationSettings) (*ResellerProfile, error) {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return nil, err
	}
	settings.ContactInfo = strings.TrimSpace(settings.ContactInfo)
	if settings.ContactEnabled && settings.ContactInfo == "" {
		return nil, infraerrors.BadRequest("RESELLER_CONTACT_REQUIRED", "开启联系方式前请填写内容")
	}
	if err := s.Repo.SaveCommunicationSettings(ctx, ownerID, settings); err != nil {
		return nil, err
	}
	return s.Repo.Profile(ctx, ownerID)
}

func (s *ResellerService) Announcements(ctx context.Context, ownerID int64) ([]ResellerAnnouncement, error) {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return nil, err
	}
	return s.Repo.ListResellerAnnouncements(ctx, ownerID)
}

func (s *ResellerService) SaveAnnouncement(ctx context.Context, ownerID, id int64, input ResellerAnnouncementInput) (*ResellerAnnouncement, error) {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return nil, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || utf8.RuneCountInString(input.Title) > reseller.AnnouncementTitleMaxLength {
		return nil, ErrAnnouncementInvalidTitle
	}
	if input.Content == "" {
		return nil, ErrAnnouncementContentRequired
	}
	if !isValidAnnouncementStatus(input.Status) {
		return nil, ErrAnnouncementInvalidStatus
	}
	if !isValidAnnouncementNotifyMode(input.NotifyMode) {
		return nil, ErrAnnouncementInvalidNotifyMode
	}
	if id == 0 {
		return s.Repo.CreateResellerAnnouncement(ctx, ownerID, input)
	}
	return s.Repo.UpdateResellerAnnouncement(ctx, ownerID, id, input)
}

func (s *ResellerService) SetAnnouncementStatus(ctx context.Context, ownerID, id int64, status string) error {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return err
	}
	if !isValidAnnouncementStatus(status) {
		return ErrAnnouncementInvalidStatus
	}
	return s.Repo.SetResellerAnnouncementStatus(ctx, ownerID, id, status)
}

func (s *ResellerService) MainAnnouncementReviews(ctx context.Context, ownerID int64) ([]ResellerAnnouncementReview, error) {
	profile, err := s.RequireEnabled(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if !profile.SyncMainAnnouncements {
		return []ResellerAnnouncementReview{}, nil
	}
	items, err := s.Repo.ListMainAnnouncementReviews(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	user, err := s.keys.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	groups, err := s.announcementSubscriptionGroups(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	visible := make([]ResellerAnnouncementReview, 0, len(items))
	for _, item := range items {
		if item.Targeting.Matches(user.Balance, groups) {
			visible = append(visible, item)
		}
	}
	return visible, nil
}

func (s *ResellerService) ReviewMainAnnouncement(ctx context.Context, ownerID, sourceID int64, version time.Time, status string) error {
	if status != "approved" && status != "rejected" {
		return infraerrors.BadRequest("ANNOUNCEMENT_REVIEW_INVALID", "请选择通过或拒绝")
	}
	items, err := s.MainAnnouncementReviews(ctx, ownerID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == sourceID {
			return s.Repo.ReviewMainAnnouncement(ctx, ownerID, sourceID, version, status)
		}
	}
	return ErrAnnouncementNotFound
}

func (s *ResellerService) CustomerAnnouncements(ctx context.Context, userID int64, unreadOnly bool) ([]UserAnnouncement, bool, error) {
	customer, err := s.Repo.CustomerAccount(ctx, userID)
	if err != nil || customer == nil {
		return nil, false, err
	}
	items, err := s.Repo.ListCustomerAnnouncements(ctx, userID)
	if err != nil {
		return nil, true, err
	}
	groups, err := s.announcementSubscriptionGroups(ctx, userID)
	if err != nil {
		return nil, true, err
	}
	visible := make([]UserAnnouncement, 0, len(items))
	for _, item := range items {
		if (!unreadOnly || item.ReadAt == nil) && item.Announcement.Targeting.Matches(customer.CreditBalance, groups) {
			visible = append(visible, item)
		}
	}
	return visible, true, nil
}

func (s *ResellerService) ReadCustomerAnnouncement(ctx context.Context, userID, id int64) (bool, error) {
	items, customer, err := s.CustomerAnnouncements(ctx, userID, false)
	if err != nil || !customer {
		return customer, err
	}
	for _, item := range items {
		if item.Announcement.ID == id {
			return true, s.Repo.MarkCustomerAnnouncementRead(ctx, userID, id)
		}
	}
	return true, ErrAnnouncementNotFound
}

func (s *ResellerService) announcementSubscriptionGroups(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	subscriptions, err := s.keys.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups := make(map[int64]struct{}, len(subscriptions))
	for _, subscription := range subscriptions {
		groups[subscription.GroupID] = struct{}{}
	}
	return groups, nil
}
