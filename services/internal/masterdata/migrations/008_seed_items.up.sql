-- どりーさん
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0001-7000-8000-000000000001', 'どりーさん', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0001-7000-8000-000000000001',
     '018f4e1a-0001-7000-8000-000000000001',
     'capture_rate_up', 'swift', 0.3,
     'どりーさんのSwift愛が炸裂した！捕獲率がアップ！', 0);

-- ざつくん
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0002-7000-8000-000000000002', 'ざつくん', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0002-7000-8000-000000000001',
     '018f4e1a-0002-7000-8000-000000000002',
     'capture_rate_up', 'ts', 0.3,
     'ざつくんのTypeScript愛が爆発した！捕獲率がアップ！', 0),
    ('018f4e1b-0002-7000-8000-000000000002',
     '018f4e1a-0002-7000-8000-000000000002',
     'escape', 'python', 0.0,
     'ざつくんはPythonが嫌いすぎて群馬に帰った。', 0);

-- レッドブル
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0003-7000-8000-000000000003', 'レッドブル', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0003-7000-8000-000000000001',
     '018f4e1a-0003-7000-8000-000000000003',
     'capture_rate_up', NULL, 0.2,
     'レッドブルを飲んで元気が出た！捕獲率がアップ！', 0);

-- モンスター
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0004-7000-8000-000000000004', 'モンスター', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0004-7000-8000-000000000001',
     '018f4e1a-0004-7000-8000-000000000004',
     'capture_rate_up', NULL, 0.2,
     'モンスターを飲んで元気が出た！捕獲率がアップ！', 0);

-- こんにゃく（群馬の名産品。pythonに使うとざつくんと同じく群馬に帰りたくなる）
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0005-7000-8000-000000000005', 'こんにゃく', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0005-7000-8000-000000000001',
     '018f4e1a-0005-7000-8000-000000000005',
     'capture_rate_up', 'python', 0.3,
     '群馬産こんにゃくのパワーでPythonを手懐けた！捕獲率がアップ！', 0);

-- クッション（開発中に使える。座り心地が良くて集中力UP → 全タイプに少し効く）
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0006-7000-8000-000000000006', 'クッション', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0006-7000-8000-000000000001',
     '018f4e1a-0006-7000-8000-000000000006',
     'capture_rate_up', NULL, 0.1,
     'クッションに座って集中力が上がった！捕獲率がアップ！', 0);

-- ひよこ（お土産。かわいさで相手がひるむ → 全タイプに少し効く）
INSERT INTO item_master (id, name, created_at) VALUES
    ('018f4e1a-0007-7000-8000-000000000007', 'ひよこ', now());

INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text, priority) VALUES
    ('018f4e1b-0007-7000-8000-000000000001',
     '018f4e1a-0007-7000-8000-000000000007',
     'capture_rate_up', NULL, 0.15,
     'ひよこのかわいさに相手がひるんだ！捕獲率がアップ！', 0);
