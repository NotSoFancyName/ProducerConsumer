-- name: CreateTask :one
INSERT INTO tasks (
    type, value, state
) VALUES (
    $1,$2,$3
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: UpdateTask :exec
UPDATE tasks
SET state = $1, last_update_time = NOW()
WHERE id = $2;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

