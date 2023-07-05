CREATE TABLE options(
    question_id INT,
    number INT,
    text VARCHAR(256),
    FOREIGN KEY(question_id) REFERENCES questions(id) ON DELETE CASCADE
)