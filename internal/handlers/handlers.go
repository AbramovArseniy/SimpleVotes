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

func (h *Handler) AuthVerifier(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := h.Auth.GetCurUserInfo(r)
		if err != nil {
			h.UnautorizedHandler(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) UnautorizedHandler(w http.ResponseWriter, r *http.Request) {
	templates.UnauthorizedTemplate.Execute(w, nil)
}

func (h *Handler) AuthFormHandler(w http.ResponseWriter, r *http.Request) {
	templates.AuthTemplate.Execute(w, nil)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

func (h *Handler) GetAddQuestionFormHandler(w http.ResponseWriter, r *http.Request) {
	templates.AddQuestionTemplate.Execute(w, nil)
}

func (h *Handler) GetPopularQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	curUser, err := h.Auth.GetCurUserInfo(r)
	if err != nil {
		log.Println("cannot get current user's data:", err)
		http.Error(w, "cannot get logged in user data", http.StatusUnauthorized)
		return
	}
	var data types.PopularQuestionsPageData
	data.User = curUser
	var questions []types.Question
	questions, err = h.Storage.GetPopularQuestions()
	if errors.Is(err, storage.ErrNotFound) {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		log.Println("error while getting popular questions:", err)
		http.Error(w, "cannot get data from database", http.StatusInternalServerError)
		return
	}
	for i, q := range questions {
		questions[i].Answered, err = h.Storage.GetAnswered(q.Id, curUser.Id)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		questions[i].IsAnswered = len(questions[i].Answered) > 0
	}
	data.Questions = questions
	w.WriteHeader(http.StatusOK)
	templates.IndexTemplate.Execute(w, data)
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
		Text:    r.PostForm.Get("question"),
		Type:    types.QuestionType(r.PostForm.Get("type")),
		Options: make([]string, 0),
		UserID:  curUser.Id,
	}
	optionNum, err := strconv.Atoi(r.PostForm.Get("option-number"))
	if err != nil {
		log.Println("cannot get option number from form:", err)
		http.Error(w, "cannot get option number from form", http.StatusInternalServerError)
		return
	}
	for i := 0; i < optionNum; i++ {
		option := r.PostForm.Get("option" + strconv.Itoa(i))
		q.Options = append(q.Options, option)
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
	r.ParseForm()
	log.Println(r.PostForm)
	qid, err := strconv.Atoi(r.PostForm.Get("qid"))
	if err != nil {
		log.Println("cannot get question id from form:", err)
		http.Error(w, "cannot get question id from form", http.StatusInternalServerError)
		return
	}
	var ans = types.Answer{
		QuestionId: qid,
		Options:    make([]int, 0),
		UserID:     curUser.Id,
	}
	q, err := h.Storage.GetQuestion(qid)
	if err != nil {
		log.Println("cannot get question data from database:", err)
		http.Error(w, "cannot get question data from database", http.StatusInternalServerError)
		return
	}
	log.Println(q.Type, q.Options)
	if q.Type == types.OneOptionType {
		if r.PostForm.Has("option") {
			opt, err := strconv.Atoi(r.PostForm.Get("option"))
			if err != nil {
				log.Println("cannot get answer from form:", err)
				http.Error(w, "cannot get answer from form", http.StatusInternalServerError)
				return
			}
			ans.Options = append(ans.Options, opt)
		}
	} else {
		for i := range q.Options {
			if r.PostForm.Has("option " + strconv.Itoa(i)) {
				ans.Options = append(ans.Options, i)
			}
		}
	}
	err = h.Storage.SaveAnswer(ans)
	if err != nil {
		log.Println("cannot save answer to database:", err)
		http.Error(w, "cannot save answer to database", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	//http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Route() chi.Router {
	r := chi.NewRouter()
	fs := http.FileServer(http.Dir("../internal/static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))
	r.Group(func(r chi.Router) {
		r.Use(h.AuthVerifier)
		r.Post("/add-question/", h.PostQuestionHandler)
		r.Get("/add-question/", h.GetAddQuestionFormHandler)
		r.Post("/add-answer/", h.PostAnswerHandler)
		r.Get("/question/{id}", h.GetQuestionHandler)
		r.Get("/user/{id}/", h.GetUserProfileHandler)
		r.Get("/", h.GetPopularQuestionsHandler)
	})

	r.Group(func(r chi.Router) {
		r.Post("/user/register/", h.RegisterHandler)
		r.Post("/user/login/", h.LoginHandler)
		r.Get("/user/register/", h.AuthFormHandler)
		r.Get("/user/login/", h.AuthFormHandler)
		r.Get("/user/unauthorized/", h.UnautorizedHandler)
		r.Get("/user/logout/", h.LogoutHandler)
	})

	return r
}
