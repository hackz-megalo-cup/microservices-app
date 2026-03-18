-- Seed Pokemon (Programming Languages)
INSERT INTO pokemon (id, name, type, hp, attack, speed, special_move_name, special_move_damage) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Go',         'procedural', 85, 70, 90, 'ゴルーチン乱舞',   760),
    ('00000000-0000-0000-0000-000000000002', 'Python',     'dynamic',    90, 65, 60, 'インデント地獄',   650),
    ('00000000-0000-0000-0000-000000000003', 'Rust',       'static',     95, 85, 80, '所有権爆砕',       1100),
    ('00000000-0000-0000-0000-000000000004', 'moonbit',    'functional', 70, 80, 95, 'モナド崩壊',       950),
    ('00000000-0000-0000-0000-000000000005', 'PHP',        'dynamic',    75, 65, 70, 'ドル記号嵐',       700),
    ('00000000-0000-0000-0000-000000000006', 'Swift',      'static',     85, 85, 85, 'プロトコル嵐',     840),
    ('00000000-0000-0000-0000-000000000007', 'TypeScript', 'static',     75, 80, 90, '型体操極意',       800),
    ('00000000-0000-0000-0000-000000000008', 'Java',       'static',     90, 75, 65, 'NullPointer処刑',  880),
    ('00000000-0000-0000-0000-000000000009', 'Whitespace', 'functional', 50, 50, 100, '虚空の一撃',      1300);

-- Seed Type Matchups
INSERT INTO type_matchup (attacking_type, defending_type, effectiveness) VALUES
    -- Static type advantages
    ('static', 'dynamic', 1.25),
    ('static', 'procedural', 1.0),
    ('static', 'functional', 1.0),
    ('static', 'static', 1.0),

    -- Dynamic type advantages
    ('dynamic', 'static', 0.8),
    ('dynamic', 'procedural', 1.0),
    ('dynamic', 'functional', 1.0),
    ('dynamic', 'dynamic', 1.0),

    -- Functional type advantages
    ('functional', 'procedural', 1.5),
    ('functional', 'static', 1.0),
    ('functional', 'dynamic', 1.0),
    ('functional', 'functional', 1.0),

    -- Procedural type advantages
    ('procedural', 'functional', 0.75),
    ('procedural', 'static', 1.0),
    ('procedural', 'dynamic', 1.0),
    ('procedural', 'procedural', 1.0);
