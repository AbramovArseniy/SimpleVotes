package handlers

import (
	"errors"
	"fmt"
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

func (a *Auth) MakeToken(name string) string {
	_, tokenString, _ := a.JWTAuth.Encode(map[string]interface{}{"login": name})
	return tokenString
}

func (a *Auth) CheckPassword(user *types.User) error {
	var userData types.User
	userData, err := a.UserStorage.GetUserByLogin(user.Login)
	if err != nil {
		return fmt.Errorf("error while getting user from database: %w", err)
	}
	if user.CheckPasswordHash(userData.Password) {
		return nil
	}
	return ErrWrongPassword
}

func (a *Auth) GetUserLogin(r *http.Request) (string, error) {
	jwtCookie, err := r.Cookie("jwt")
	if errors.Is(err, http.ErrNoCookie) {
		return "", ErrNotAuthorized
	}
	if err != nil {
		return "", fmt.Errorf("error while getting cookie: %w", err)
	}
	token, err := a.JWTAuth.Decode(jwtCookie.Value)
	if err != nil {
		return "", fmt.Errorf("error while decoding cookie: %w", err)
	}
	loginInterface, ok := token.Get("login")
	if !ok {
		return "", ErrNotAuthorized
	}
	return loginInterface.(string), nil
}
