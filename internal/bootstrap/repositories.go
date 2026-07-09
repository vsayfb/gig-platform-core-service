package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vsayfb/gig-platform-core-service/internal/application"
	"github.com/vsayfb/gig-platform-core-service/internal/category"
	"github.com/vsayfb/gig-platform-core-service/internal/contract"
	"github.com/vsayfb/gig-platform-core-service/internal/gig"
	"github.com/vsayfb/gig-platform-core-service/internal/notification"
	"github.com/vsayfb/gig-platform-core-service/internal/review"
	"github.com/vsayfb/gig-platform-core-service/internal/user"
	"github.com/vsayfb/gig-platform-core-service/internal/user/auth"
	"github.com/vsayfb/gig-platform-core-service/internal/user/location"
	"github.com/vsayfb/gig-platform-core-service/internal/user/reputation"
)

type repositories struct {
	user         user.UserRepository
	auth         auth.UserAuthRepository
	category     category.CategoryRepository
	location     location.UserLocationRepository
	reputation   reputation.UserReputationRepository
	gig          gig.GigRepository
	application  application.ApplicationRepository
	contract     contract.ContractRepository
	review       review.ReviewRepository
	notification notification.NotificationRepository
}

func newRepositories(db *pgxpool.Pool) *repositories {
	return &repositories{
		user:         user.NewUserRepository(db),
		auth:         auth.NewUserAuthRepository(db),
		category:     category.NewCategoryRepository(db),
		location:     location.NewUserLocationRepository(db),
		reputation:   reputation.NewUserReputationRepository(db),
		gig:          gig.NewRepository(db),
		application:  application.NewRepository(db),
		contract:     contract.NewConctractRepository(db),
		review:       review.NewReviewRepository(db),
		notification: notification.NewNotificationRepository(db),
	}
}
