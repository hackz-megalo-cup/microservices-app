exports.up = (pgm) => {
  pgm.createTable("outbox_events", {
    id: { type: "uuid", primaryKey: true },
    event_type: { type: "text", notNull: true },
    topic: { type: "text", notNull: true },
    payload: { type: "jsonb", notNull: true },
    created_at: { type: "timestamptz", notNull: true, default: pgm.func("NOW()") },
    published: { type: "boolean", notNull: true, default: false },
    published_at: { type: "timestamptz" },
  });
  pgm.createIndex("outbox_events", "created_at", {
    where: "published = FALSE",
    name: "idx_outbox_unpublished",
  });
};

exports.down = (pgm) => {
  pgm.dropTable("outbox_events");
};
