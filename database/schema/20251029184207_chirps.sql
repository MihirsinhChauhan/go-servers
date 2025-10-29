-- +goose Up
-- +goose StatementBegin
CREATE TABLE chirps (
    id UUID primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    body text not null,
    user_id UUID not null,

    CONSTRAINT fk_chirps_users 
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE

)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE chirps
-- +goose StatementEnd
