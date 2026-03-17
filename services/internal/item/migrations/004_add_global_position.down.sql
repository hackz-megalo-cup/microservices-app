DROP INDEX IF EXISTS idx_event_store_global_position;
ALTER TABLE event_store DROP COLUMN IF EXISTS global_position;
