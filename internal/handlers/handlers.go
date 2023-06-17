package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/AbramovArseniy/SimpleVotes/internal/storage"
	"github.com/AbramovArseniy/SimpleVotes/internal/types"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Storage storage.Storage
}

func (h *Handler) PostQuestionHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("cannot parse question form:", err)
		http.Error(w, "cannot parse question form", http.StatusBadRequest)
		return
	}
	var q = types.Question{
		Text:    r.FormValue("Question"),
		Type:    types.QuestionType(r.FormValue("Question type")),
		Options: make([]string, 10),
	}
	cnt := 1
	for r.Form.Has("Option " + strconv.Itoa(cnt)) {
		option := r.Form.Get("Option " + strconv.Itoa(cnt))
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
	r.Post("/add-question/", h.PostQuestionHandler)
	return r
}
