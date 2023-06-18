package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/AbramovArseniy/SimpleVotes/internal/storage"
	"github.com/AbramovArseniy/SimpleVotes/internal/types"
	"github.com/go-chi/jwtauth"
)

var (
	ErrWrongPassword = errors.New("wrong password")
	ErrNotAuthorized = errors.New("user is not authorized")
)

type Auth struct {
	JWTSecret   string
	JWTAuth     *jwtauth.JWTAuth
	UserStorage storage.Storage
}

func NewAuth(secret string, userStorage storage.Storage) *Auth {
	jwtAuth := jwtauth.New("HS256", []byte(secret), nil)
	return &Auth{
		JWTSecret:   secret,
		JWTAuth:     jwtAuth,
		UserStorage: userStorage,
	}
}

func (a *Auth) MakeToken(user types.User) string {
	_, tokenString, _ := a.JWTAuth.Encode(map[string]interface{}{"id": user.Id, "login": user.Login})
	return tokenString
}

func (a *Auth) CheckPassword(user *types.User) error {
	var userData types.User
	userData, err := a.UserStorage.GetUserByLogin(user.Login)
	if err != nil {
		return fmt.Errorf("error while getting user from database: %w", err)
	}
	if user.CheckPasswordHash(userData.Password) {
		user.Id = userData.Id
		return nil
	}
	return ErrWrongPassword
}

func (a *Auth) GetCurUserInfo(r *http.Request) (types.User, error) {
	jwtCookie, err := r.Cookie("jwt")
	if errors.Is(err, http.ErrNoCookie) {
		return types.User{}, ErrNotAuthorized
	}
	if err != nil {
		return types.User{}, fmt.Errorf("error while getting cookie: %w", err)
	}
	token, err := a.JWTAuth.Decode(jwtCookie.Value)
	if err != nil {
		return types.User{}, fmt.Errorf("error while decoding cookie: %w", err)
	}
	var ok bool
	id, ok := token.Get("id")
	if !ok {
		return types.User{}, ErrNotAuthorized
	}
	login, ok := token.Get("login")
	log.Println(id)
	if !ok {
		return types.User{}, ErrNotAuthorized
	}
	var u = types.User{
		Id:    int(id.(float64)),
		Login: login.(string),
	}
	return u, err
}
