CREATE TABLE questions(
    id SERIAL UNIQUE
    text VARCHAR(512)
    options VARCHAR(512)[]
    users_id INT
    PRIMARY KEY(id)
    FOREIGN KEY(user_id) REFERENCES(users(id)) ON DELETE NO ACTION
)