DROP TABLE IF EXISTS commands.commands;
DROP FUNCTION IF EXISTS commands.update_updated_at_column;
DROP TRIGGER IF EXISTS update_commands_updated_at ON commands.commands;
DROP SCHEMA IF EXISTS commands;