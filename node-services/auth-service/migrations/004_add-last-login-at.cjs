exports.up = (pgm) => {
  pgm.addColumn("users", {
    last_login_at: {
      type: "timestamptz",
      notNull: false,
    },
  });
};

exports.down = (pgm) => {
  pgm.dropColumn("users", "last_login_at");
};
