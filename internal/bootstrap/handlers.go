package bootstrap

import (
	"github.com/vsayfb/gig-platform-core-service/internal/application"
	"github.com/vsayfb/gig-platform-core-service/internal/category"
	"github.com/vsayfb/gig-platform-core-service/internal/contract"
	"github.com/vsayfb/gig-platform-core-service/internal/gig"
	"github.com/vsayfb/gig-platform-core-service/internal/notification"
	"github.com/vsayfb/gig-platform-core-service/internal/review"
	"github.com/vsayfb/gig-platform-core-service/internal/user"
	"github.com/vsayfb/gig-platform-core-service/internal/user/auth"
	"github.com/vsayfb/gig-platform-core-service/internal/user/location"
	"github.com/vsayfb/gig-platform-core-service/pkg/squs"
)

type handlers struct {
	user         *user.UserHandler
	auth         *auth.UserAuthHandler
	category     *category.CategoryHandler
	location     *location.UserLocationHandler
	gig          *gig.GigHandler
	application  *application.ApplicationHandler
	contract     *contract.Handler
	review       *review.ReviewHandler
	notification *notification.NotificationHandler
}

func newHandlers(svcs *services, sqsPublisher *squs.SQSPublisher) *handlers {
	return &handlers{
		user:         user.NewUserHandler(svcs.user),
		auth:         auth.NewUserAuthHandler(svcs.auth),
		category:     category.NewCategoryHandler(svcs.category),
		location:     location.NewUserLocationHandler(svcs.location),
		gig:          gig.NewGigHandler(svcs.gig, sqsPublisher),
		application:  application.NewApplicationHandler(svcs.application),
		contract:     contract.NewContractHandler(svcs.contract),
		review:       review.NewReviewHandler(svcs.review),
		notification: notification.NewNotificationHandler(svcs.notification),
	}
}
