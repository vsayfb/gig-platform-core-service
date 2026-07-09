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
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
)

type services struct {
	user         user.UserService
	auth         auth.UserAuthService
	category     category.CategoryService
	location     location.UserLocationService
	reputation   reputation.UserReputationService
	gig          gig.GigService
	application  application.ApplicationService
	contract     contract.ContractService
	review       review.ReviewService
	notification notification.NotificationService
}

func newServices(repos *repositories, googleVerifier *google.Verifier, jwtManager *jwt.Manager, db *pgxpool.Pool) *services {
	reputationSvc := reputation.NewUserReputationService(repos.reputation)

	return &services{
		user:         user.NewUserService(repos.user),
		reputation:   reputationSvc,
		auth:         auth.NewUserAuthService(repos.auth, repos.user, reputationSvc, googleVerifier, jwtManager, db),
		category:     category.NewCategoryService(repos.category),
		location:     location.NewUserLocationService(repos.location),
		gig:          gig.NewGigService(repos.gig),
		application:  application.NewApplicationService(repos.application, repos.gig),
		contract:     contract.NewContractService(repos.contract, repos.application, repos.gig, db),
		review:       review.NewReviewService(repos.review, repos.contract, reputationSvc, db),
		notification: notification.NewNotificationService(repos.notification),
	}
}
