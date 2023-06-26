package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/AbramovArseniy/SimpleVotes/internal/storage"
	"github.com/AbramovArseniy/SimpleVotes/internal/types"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	saveQuestionQuery       = `INSERT INTO questions (text, type, user_id) VALUES($1, $2, $3) RETURNING id`
	getQuestionsByUserQuery = `SELECT questions.id, questions.text as text, questions.type as type, questions.user_id, COALESCE(ans_cnt.cnt_usr,0)
								FROM questions
								LEFT JOIN  
								(SELECT answers.question_id as qid, COUNT(answers.user_id) as cnt_usr
								FROM answers 
								GROUP BY answers.question_id) AS ans_cnt
								ON ans_cnt.qid = questions.id
								WHERE user_id=$1
								ORDER BY ans_cnt.cnt_usr`
	getUserByLoginQuery       = `SELECT id, password FROM users WHERE login=$1`
	getQuestionByIdQuery      = `SELECT text, type, user_id FROM questions WHERE id=$1`
	getUserByIdQuery          = `SELECT login FROM users WHERE id=$1`
	getAnswersWithOptionQuery = `SELECT COUNT(*) FROM answers WHERE question_id=$1 AND option=$2`
	getOptionsByQuestionQuery = `SELECT text FROM options WHERE question_id = $1 ORDER BY number`
	getPopularQuestionsQuery  = `SELECT questions.id, questions.text as text, questions.type as type, questions.user_id, COALESCE(ans_cnt.cnt_usr,0)
								FROM questions
								LEFT JOIN  
								(SELECT answers.question_id as qid, COUNT(answers.user_id) as cnt_usr
								FROM answers 
								GROUP BY answers.question_id) AS ans_cnt
								ON ans_cnt.qid = questions.id
								ORDER BY ans_cnt.cnt_usr`
	getAllAnswersQuery    = `SELECT COUNT(*) FROM answers WHERE question_id=$1`
	saveAnswerQuery       = `INSERT INTO answers (question_id, option, user_id) VALUES($1, $2, $3)`
	registerUserQuery     = `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`
	saveOptionQuery       = `INSERT INTO options (question_id, number, text) VALUES ($1, $2, $3)`
	getAnswersByUserQuery = `SELECT option FROM answers WHERE question_id =$1 AND user_id=$2`
)

type Database struct {
	DB   *sql.DB
	Addr string
}

func NewDatabase(address string) (*Database, error) {
	db, err := sql.Open("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("error while opening db: %w", err)
	}
	database := &Database{
		DB:   db,
		Addr: address,
	}
	err = database.Migrate()
	if err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}
	return database, nil
}

func (db *Database) Migrate() error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://../internal/storage/database/migrations",
		db.Addr, driver)
	if err != nil {
		return fmt.Errorf("could not create migration: %w", err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (db *Database) GetOptions(questionID int) ([]string, error) {
	options := make([]string, 0)
	rows, err := db.DB.Query(getOptionsByQuestionQuery, questionID)
	if err != nil {
		return nil, fmt.Errorf("error while making sql query: %w", err)
	}
	for rows.Next() {
		var opt string
		err := rows.Scan(&opt)
		if err != nil {
			return nil, err
		}
		options = append(options, opt)
	}
	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return options, nil
}

func (db *Database) SaveQuestion(q types.Question) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	insertQuestionStmt, err := tx.Prepare(saveQuestionQuery)
	if err != nil {
		return err
	}
	err = insertQuestionStmt.QueryRow(q.Text, q.Type, q.UserID).Scan(&q.Id)
	if err != nil {
		return fmt.Errorf("error while making sql query: %w", err)
	}
	insertOptionsStmt, err := tx.Prepare(saveOptionQuery)
	if err != nil {
		return err
	}
	for i, option := range q.Options {
		_, err := insertOptionsStmt.Exec(q.Id, i+1, option)
		if err != nil {
			return err
		}
	}
	tx.Commit()
	return nil
}

func (db *Database) SaveAnswer(a types.Answer) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	insertAnswerStmt, err := tx.Prepare(saveAnswerQuery)
	if err != nil {
		return err
	}
	for _, opt := range a.Options {
		_, err = insertAnswerStmt.Exec(a.QuestionId, opt, a.UserID)
		if err != nil {
			return fmt.Errorf("error while making sql query: %w", err)
		}
	}
	tx.Commit()
	return nil
}

func (db *Database) GetPercentages(q types.Question) ([]int, error) {
	percentages := make([]int, len(q.Options))
	var totalAns int
	err := db.DB.QueryRow(getAllAnswersQuery, q.Id).Scan(&totalAns)
	if totalAns == 0 {
		for range q.Options {
			percentages = append(percentages, 0)
		}
		return percentages, nil
	}
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error while getting data from db: %w", err)
	}
	for i := range q.Options {
		var ans int
		err := db.DB.QueryRow(getAnswersWithOptionQuery, q.Id, i).Scan(&ans)
		if err == sql.ErrNoRows {
			percentages = append(percentages, 0)
		} else if err != nil {
			return nil, fmt.Errorf("error while getting data from db: %w", err)
		} else {
			percentages = append(percentages, (100*ans)/totalAns)
		}
	}
	return percentages, nil
}

func (db *Database) GetQuestionsByUser(user_id int) ([]types.Question, error) {
	var questions = make([]types.Question, 100)
	rows, err := db.DB.Query(getQuestionsByUserQuery, user_id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while making query to database: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var userCnt int
		var q types.Question
		err := rows.Scan(&q.Id, &q.Text, &q.Type, &q.UserID, &userCnt)
		if err != nil {
			return nil, err
		}
		q.Options, err = db.GetOptions(q.Id)
		if err != nil {
			return nil, fmt.Errorf("error while getting options: %w", err)
		}
		q.Percentages, err = db.GetPercentages(q)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return questions, nil
}

func (db *Database) GetPopularQuestions() ([]types.Question, error) {
	var questions []types.Question
	rows, err := db.DB.Query(getPopularQuestionsQuery)
	if err != nil {
		return questions, fmt.Errorf("error while doing query to database: %w", err)
	}
	for rows.Next() {
		var q types.Question
		var userCnt int
		err := rows.Scan(&q.Id, &q.Text, &q.Type, &q.UserID, &userCnt)
		if err != nil {
			return nil, err
		}
		q.Options, err = db.GetOptions(q.Id)
		if err != nil {
			return nil, fmt.Errorf("error while getting options: %w", err)
		}
		q.Percentages, err = db.GetPercentages(q)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)

	}
	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return questions, nil
}

func (db *Database) GetQuestion(id int) (types.Question, error) {
	var q types.Question
	err := db.DB.QueryRow(getQuestionByIdQuery, id).Scan(&q.Text, &q.Type, &q.UserID)
	if err == sql.ErrNoRows {
		return q, storage.ErrNotFound
	}
	if err != nil {
		return q, fmt.Errorf("error while doing query to database: %w", err)
	}
	q.Options, err = db.GetOptions(id)
	if err != nil {
		return q, fmt.Errorf("error while getting options: %w", err)
	}
	return q, nil
}

func (db *Database) RegisterUser(u *types.User) error {
	if err := u.GeneratePasswordHash(); err != nil {
		return fmt.Errorf("cannot generate password hash: %w", err)
	}
	err := db.DB.QueryRow(registerUserQuery, u.Login, u.Password).Scan(&u.Id)
	if err != nil {
		return fmt.Errorf("error while making sql query question: %w", err)
	}
	return nil
}

func (db *Database) GetAnswered(qid, userId int) ([]int, error) {
	options := make([]int, 0)
	rows, err := db.DB.Query(getAnswersByUserQuery, qid, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var option int
		err = rows.Scan(&option)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return options, nil
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return options, nil
}

func (db *Database) GetUserByLogin(login string) (types.User, error) {
	var user types.User
	err := db.DB.QueryRow(getUserByLoginQuery, login).Scan(&user.Id, &user.Password)
	if err == sql.ErrNoRows {
		return user, storage.ErrNotFound
	}
	if err != nil {
		return user, fmt.Errorf("error while doing query to database: %w", err)
	}
	return user, nil
}

func (db *Database) GetUserById(id int) (types.User, error) {
	var user types.User
	err := db.DB.QueryRow(getUserByLoginQuery, id).Scan(&user.Login)
	if err == sql.ErrNoRows {
		return user, storage.ErrNotFound
	}
	if err != nil {
		return user, fmt.Errorf("error while doing query to database: %w", err)
	}
	return user, nil
}

func (db *Database) Close() {
	db.DB.Close()
}
