exports.up = (pgm) => {
  pgm.createTable("user_pokemon", {
    user_id: {
      type: "uuid",
      notNull: true,
      references: '"users"(id)',
      onDelete: "CASCADE",
    },
    pokemon_id: {
      type: "varchar",
      notNull: true,
    },
    caught_at: {
      type: "timestamptz",
      notNull: true,
      default: pgm.func("NOW()"),
    },
  });

  pgm.addConstraint("user_pokemon", "user_pokemon_pk", {
    primaryKey: ["user_id", "pokemon_id"],
  });

  pgm.createIndex("user_pokemon", "user_id");
};

exports.down = (pgm) => {
  pgm.dropTable("user_pokemon");
};
