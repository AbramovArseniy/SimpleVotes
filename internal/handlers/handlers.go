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
	"github.com/AbramovArseniy/SimpleVotes/internal/templates"
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
	userData, err := h.Storage.GetUserByLogin(u.Login)
	if err == nil {
		http.Error(w, "this username is taken", http.StatusConflict)
		return
	}
	log.Println(errors.Is(err, storage.ErrNotFound))
	if errors.Is(err, storage.ErrNotFound) {
		if err := h.Storage.RegisterUser(&u); err != nil {
			log.Println("error while inserting new user data:", err)
			http.Error(w, "cannot register user", http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		token := h.Auth.MakeToken(userData)
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
	token := h.Auth.MakeToken(u)

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
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    "",
	})

	//http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) GetQuestionHandler(w http.ResponseWriter, r *http.Request) {
	qid, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println("cannot get question id from url")
		http.Error(w, "cannot get question id from url", http.StatusInternalServerError)
		return
	}
	q, err := h.Storage.GetQuestion(qid)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "no such question", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("error while getting question:", err)
		http.Error(w, "cannot get data from database", http.StatusInternalServerError)
		return
	}
	log.Println(q)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetPopularQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	var questions []types.Question
	questions, err := h.Storage.GetPopularQuestions()
	if errors.Is(err, storage.ErrNotFound) {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		log.Println("error while getting popular questions:", err)
		http.Error(w, "cannot get data from database", http.StatusInternalServerError)
		return
	}
	log.Println(questions)
	w.WriteHeader(http.StatusOK)
	templates.IndexTemplate.Execute(w, nil)
}

func (h *Handler) GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println("cannot get question id from url")
		http.Error(w, "cannot get question id from url", http.StatusInternalServerError)
		return
	}
	user, err := h.Storage.GetUserById(userID)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "no such user", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("error while getting popular questions:", err)
		http.Error(w, "cannot get data from database", http.StatusInternalServerError)
		return
	}
	var questions []types.Question
	questions, err = h.Storage.GetQuestionsByUser(userID)
	if errors.Is(err, storage.ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		log.Println("error while getting popular questions:", err)
		http.Error(w, "cannot get data from database", http.StatusInternalServerError)
		return
	}
	log.Println("user:", user)
	log.Println("questions:", questions)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PostQuestionHandler(w http.ResponseWriter, r *http.Request) {
	curUser, err := h.Auth.GetCurUserInfo(r)
	if err != nil {
		log.Println("cannot get current user's data:", err)
		http.Error(w, "cannot get logged in user data", http.StatusUnauthorized)
		return
	}
	err = r.ParseForm()
	if err != nil {
		log.Println("cannot parse question form:", err)
		http.Error(w, "cannot parse question form", http.StatusBadRequest)
		return
	}
	log.Println(curUser.Id, curUser.Login)
	var q = types.Question{
		Text:    r.PostForm.Get("Question"),
		Type:    types.QuestionType(r.PostForm.Get("Question type")),
		Options: make([]string, 0),
		UserID:  curUser.Id,
	}
	cnt := 1
	for r.PostForm.Has("Option " + strconv.Itoa(cnt)) {
		option := r.PostForm.Get("Option " + strconv.Itoa(cnt))
		q.Options = append(q.Options, option)
		cnt++
	}
	err = h.Storage.SaveQuestion(q)
	if err != nil {
		log.Println("cannot save question to database:", err)
		http.Error(w, "cannot save question to dataabse", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PostAnswerHandler(w http.ResponseWriter, r *http.Request) {
	curUser, err := h.Auth.GetCurUserInfo(r)
	if err != nil {
		log.Println("cannot get current user's data:", err)
		http.Error(w, "cannot get logged in user data", http.StatusUnauthorized)
		return
	}
	qid, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println("cannot get question id from url")
		http.Error(w, "cannot get question id from url", http.StatusInternalServerError)
		return
	}
	r.ParseForm()
	var ans = types.Answer{
		QuestionId: qid,
		Options:    make([]int, 0),
		UserID:     curUser.Id,
	}
	q, err := h.Storage.GetQuestion(qid)
	if err != nil {
		log.Println("cannot get question data from database")
		http.Error(w, "cannot get question data from database", http.StatusInternalServerError)
		return
	}
	for i, opt := range q.Options {
		if r.PostForm.Has(opt) {
			ans.Options = append(ans.Options, i)
		}
	}
	err = h.Storage.SaveAnswer(ans)
	if err != nil {
		log.Println("cannot save answer to database")
		http.Error(w, "cannot save answer to database", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	//http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Route() chi.Router {
	r := chi.NewRouter()
	fs := http.FileServer(http.Dir("../internal/static/"))
	r.Handle("/static/", http.StripPrefix("/static/", fs))
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(h.Auth.JWTAuth))
		r.Post("/add-question/", h.PostQuestionHandler)
		r.Post("/question/{id}", h.PostAnswerHandler)
		r.Get("/question/{id}", h.GetQuestionHandler)
		r.Get("/user/{id}/", h.GetUserProfileHandler)
		r.Get("/", h.GetPopularQuestionsHandler)
	})

	r.Group(func(r chi.Router) {
		r.Post("/user/register/", h.RegisterHandler)
		r.Post("/user/login/", h.LoginHandler)
		r.Get("/user/logout/", h.LogoutHandler)
	})

	return r
}
