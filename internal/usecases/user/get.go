package user

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/cache"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// GetUseCase implementa o caso de uso de buscar user por ID.
type GetUseCase struct {
	repo   interfaces.Repository
	cache  interfaces.Cache   // optional, set via WithCache()
	flight *cache.FlightGroup // optional, prevents cache stampede
}

// NewGetUseCase cria uma nova instância do GetUseCase.
func NewGetUseCase(repo interfaces.Repository) *GetUseCase {
	return &GetUseCase{
		repo: repo,
	}
}

// WithCache sets an optional cache for the use case (builder pattern).
func (uc *GetUseCase) WithCache(c interfaces.Cache) *GetUseCase {
	uc.cache = c
	return uc
}

// WithFlight adds singleflight protection against cache stampede (thundering herd).
func (uc *GetUseCase) WithFlight(fg *cache.FlightGroup) *GetUseCase {
	uc.flight = fg
	return uc
}

// Execute busca um user pelo ID.
//
// Fluxo com cache:
//  1. Tenta buscar no cache
//  2. Se cache miss, busca no DB
//  3. Armazena no cache para próximas requisições
func (uc *GetUseCase) Execute(ctx context.Context, input dto.GetInput) (*dto.GetOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.User.Get")
	defer span.End()

	ctx = injectLogContext(ctx, "get")

	// Validar e converter ID
	id, parseErr := vo.ParseID(input.ID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "user get failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.ID))
	cacheKey := "user:" + input.ID

	// 1. Tentar cache primeiro
	if uc.cache != nil {
		var cached dto.GetOutput
		if cacheErr := uc.cache.Get(ctx, cacheKey, &cached); cacheErr == nil {
			span.SetAttributes(attribute.Bool("cache.hit", true))
			logutil.LogInfo(ctx, "cache hit", "key", cacheKey)
			return &cached, nil
		}
	}

	// 2. Buscar no repositório (cache miss — with singleflight if configured)
	var e *userdomain.User

	if uc.flight != nil {
		val, flightErr, _ := uc.flight.Do(input.ID, func() (any, error) {
			return uc.repo.FindByID(ctx, id)
		})
		if flightErr != nil {
			span.SetStatus(otelcodes.Error, flightErr.Error())
			logutil.LogWarn(ctx, "user get failed", "error", flightErr.Error())
			return nil, flightErr
		}
		e = val.(*userdomain.User)
	} else {
		var findErr error
		e, findErr = uc.repo.FindByID(ctx, id)
		if findErr != nil {
			span.SetStatus(otelcodes.Error, findErr.Error())
			logutil.LogWarn(ctx, "user get failed", "error", findErr.Error())
			return nil, findErr
		}
	}

	// 3. Converter para DTO de saída
	output := &dto.GetOutput{
		ID:        e.ID.String(),
		Name:      e.Name,
		Email:     e.Email.String(),
		Active:    e.Active,
		CreatedAt: e.CreatedAt.Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}

	// 4. Armazenar no cache
	if uc.cache != nil {
		if setCacheErr := uc.cache.Set(ctx, cacheKey, output); setCacheErr != nil {
			logutil.LogWarn(ctx, "failed to cache user", "key", cacheKey, "error", setCacheErr.Error())
		}
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))
	logutil.LogInfo(ctx, "user retrieved", "user.id", e.ID.String())

	return output, nil
}
