-- This should be in the 000002_create_queue_table.up.sql file
CREATE TABLE IF NOT EXISTS commands.queue (
                                              queue_id SERIAL PRIMARY KEY,
                                              command_id INTEGER NOT NULL,
                                              status VARCHAR(50) NOT NULL DEFAULT 'waiting',
                                              FOREIGN KEY (command_id) REFERENCES commands.commands(id)
);