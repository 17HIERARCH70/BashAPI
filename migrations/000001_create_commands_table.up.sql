CREATE SCHEMA IF NOT EXISTS commands;

CREATE TABLE IF NOT EXISTS commands.commands (
                                                 id SERIAL PRIMARY KEY,
                                                 script TEXT NOT NULL,
                                                 output TEXT DEFAULT '',
                                                 status VARCHAR(50) NOT NULL DEFAULT 'pending',
                                                 pid INTEGER,
                                                 created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                                 updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION commands.update_updated_at_column()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_commands_updated_at
    BEFORE UPDATE ON commands.commands
    FOR EACH ROW EXECUTE FUNCTION commands.update_updated_at_column();
