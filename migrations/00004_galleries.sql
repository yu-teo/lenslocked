-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS galleries (
    id SERIAL PRIMARY KEY,
    user_id INT,
    title TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE galleries;
-- +goose StatementEnd