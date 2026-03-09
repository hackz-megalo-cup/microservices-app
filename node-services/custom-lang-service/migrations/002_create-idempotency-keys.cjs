exports.up = (pgm) => {
  pgm.createTable('idempotency_keys', {
    key: { type: 'text', primaryKey: true },
    response: { type: 'text' },
    status_code: { type: 'integer', notNull: true, default: 0 },
    created_at: { type: 'timestamptz', notNull: true, default: pgm.func('now()') },
    expires_at: {
      type: 'timestamptz',
      notNull: true,
      default: pgm.func("now() + INTERVAL '24 hours'"),
    },
  });
  pgm.createIndex('idempotency_keys', 'expires_at', {
    name: 'idx_idempotency_keys_expires',
  });
};

exports.down = (pgm) => {
  pgm.dropTable('idempotency_keys');
};
