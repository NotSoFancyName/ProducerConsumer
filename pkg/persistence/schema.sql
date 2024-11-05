CREATE TYPE task_state AS ENUM ('received', 'processing', 'done');

CREATE TABLE tasks (
       id BIGSERIAL PRIMARY KEY,
       type integer,
       value integer,
       state task_state,
       creation_time timestamp DEFAULT now(),
       last_update_time timestamp DEFAULT now()
);

CREATE INDEX idx_tasks_id ON tasks (id);
