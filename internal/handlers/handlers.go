package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/AbramovArseniy/SimpleVotes/internal/config"
	"github.com/AbramovArseniy/SimpleVotes/internal/storage"
	"github.com/AbramovArseniy/SimpleVotes/internal/storage/database"
	"github.com/AbramovArseniy/SimpleVotes/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
)

type Handler struct {
	Storage storage.Storage
	Auth    *Auth
}

func NewHandler(cfg config.Config) *Handler {

	db, err := database.NewDatabase(cfg.DBAddress)
	if err != nil {
		log.Println("error while creating database:", err)
	}
	return &Handler{
		Storage: db,
		Auth:    NewAuth(cfg.JWTSecret, db),
	}
}

func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var u types.User
	r.ParseForm()
	u.Login = r.PostForm.Get("login")
	u.Password = r.PostForm.Get("password")
	log.Println(u.Login, u.Password)

	if u.Login == "" || u.Password == "" {
		http.Error(w, "Missing username or password.", http.StatusBadRequest)
		return
	}
	_, err := h.Storage.GetUserByLogin(u.Login)
	if err == nil {
		http.Error(w, "this username is taken", http.StatusConflict)
		return
	}
	log.Println(errors.Is(err, storage.ErrNotFound))
	if errors.Is(err, storage.ErrNotFound) {
		if err := h.Storage.RegisterUser(u); err != nil {
			log.Println("error while inserting new user data:", err)
			http.Error(w, "cannot register user", http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		token := h.Auth.MakeToken(u.Login)

		http.SetCookie(w, &http.Cookie{
			HttpOnly: true,
			Expires:  time.Now().Add(7 * 24 * time.Hour),
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Name:     "jwt",
			Value:    token,
		})
		//http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	log.Println("error while getting users data:", err)
	http.Error(w, "cannot register user", http.StatusInternalServerError)
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u types.User
	r.ParseForm()
	u.Login = r.PostForm.Get("login")
	u.Password = r.PostForm.Get("password")
	log.Println(u.Login, u.Password)

	if u.Login == "" || u.Password == "" {
		http.Error(w, "Missing username or password.", http.StatusBadRequest)
		return
	}
	if err := h.Auth.CheckPassword(&u); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "no user with this username", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrWrongPassword) {
			http.Error(w, "wrong password", http.StatusBadRequest)
			return
		}
		http.Error(w, "cannot check if password is correct", http.StatusInternalServerError)
		return
	}
	token := h.Auth.MakeToken(u.Login)

	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Name:     "jwt",
		Value:    token,
	})

	//http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		MaxAge:   -1,
		Path:     "/user/login",
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    "",
	})
	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		MaxAge:   -1,
		Path:     "/user/register",
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    "",
	})

	//http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) PostQuestionHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("cannot parse question form:", err)
		http.Error(w, "cannot parse question form", http.StatusBadRequest)
		return
	}
	var q = types.Question{
		Text:    r.PostForm.Get("Question"),
		Type:    types.QuestionType(r.PostForm.Get("Question type")),
		Options: make([]string, 10),
	}
	cnt := 1
	for r.Form.Has("Option " + strconv.Itoa(cnt)) {
		option := r.PostForm.Get("Option " + strconv.Itoa(cnt))
		q.Options = append(q.Options, option)
	}
	err = h.Storage.SaveQuestion(q)
	if err != nil {
		log.Println("cannot save question to dataabse:", err)
		http.Error(w, "cannot save question to dataabse", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Route() chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(h.Auth.JWTAuth))
		r.Post("/add-question/", h.PostQuestionHandler)
	})

	r.Group(func(r chi.Router) {
		r.Post("/user/register/", h.RegisterHandler)
		r.Post("/user/login/", h.LoginHandler)
		r.Get("/user/logout/", h.LogoutHandler)
	})

	return r
}
