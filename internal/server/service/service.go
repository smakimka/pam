// Пакет service отвечает за логику обработки сообщений и содержит все методы, обрабатывающие grpc запросы
package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/smakimka/pam/internal/protobuf/pamserver"
	"github.com/smakimka/pam/internal/server/model"
	"github.com/smakimka/pam/internal/server/storage"
)

type PamService struct {
	pamserver.UnimplementedPamServerServer
	s          storage.Storage
	expiryTime int
}

func newPamSerice(s storage.Storage, expiryTime int) *PamService {
	return &PamService{s: s, expiryTime: expiryTime}
}

func (p *PamService) prolongToken(ctx context.Context, token string) error {
	now := time.Now()

	if err := p.s.UpdateTokenExpiry(ctx, token, now.Add(time.Duration(p.expiryTime)*time.Second)); err != nil {
		return err
	}

	return nil
}

func (p *PamService) createToken(ctx context.Context, userID int) (*model.TokenData, error) {
	token := &model.TokenData{}

	tokenValue, err := uuid.NewRandom()
	if err != nil {
		return token, err
	}
	expiry := time.Now().Add(time.Duration(time.Duration(p.expiryTime) * time.Second))

	tokenID, err := p.s.CreateAuthToken(ctx, userID, tokenValue.String(), expiry)
	if err != nil {
		return token, err
	}

	token.ID = tokenID
	token.Value = tokenValue.String()

	return token, err
}

// Register Отвечает за регистрацию пользователей, возвращает токен авторизации
func (p *PamService) Register(ctx context.Context, in *pamserver.AuthData) (*pamserver.AuthResponse, error) {
	log.Info().Msg("got a regiter request")
	resp := &pamserver.AuthResponse{}

	pwdHash, err := bcrypt.GenerateFromPassword([]byte(in.Pwd), bcrypt.DefaultCost)
	if err != nil {
		return resp, err
	}

	id, err := p.s.CreateUser(ctx, in.Username, pwdHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				resp.Error = "user already exists"
				log.Info().Msg("user already exists, returning error")
				return resp, nil
			}
		}

		log.Err(err).Msg("error creating user")
		return resp, status.Error(codes.Internal, err.Error())
	}

	token, err := p.createToken(ctx, id)
	if err != nil {
		return resp, status.Error(codes.Internal, err.Error())
	}

	log.Info().Msgf("new user id: %d", id)
	return &pamserver.AuthResponse{Token: token.Value}, nil
}

// Authenticate Отечает за авторизацию, возвращает токен авторизации
func (p *PamService) Authenticate(ctx context.Context, in *pamserver.AuthData) (*pamserver.AuthResponse, error) {
	log.Info().Msg("got an auth request")
	resp := &pamserver.AuthResponse{}

	user, err := p.s.GetUser(ctx, in.Username)
	if err != nil {
		return resp, status.Error(codes.NotFound, "wrong username or password")
	}

	if err = bcrypt.CompareHashAndPassword(user.Pwd, []byte(in.Pwd)); err != nil {
		return resp, status.Error(codes.NotFound, "wrong username or password")
	}

	token, err := p.createToken(ctx, user.ID)
	if err != nil {
		return resp, status.Error(codes.Internal, err.Error())
	}

	return &pamserver.AuthResponse{Token: token.Value}, nil
}

// Upload Отвечает за загрузку данных на сервер, name является уникальным параметром и идентифицирует данные.
// При повторном вызове с тем же именем данные будут перезаписаны
func (p *PamService) Upload(ctx context.Context, in *pamserver.UploadData) (*pamserver.UploadResponse, error) {
	log.Info().Msg("got upload request")
	resp := &pamserver.UploadResponse{}

	userID, ok := ctx.Value(model.UserID).(int)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}
	authToken, ok := ctx.Value(model.AuthToken).(string)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}

	err := p.prolongToken(ctx, authToken)
	if err != nil {
		return resp, status.Error(codes.Internal, "error prolonging token")
	}

	_, err = p.s.UpsertData(ctx, userID, in.Name, int(in.Type), in.Data)
	if err != nil {
		return resp, status.Error(codes.Internal, "error upserting data")
	}

	return resp, nil
}

// Get Отвечает за получение данных по имени, нужна авторизация
func (p *PamService) Get(ctx context.Context, in *pamserver.GetData) (*pamserver.GetDataResponse, error) {
	log.Info().Msg("got get data request")
	resp := &pamserver.GetDataResponse{}

	userID, ok := ctx.Value(model.UserID).(int)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}
	authToken, ok := ctx.Value(model.AuthToken).(string)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}

	err := p.prolongToken(ctx, authToken)
	if err != nil {
		return resp, status.Error(codes.Internal, "error prolonging token")
	}

	data, err := p.s.GetData(ctx, userID, in.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return resp, status.Error(codes.NotFound, "this data does not exist")
		}

		return resp, status.Error(codes.Internal, "internal error")
	}

	resp.Kind = int32(data.Kind)
	resp.Data = data.Bytes

	return resp, nil
}

// GetNames Отвечает за получение имен всех сохраненных данных пользователя, нужна авторизация
func (p *PamService) GetNames(ctx context.Context, in *pamserver.GetDataNames) (*pamserver.GetDataNamesResponse, error) {
	log.Info().Msg("got get data names request")
	resp := &pamserver.GetDataNamesResponse{}

	userID, ok := ctx.Value(model.UserID).(int)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}
	authToken, ok := ctx.Value(model.AuthToken).(string)
	if !ok {
		return resp, status.Error(codes.Internal, "internal ctx error")
	}

	err := p.prolongToken(ctx, authToken)
	if err != nil {
		return resp, status.Error(codes.Internal, "error prolonging token")
	}

	data, err := p.s.GetDataNames(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return resp, status.Error(codes.NotFound, "this data does not exist")
		}

		return resp, status.Error(codes.Internal, "internal error")
	}

	resp.Names = data

	return resp, nil
}
