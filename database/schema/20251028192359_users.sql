-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    email text not null unique
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
