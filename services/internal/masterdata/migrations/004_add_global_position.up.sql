ALTER TABLE event_store ADD COLUMN global_position BIGSERIAL;
CREATE UNIQUE INDEX idx_event_store_global_position ON event_store (global_position);
