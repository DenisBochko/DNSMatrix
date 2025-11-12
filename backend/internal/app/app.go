package app

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"hackathon-back/internal/msg/inbox"
	elasticsearch "hackathon-back/pkg/article"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"hackathon-back/internal/api/http/handler"
	"hackathon-back/internal/api/http/route"
	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/config"
	"hackathon-back/internal/model"
	"hackathon-back/internal/msg/outbox"
	"hackathon-back/internal/repository"
	"hackathon-back/internal/service"
	"hackathon-back/pkg/geoip"
	"hackathon-back/pkg/jwt"
	"hackathon-back/pkg/kafka"
	"hackathon-back/pkg/mailer"
	"hackathon-back/pkg/postgres"
	"hackathon-back/pkg/redis"
	"hackathon-back/pkg/server"
)

const (
	consumerBufferSize = 1000
)

const defaultTimeout = 15 * time.Second

// ДОБАВИТЬ FAQ интерфейсы
/*
type FAQRepository interface {
	Create(ctx context.Context, ext interface{}, faq *model.FAQ) error
	GetByID(ctx context.Context, ext interface{}, id uuid.UUID) (*model.FAQ, error)
	Update(ctx context.Context, ext interface{}, id uuid.UUID, updateData *model.FAQUpdateRequest) error
	Delete(ctx context.Context, ext interface{}, id uuid.UUID) error
	List(ctx context.Context, ext interface{}, params model.FAQQueryParams) ([]model.FAQ, int, error)
	GetByCategory(ctx context.Context, ext interface{}, category string) ([]model.FAQ, error)
	GetCategories(ctx context.Context, ext interface{}) ([]string, error)
}
*/

type FAQRepository interface {
	Create(ctx context.Context, ext repository.RepoExtension, faq *model.FAQ) error
	GetByID(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) (*model.FAQ, error)
	Update(ctx context.Context, ext repository.RepoExtension, id uuid.UUID, updateData *model.FAQUpdateRequest) error
	Delete(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error
	List(ctx context.Context, ext repository.RepoExtension, params model.FAQQueryParams) ([]model.FAQ, int, error)
	GetByCategory(ctx context.Context, ext repository.RepoExtension, category string) ([]model.FAQ, error)
	GetCategories(ctx context.Context, ext repository.RepoExtension) ([]string, error)
}

type FAQService interface {
	Create(ctx context.Context, req *model.FAQCreateRequest, createdBy uuid.UUID) (*model.FAQ, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.FAQ, error)
	Update(ctx context.Context, id uuid.UUID, req *model.FAQUpdateRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params model.FAQQueryParams) (*model.FAQListResponse, error)
	GetByCategory(ctx context.Context, category string) ([]model.FAQ, error)
	GetCategories(ctx context.Context) ([]string, error)
	GetCategoriesWithFAQs(ctx context.Context) ([]model.FAQCategoryResponse, error)
}

type FAQHandler interface {
	CreateFAQ(c *gin.Context)
	GetFAQ(c *gin.Context)
	UpdateFAQ(c *gin.Context)
	DeleteFAQ(c *gin.Context)
	ListFAQs(c *gin.Context)
	GetFAQsByCategory(c *gin.Context)
	GetCategories(c *gin.Context)
	GetCategoriesWithFAQs(c *gin.Context)
}

// Существующие интерфейсы остаются без изменений...
type HealthRepository interface {
	IsOK() (bool, error)
	SelectData(ctx context.Context, ext repository.RepoExtension) (*model.TestTable, error)
}

type HealthService interface {
	IsOK() (bool, error)
	GetTestData(ctx context.Context) (*model.TestTable, error)
}

type HealthHandler interface {
	Ping(c *gin.Context)
	ProtectedPing(c *gin.Context)
	Health(c *gin.Context)
}

type AuthRepository interface {
	Pool() *pgxpool.Pool
	UpdateUserAsConfirmed(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID) error
	InsertVerificationToken(ctx context.Context, ext repository.RepoExtension, verificationToken *model.VerificationToken) error
	SelectVerificationToken(ctx context.Context, ext repository.RepoExtension, token []byte) (*model.VerificationToken, error)
	DeleteVerificationTokenByUserID(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID) error
}

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (user *model.User, userToken []byte, err error)
	ResendConfirmation(ctx context.Context, email string) ([]byte, error)
	Confirmation(ctx context.Context, incCode string, incToken []byte) error
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error)
	TestLogin(ctx context.Context) (accessToken, refreshToken string, err error)
}

type AuthHandler interface {
	Register(c *gin.Context)
	ResendConfirmation(c *gin.Context)
	Confirmation(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	Refresh(c *gin.Context)
	TestLogin(c *gin.Context)
}

type UserRepository interface {
	Pool() *pgxpool.Pool
	InsertUser(ctx context.Context, ext repository.RepoExtension, user *model.User) (*model.User, error)
	InsertTestUser(ctx context.Context, ext repository.RepoExtension, user *model.User) (*model.User, error)
	SelectUserByID(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) (*model.User, error)
	SelectUserByEmail(ctx context.Context, ext repository.RepoExtension, email string) (*model.User, error)
	Delete(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error
	Block(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error
	InsertPasswordResetToken(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID, token []byte, expiresAt time.Time) error
	SelectUserByResetToken(ctx context.Context, ext repository.RepoExtension, token []byte) (*model.User, error)
	DeletePasswordResetToken(ctx context.Context, ext repository.RepoExtension, token []byte) error
	UpdateUserPassword(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID, hashedPassword []byte) error
}

type UserService interface {
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	BlockUser(ctx context.Context, id uuid.UUID) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	DeleteSelf(ctx context.Context, userID uuid.UUID) error
}

type UserHandler interface {
	DeleteUser(c *gin.Context)
	BlockUser(c *gin.Context)
	GetUser(*gin.Context)
	GetUserJWT(c *gin.Context)
	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
	DeleteSelf(c *gin.Context)
}

type ArticleRepository interface {
	EnsureIndex(ctx context.Context) (err error)
	Create(ctx context.Context, article *model.Article) (err error)
	Get(ctx context.Context, id string) (article *model.Article, err error)
	Delete(ctx context.Context, id string) (err error)
	Patch(ctx context.Context, id string, fields map[string]interface{}) (err error)
	Search(ctx context.Context, query string, from, size int, sort string) (results []model.SearchResult, err error)
}

type ArticleService interface {
	CreateArticle(ctx context.Context, req *model.ArticleCreateRequest) (*model.Article, error)
	GetArticle(ctx context.Context, id string) (*model.Article, error)
	DeleteArticle(ctx context.Context, id string) error
	UpdateArticle(ctx context.Context, id string, upd model.ArticleUpdate) error
	SearchArticles(ctx context.Context, query string) ([]model.SearchResult, error)
}

type ArticleHandler interface {
	CreateArticle(c *gin.Context)
	GetArticle(c *gin.Context)
	DeleteArticle(c *gin.Context)
	UpdateArticle(c *gin.Context)
	SearchArticles(c *gin.Context)
}

type APIKeyRepository interface {
	Insert(ctx context.Context, key *model.APIKey) error
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	GetAllActive(ctx context.Context) ([]model.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type APIKeyRepositoryInterface interface {
	GetAllActive(ctx context.Context) ([]model.APIKey, error)
}

type APIKeyService interface {
	Generate(ctx context.Context, userID uuid.UUID, name string, ttl time.Duration) (string, error)
	GetUserKeys(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type APIKeyHandler interface {
	Create(c *gin.Context)
	List(c *gin.Context)
	Revoke(c *gin.Context)
}

type OutboxRepository interface {
	InsertMessage(ctx context.Context, ext repository.RepoExtension, message model.OutboxMessage) error
	UpdateAsSent(ctx context.Context, ext repository.RepoExtension, messageID uuid.UUID) error
	SelectUnsentBatch(ctx context.Context, ext repository.RepoExtension, batchSize int) ([]model.OutboxMessage, error)
}

type InboxRepository interface {
	InsertMessage(ctx context.Context, ext repository.RepoExtension, message model.InboxMessage) error
	UpdateAsProcessed(ctx context.Context, ext repository.RepoExtension, messageID uuid.UUID) error
	SelectUnprocessedBatch(ctx context.Context, ext repository.RepoExtension, batchSize int) ([]model.InboxMessage, error)
}

type Publisher interface {
	Run(ctx context.Context)
}

type Subscriber interface {
	Run(ctx context.Context)
}

type AgentRepository interface {
	SelectAgents(ctx context.Context, ext repository.RepoExtension) ([]*model.Agent, error)
	SelectAgentByRegion(ctx context.Context, ext repository.RepoExtension, region string) (*model.Agent, error)
}

type RequestRepository interface {
	Pool() *pgxpool.Pool

	SelectResultsByRequestID(ctx context.Context, ext repository.RepoExtension, requestID uuid.UUID) ([]model.CheckResultResponse, error)
	InsertRequest(ctx context.Context, ext repository.RepoExtension, request *model.Request) error
	InsertAssignment(ctx context.Context, ext repository.RepoExtension, assignment *model.Assignment) error
	InsertCheckResult(ctx context.Context, ext repository.RepoExtension, checkResult *model.CheckResult) error
}

type RequestService interface {
	CreateRequest(ctx context.Context, req model.TaskMessageRequest, ip net.IP, ua string) (*model.Request, error)
	GetResultsByRequestID(ctx context.Context, requestID uuid.UUID) ([]model.CheckResultResponse, error)
}

type RequestHandler interface {
	CreateRequest(c *gin.Context)
	GetResults(c *gin.Context)
	StreamResults(c *gin.Context)
}

type App struct {
	Cfg        *config.Config
	Log        *zap.Logger
	Handler    *Handler
	Service    *Service
	Security   *Security
	DB         postgres.Postgres
	RDB        redis.Redis
	Mailer     mailer.Mailer
	HTTPServer server.HTTPServer
	EBus       *EBus
	GeoDB      geoip.GeoIP
}

// ДОБАВИТЬ FAQRepository в структуру Repository
type Repository struct {
	ArticleRepository ArticleRepository
	APIKeyRepository  APIKeyRepository
	FAQRepository     FAQRepository // ДОБАВИТЬ
	HealthRepository  HealthRepository
	AuthRepository    AuthRepository
	UserRepository    UserRepository
	OutboxRepository  OutboxRepository
	InboxRepository   InboxRepository
	AgentRepository   AgentRepository
	RequestRepository RequestRepository
}

// ДОБАВИТЬ FAQService в структуру Service
type Service struct {
	HealthService  HealthService
	AuthService    AuthService
	UserService    *service.UserService
	RequestService RequestService
	ArticleService ArticleService
	APIKeyService  APIKeyService
	FAQService     FAQService // ДОБАВИТЬ
}

// ДОБАВИТЬ FAQHandler в структуру Handler
type Handler struct {
	RequestHandler RequestHandler
	HealthHandler  HealthHandler
	AuthHandler    AuthHandler
	UserHandler    UserHandler
	ArticleHandler ArticleHandler
	APIKeyHandler  APIKeyHandler
	FAQHandler     FAQHandler // ДОБАВИТЬ
}

type Security struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

type EBus struct {
	OutboxPublisher Publisher
	InboxSubscriber Subscriber
}

func New(cfg *config.Config, log *zap.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	db, err := initDB(&cfg.Database)
	if err != nil {
		log.Error("Failed to initialize database", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	rdb, err := initRedis(&cfg.Redis)
	if err != nil {
		log.Error("Failed to initialize redis", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize redis: %w", err)
	}

	sec, err := initSecurity(log, cfg.Key)
	if err != nil {
		log.Error("Failed to initialize security", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize security: %w", err)
	}

	mlr := initMailer(log, &cfg.Mailer)

	es, err := initElastic(log, &cfg.Elastic)
	if err != nil {
		log.Error("Failed to initialize elastic", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize elastic: %w", err)
	}

	repo := initRepository(log, db, es)

	if err := repo.ArticleRepository.EnsureIndex(ctx); err != nil {
		log.Error("Failed to EnsureIndex an article repository", zap.Error(err))
		return nil, fmt.Errorf("failed to EnsureIndex an article repository: %w", err)
	}

	geo, err := initGeo(log, &cfg.Geo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize geo: %w", err)
	}

	svc := initService(log, &cfg.JWT, sec, repo, mlr, rdb, geo)

	hdl := initHandler(log, &cfg.JWT, svc)

	httpServer := initHTTPServer(log, cfg, sec.PublicKey, hdl, repo)

	eBus, err := initEBus(log, &cfg.Kafka, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ebus: %w", err)
	}

	return &App{
		Cfg:        cfg,
		Log:        log,
		Handler:    hdl,
		Service:    svc,
		DB:         db,
		RDB:        rdb,
		Mailer:     mlr,
		HTTPServer: httpServer,
		EBus:       eBus,
		GeoDB:      geo,
	}, nil
}

func MustNew(cfg *config.Config, log *zap.Logger) *App {
	app, err := New(cfg, log)
	if err != nil {
		panic(err)
	}
	return app
}

func (a *App) Run(ctx context.Context) error {
	errs := make(chan error, 1)
	defer close(errs)

	go func() {
		if err := a.HTTPServer.Run(); err != nil {
			errs <- err
		}
	}()

	go func() {
		a.EBus.OutboxPublisher.Run(ctx)
	}()

	go func() {
		a.EBus.InboxSubscriber.Run(ctx)
	}()

	if err := <-errs; err != nil {
		return err
	}

	return nil
}

func (a *App) Shutdown() error {
	a.DB.Close()
	a.Log.Debug("Database closed")

	err := apperrors.ErrShutdown

	if rdbErr := a.RDB.Close(); rdbErr != nil {
		err = fmt.Errorf("%w, failed to close RDB: %w", err, rdbErr)
	}

	a.Log.Debug("Redis closed")

	if srvErr := a.HTTPServer.Shutdown(); srvErr != nil {
		err = fmt.Errorf("%w, failed to shutdown http server: %w", err, srvErr)
	}

	a.Log.Debug("Http server shutdown")

	if geoErr := a.GeoDB.Close(); geoErr != nil {
		err = fmt.Errorf("%w, failed to close GeoDB: %w", err, &geoErr)
	}

	a.Log.Debug("GeoDB closed")

	if !errors.Is(err, apperrors.ErrShutdown) {
		return err
	}

	return nil
}

func initDB(cfg *config.Database) (postgres.Postgres, error) {
	postgresCfg := &postgres.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Name:     cfg.Name,
		SSLMode:  cfg.SSLMode,
		MaxConns: cfg.MaxConns,
		MinConns: cfg.MinConns,
		Migration: postgres.Migration{
			Path:      cfg.Migration.Path,
			AutoApply: cfg.Migration.AutoApply,
		},
	}

	db, err := postgres.New(postgresCfg)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initRedis(cfg *config.Redis) (redis.Redis, error) {
	redisCfg := &redis.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	rdb, err := redis.New(redisCfg)
	if err != nil {
		return nil, err
	}

	return rdb, nil
}

func initMailer(log *zap.Logger, cfg *config.Mailer) mailer.Mailer {
	mailerCfg := &mailer.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		From:     cfg.From,
		UseTLS:   cfg.UseTLS,
	}

	mlr := mailer.New(mailerCfg)
	log.Debug("Mailer initialized")
	return mlr
}

func initElastic(log *zap.Logger, cfg *config.Elastic) (elasticsearch.Elasticsearch, error) {
	elasticCfg := &elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
		Timeout:   cfg.Timeout,
	}

	client, err := elasticsearch.New(elasticCfg)
	if err != nil {
		return nil, err
	}

	log.Debug("Elasticsearch initialized")
	return client, nil
}

func initSecurity(log *zap.Logger, cfg config.Key) (*Security, error) {
	privateKey, err := jwt.LoadECDSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	log.Debug("Private key loaded")

	publicKey, err := jwt.LoadECDSAPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	log.Debug("Public key loaded")

	return &Security{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// ОБНОВИТЬ initHandler - добавить FAQHandler
func initHandler(log *zap.Logger, jwtCfg *config.JWT, svc *Service) *Handler {
	healthHandler := handler.NewHealthHandler(log, svc.HealthService)
	log.Debug("Health handler initialized")

	authHandler := handler.NewAuthHandler(log, svc.AuthService, jwtCfg.AccessTokenTTL, jwtCfg.RefreshTokenTTL)
	log.Debug("Auth handler initialized")

	userHandler := handler.NewUserHandler(svc.UserService)
	log.Debug("User handler initialized")

	articleHandler := handler.NewArticleHandler(svc.ArticleService)
	log.Debug("Article handler initialized")

	apiKeyHandler := handler.NewAPIKeyHandler(svc.APIKeyService)
	log.Debug("API Key handler initialized")

	// ДОБАВИТЬ FAQ handler
	faqHandler := handler.NewFAQHandler(svc.FAQService)
	log.Debug("FAQ handler initialized")

	requestHandler := handler.NewRequestHandler(log, svc.RequestService)
	log.Debug("Request handler initialized")

	return &Handler{
		RequestHandler: requestHandler,
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		ArticleHandler: articleHandler,
		APIKeyHandler:  apiKeyHandler,
		FAQHandler:     faqHandler, // ДОБАВИТЬ
	}
}

// ОБНОВИТЬ initService - добавить FAQService
func initService(
	log *zap.Logger,
	jwtCfg *config.JWT,
	sec *Security,
	repo *Repository,
	mlr mailer.Mailer,
	rdb redis.Redis,
	geoDB geoip.GeoIP,
) *Service {
	healthSvc := service.NewHealthService(log, repo.HealthRepository)
	log.Debug("Health service initialized")

	authSvc := service.NewAuthService(log, sec.PublicKey, sec.PrivateKey, repo.AuthRepository, repo.UserRepository, mlr, rdb, jwtCfg.AccessTokenTTL, jwtCfg.RefreshTokenTTL)
	log.Debug("Auth service initialized")

	userSvc := service.NewUserService(repo.UserRepository, mlr)
	log.Debug("User service initialized")

	requestSvc := service.NewRequestService(log, repo.RequestRepository, repo.OutboxRepository, repo.AgentRepository, geoDB)
	log.Debug("Request service initialized")

	articleSvc := service.NewArticleService(repo.ArticleRepository)
	log.Debug("Article service initialized")

	apiKeySvc := service.NewAPIKeyService(repo.APIKeyRepository)
	log.Debug("API Key service initialized")

	// ДОБАВИТЬ FAQ service
	faqSvc := service.NewFAQService(repo.FAQRepository)
	log.Debug("FAQ service initialized")

	return &Service{
		RequestService: requestSvc,
		HealthService:  healthSvc,
		AuthService:    authSvc,
		UserService:    userSvc,
		ArticleService: articleSvc,
		APIKeyService:  apiKeySvc,
		FAQService:     faqSvc, // ДОБАВИТЬ
	}
}

// ОБНОВИТЬ initRepository - добавить FAQRepository
func initRepository(log *zap.Logger, db postgres.Postgres, es elasticsearch.Elasticsearch) *Repository {
	healthRepo := repository.NewHealthRepository(db.Pool())
	log.Debug("Health repository initialized")

	authRepo := repository.NewAuthRepository(db.Pool())
	log.Debug("Auth repository initialized")

	userRepo := repository.NewUserRepository(db.Pool())
	log.Debug("User repository initialized")

	requestRepo := repository.NewRequestRepository(db.Pool())
	log.Debug("Request repository initialized")

	outboxRepo := repository.NewOutboxRepository(db.Pool())
	log.Debug("Outbox repository initialized")

	inboxRepo := repository.NewInboxRepository(db.Pool())
	log.Debug("Inbox repository initialized")

	agentRepo := repository.NewAgentRepository(db.Pool())
	log.Debug("Agent repository initialized")

	articleRepo := repository.NewElasticRepository(es.Client())
	log.Debug("Article repository initialized")

	apiKeyRepo := repository.NewAPIKeyRepository(db.Pool())
	log.Debug("Api key repository initialized")

	// ДОБАВИТЬ FAQ repository
	faqRepo := repository.NewFAQRepository(db.Pool())
	log.Debug("FAQ repository initialized")

	return &Repository{
		RequestRepository: requestRepo,
		InboxRepository:   inboxRepo,
		AgentRepository:   agentRepo,
		HealthRepository:  healthRepo,
		AuthRepository:    authRepo,
		UserRepository:    userRepo,
		OutboxRepository:  outboxRepo,
		ArticleRepository: articleRepo,
		APIKeyRepository:  apiKeyRepo,
		FAQRepository:     faqRepo, // ДОБАВИТЬ
	}
}

// ОБНОВИТЬ initHTTPServer - добавить FAQHandler в вызов SetupRouter
func initHTTPServer(log *zap.Logger, cfg *config.Config, publicKey *ecdsa.PublicKey, hdl *Handler, repo *Repository) server.HTTPServer {
	router := route.SetupRouter(
		log,
		cfg,
		publicKey,
		hdl.HealthHandler,
		hdl.AuthHandler,
		hdl.UserHandler,
		hdl.ArticleHandler,
		repo.APIKeyRepository,
		hdl.FAQHandler,
		hdl.RequestHandler,
	)

	httpServer := server.NewHTTPServer(
		server.WithAddr(cfg.HTTPServer.Host, cfg.HTTPServer.Port),
		server.WithTimeout(cfg.HTTPServer.Timeout.Read, cfg.HTTPServer.Timeout.Write, cfg.HTTPServer.Timeout.Idle),
		server.WithHandler(router),
	)

	return httpServer
}

func initEBus(log *zap.Logger, cfg *config.Kafka, repo *Repository) (*EBus, error) {
	producer, err := kafka.NewProducer(
		cfg.Brokers,
		kafka.WithBalancer(kafka.RoundRobin),
		kafka.WithRequiredAcks(kafka.RequireAll),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init kafka producer: %w", err)
	}

	log.Debug("Kafka producer initialized")

	outboxCfg := outbox.Config{
		Name:         cfg.Producer.Name,
		WorkerCount:  cfg.Producer.WorkerCount,
		PollInterval: cfg.Producer.PollInterval,
		BatchSize:    cfg.Producer.BatchSize,
	}

	publisher := outbox.NewPublisher(
		log,
		outboxCfg,
		producer,
		repo.OutboxRepository,
	)

	log.Debug("Outbox publisher initialized")

	consumerGroup, err := kafka.NewConsumerGroupRunner(
		cfg.Brokers,
		cfg.Subscriber.GroupID,
		[]string{cfg.Subscriber.Topic},
		consumerBufferSize,
		kafka.WithBalancerConsumer(kafka.RoundrobinBalanceStrategy),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	go func() {
		startAndRunningStr := <-consumerGroup.Info()

		log.Info(startAndRunningStr)
	}()

	inboxCfg := inbox.Config{
		Name:        cfg.Subscriber.Name,
		WorkerCount: cfg.Subscriber.WorkerCount,
		BatchSize:   cfg.Producer.BatchSize,
		Topic:       cfg.Subscriber.Topic,
	}

	subscriber := inbox.NewSubscriber(
		log,
		inboxCfg,
		consumerGroup,
		repo.InboxRepository,
		repo.RequestRepository,
	)

	return &EBus{
		OutboxPublisher: publisher,
		InboxSubscriber: subscriber,
	}, err
}

func initGeo(log *zap.Logger, cfg *config.Geo) (geoip.GeoIP, error) {
	geo, err := geoip.NewGeo(cfg.GeoLiteCountryPath, cfg.GeoLiteASNPath)
	if err != nil {
		return geo, fmt.Errorf("failed to init geoip: %w", err)
	}

	log.Debug("GeoIP initialized")

	return geo, nil
}
