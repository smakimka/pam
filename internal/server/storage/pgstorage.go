package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/smakimka/pam/internal/server/model"
)

var ErrNoActiveToken = errors.New("no active token")

type PGStorage struct {
	p *pgxpool.Pool
}

func NewPGStorage(p *pgxpool.Pool) (*PGStorage, error) {
	s := &PGStorage{
		p: p,
	}

	err := s.p.Ping(context.Background())
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *PGStorage) Init(ctx context.Context) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `create table if not exists users (
        id serial primary key,
        username text,
        pwd bytea,
        constraint c_username_uq unique (username)
    )`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `create table if not exists auths (
        id serial primary key,
        user_id int references users(id),
        token text,
        creation_timestamp timestamp default current_timestamp,
        expiry_timestamp timestamp,
        constraint c_token_uq unique (token)
    )`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `create table if not exists user_data (
        id serial primary key,
        user_id int references users(id),
        name text,
        type int,
        data bytea,
        constraint c_name_uq unique (user_id, name)
    )`)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) ClearDBAfterTestDoNotUse(ctx context.Context) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `delete from user_data`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `delete from auths`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `delete from users`)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) GetUser(ctx context.Context, username string) (*model.UserData, error) {
	user := &model.UserData{}

	row := s.p.QueryRow(ctx, `select id, username, pwd from users where username like $1`, username)
	if err := row.Scan(&user.ID, &user.Username, &user.Pwd); err != nil {
		return user, err
	}

	return user, nil
}

func (s *PGStorage) CreateUser(ctx context.Context, username string, pwd []byte) (int, error) {
	var newUserID int

	tx, err := s.p.Begin(ctx)
	if err != nil {
		return newUserID, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `insert into users as u (username, pwd) values ($1, $2) returning u.id`, username, pwd)
	if err = row.Scan(&newUserID); err != nil {
		return newUserID, err
	}

	if err = tx.Commit(ctx); err != nil {
		return newUserID, err
	}

	return newUserID, nil
}

func (s *PGStorage) GetUserByToken(ctx context.Context, token string, now time.Time) (*model.UserData, error) {
	userData := &model.UserData{}

	row := s.p.QueryRow(ctx, `select u.id, u.username from users as u 
    join auths as a on a.user_id = u.id
    where a.token = $1 and a.expiry_timestamp >= $2`, token, now)
	if err := row.Scan(&userData.ID, &userData.Username); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info().Msg("no active token found")
			return userData, ErrNoActiveToken
		}

		return userData, err
	}

	return userData, nil
}

func (s *PGStorage) CreateAuthToken(ctx context.Context, userID int, value string, expiry time.Time) (int, error) {
	var newTokenID int

	tx, err := s.p.Begin(ctx)
	if err != nil {
		return newTokenID, err
	}
	defer tx.Rollback(ctx)

	row := s.p.QueryRow(ctx, `insert into auths as a (user_id, token, expiry_timestamp) values ($1, $2, $3) returning a.id`, userID, value, expiry)
	if err = row.Scan(&newTokenID); err != nil {
		return newTokenID, err
	}

	if err = tx.Commit(ctx); err != nil {
		return newTokenID, err
	}

	return newTokenID, err
}

func (s *PGStorage) UpdateTokenExpiry(ctx context.Context, token string, newExpiry time.Time) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `update auths set expiry_timestamp = $1 where token = $2`, newExpiry, token)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) UpsertData(ctx context.Context, userID int, name string, kind int, data []byte) (int, error) {
	var DataID int

	tx, err := s.p.Begin(ctx)
	if err != nil {
		return DataID, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `insert into user_data as ud (user_id, name, type, data)
    values ($1, $2, $3, $4) on conflict on constraint c_name_uq do 
    update set type = $3, data = $4 returning ud.id`, userID, name, kind, data)
	if err = row.Scan(&DataID); err != nil {
		return DataID, err
	}

	if err = tx.Commit(ctx); err != nil {
		return DataID, err
	}

	return DataID, err
}

func (s *PGStorage) GetData(ctx context.Context, userID int, name string) (*model.Data, error) {
	data := &model.Data{UserID: userID, Name: name}

	row := s.p.QueryRow(ctx, `select id, type, data from user_data where user_id = $1 and name = $2`, userID, name)
	if err := row.Scan(&data.ID, &data.Kind, &data.Bytes); err != nil {
		return data, err
	}

	return data, nil
}

func (s *PGStorage) GetDataNames(ctx context.Context, userID int) ([]string, error) {
	res := []string{}

	rows, err := s.p.Query(ctx, `select name from user_data where user_id = $1`, userID)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string

		err = rows.Scan(&name)
		if err != nil {
			return res, err
		}

		res = append(res, name)
	}

	if rows.Err() != nil {
		return res, rows.Err()
	}

	return res, nil
}
