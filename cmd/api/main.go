package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/vsayfb/gig-platform-core-service/config"
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
	"github.com/vsayfb/gig-platform-core-service/pkg/database"
	"github.com/vsayfb/gig-platform-core-service/pkg/google"
	"github.com/vsayfb/gig-platform-core-service/pkg/grpcserver"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/logger"
	"github.com/vsayfb/gig-platform-core-service/pkg/metrics"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
	"github.com/vsayfb/gig-platform-core-service/pkg/squs"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	pb "github.com/vsayfb/gig-platform-protos/contracts"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger.Init(cfg.Env)

	slog.Info("starting app", "env", cfg.Env)

	db, err := database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		slog.Error("failed to connect to db", "err", err)

		os.Exit(1)
	}

	defer db.Close()

	slog.Info("db connected")

	if cfg.Env != "production" {
		if err := database.RunMigrations(cfg.DB.URL()); err != nil {
			slog.Error("failed to run migrations", "err", err)

			os.Exit(1)
		}
	}

	googleVerifier, err := google.NewVerifier(ctx, cfg.Google.ClientID)

	if err != nil {
		slog.Error("failed to initialize google verifier", "err", err)

		os.Exit(1)
	}

	slog.Info("google oidc verifier initialized")

	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Expiration)

	awsConfig, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithRegion("us-east-1"),
		awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	sqsClient := sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(cfg.SQS.BaseURL)
	})

	sqsPublisher := squs.NewSQSPublisher(sqsClient, cfg.SQS.QueueURL)

	userRepo := user.NewUserRepository(db)
	authRepo := auth.NewUserAuthRepository(db)
	categoryRepo := category.NewCategoryRepository(db)
	locationRepo := location.NewUserLocationRepository(db)
	reputationRepo := reputation.NewUserReputationRepository(db)
	gigRepo := gig.NewRepository(db)
	applicationRepo := application.NewRepository(db)
	contractRepo := contract.NewConctractRepository(db)
	reviewRepo := review.NewReviewRepository(db)
	notificationRepo := notification.NewNotificationRepository(db)

	userService := user.NewUserService(userRepo)
	reputationService := reputation.NewUserReputationService(reputationRepo)
	authService := auth.NewUserAuthService(authRepo, userRepo, reputationService, googleVerifier, jwtManager, db)
	categoryService := category.NewCategoryService(categoryRepo)
	locationService := location.NewUserLocationService(locationRepo)
	gigService := gig.NewGigService(gigRepo)
	applicationService := application.NewApplicationService(applicationRepo, gigRepo)
	contractService := contract.NewContractService(contractRepo, applicationRepo, gigRepo, db)
	reviewService := review.NewReviewService(reviewRepo, contractRepo, *reputationService, db)
	notificationService := notification.NewNotificationService(*notificationRepo)

	userHandler := user.NewUserHandler(userService)
	authHandler := auth.NewUserAuthHandler(authService)
	categoryHandler := category.NewCategoryHandler(categoryService)
	locationHandler := location.NewUserLocationHandler(locationService)
	gigHandler := gig.NewGigHandler(gigService, sqsPublisher)
	applicationHandler := application.NewApplicationHandler(applicationService)
	contractHandler := contract.NewContractHandler(contractService)
	reviewHandler := review.NewReviewHandler(reviewService)
	notificationHandler := notification.NewNotificationHandler(notificationService)

	slog.Info("dependencies injected")

	r := chi.NewRouter()

	if cfg.Env != "production" {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		}))

	}

	metrics.Register()

	metrics_svc := metrics.StartServer(cfg.REST.MetricsServerPort)

	r.Use(chimiddleware.RequestID)
	r.Use(middleware.StructuredLogger)
	r.Use(middleware.MetricsMiddleware)
	r.Use(chimiddleware.Recoverer)

	r.Group(func(r chi.Router) {
		authHandler.RegisterRoutes(r)
		categoryHandler.RegisterRoutes(r, jwtManager)
		gigHandler.RegisterRoutes(r, jwtManager)
		applicationHandler.RegisterRoutes(r, jwtManager)
		reviewHandler.RegisterRoutes(r, jwtManager)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		userHandler.RegisterRoutes(r)
		locationHandler.RegisterRoutes(r)
		contractHandler.RegisterRoutes(r)
		notificationHandler.RegisterRoutes(r)
	})

	grpcHandler := grpcserver.NewGRPCHandler(userService)

	grpcService := grpcserver.New(cfg.GRPC.Port)

	pb.RegisterUserServiceServer(grpcService.GRPCServer(), grpcHandler)

	go func() {
		if err := grpcService.Start(); err != nil {
			slog.Error("grpc failed", "err", err)
			os.Exit(1)
		}

	}()

	httpSrv := &http.Server{
		Addr:    ":" + cfg.REST.Port,
		Handler: r,
	}

	go func() {
		slog.Info("grpc ready", "port", cfg.GRPC.Port)
		slog.Info("rest ready", "port", cfg.REST.Port)

		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("rest failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	grpcService.Stop()

	_ = httpSrv.Shutdown(ctx)
	_ = metrics_svc.Shutdown(ctx)

	slog.Info("shutdown complete")
}
