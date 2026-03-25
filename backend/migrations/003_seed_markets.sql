-- +goose Up
-- +goose StatementBegin

-- Seed a system user that owns the seed markets.
-- scid_sub is a fixed sentinel; real logins will create a separate row.
INSERT OR IGNORE INTO users (scid_sub, display_name, email, balance, is_moderator, is_admin)
VALUES ('seed:system', 'ScolyBot', '', 0, 1, 0);

-- Active markets (already approved, ready for trading)
INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will the multi-cargo elevator desync be fixed in patch 4.1?',
    'The elevator at Teasa Spaceport has been eating cargo since 3.22. Players regularly lose freight mid-transfer when the elevator desyncs between driver and server.',
    'bug_fixes',
    'Resolves YES if CIG''s 4.1 patch notes explicitly list the multi-cargo elevator desync as fixed, or if the bug is delisted from the Known Issues page on live.',
    '2026-07-01T23:59:59Z',
    'active',
    u.id,
    150
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will the Stanton–Pyro jump point be stable at 4.1 launch?',
    'The jump point has had erratic behaviour since Pyro went live — players report bouncing back to origin or entering permanent loading screens at the threshold.',
    'feature_delivery',
    'Resolves YES if the 4.1 live build ships without a Known Issue entry for jump point instability between Stanton and Pyro, and no CIG comm-link acknowledges the issue within 7 days of launch.',
    '2026-06-15T23:59:59Z',
    'active',
    u.id,
    120
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will patch 4.1 ship to live before 30 April 2026?',
    'Tracking CIG''s cadence. 4.0 hit live in December 2025. The question is whether the team can turn around 4.1 in under five months.',
    'patch_timing',
    'Resolves YES if the 4.1 release appears in the RSI Comm-Link as deployed to the live environment on or before 30 April 2026 (UTC).',
    '2026-04-30T23:59:59Z',
    'active',
    u.id,
    200
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will the Hull C be flyable before the end of 2026?',
    'The Hull C has been in active development for years. With dynamic cargo now in the game, the remaining blocker appears to be the articulated hull attachment system.',
    'feature_delivery',
    'Resolves YES if a flyable Hull C (not a placeholder stub) appears in any official build — PTU or live — before 31 December 2026.',
    '2026-12-31T23:59:59Z',
    'active',
    u.id,
    100
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will IAE 2026 be held in November as usual?',
    'The annual Intergalactic Aerospace Expo has run in November every year since 2019. Will the streak continue?',
    'community_events',
    'Resolves YES if CIG announces and runs an Intergalactic Aerospace Expo event during November 2026.',
    '2026-11-30T23:59:59Z',
    'active',
    u.id,
    80
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will the inventory item duplication exploit be patched before 4.2?',
    'A well-known item dupe involving rapid container swaps has been circulating since late 4.0. CIG has acknowledged the report.',
    'bug_fixes',
    'Resolves YES if CIG''s patch notes for any build between now and the 4.2 release explicitly address the item duplication exploit, or if the Known Issues entry is removed.',
    '2026-09-01T23:59:59Z',
    'active',
    u.id,
    100
FROM users u WHERE u.scid_sub = 'seed:system';

-- Pending review markets (submitted but not yet moderated)
INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will server meshing support 4 simultaneous shards by end of 2026?',
    'CIG''s road to a persistent universe runs through server meshing. Currently live builds run 2-shard configurations for Stanton. The question is whether we see 4-shard operation on live before year end.',
    'feature_delivery',
    'Resolves YES if CIG announces or demonstrates 4-shard live server meshing for any star system before 31 December 2026.',
    '2026-12-31T23:59:59Z',
    'pending_review',
    u.id,
    100
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will the Reclaimer tractor beam stop launching salvage into orbit?',
    'The Reclaimer''s built-in tractor beams intermittently apply massive upward velocity to salvage panels, sending them — and sometimes the operator — into space at high speed.',
    'bug_fixes',
    'Resolves YES if the bug is absent from 4.1 Known Issues on live and no new high-upvote Issue Council reports appear within 14 days of ship.',
    '2026-08-01T23:59:59Z',
    'pending_review',
    u.id,
    120
FROM users u WHERE u.scid_sub = 'seed:system';

INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, status, created_by, liquidity_param)
SELECT
    'Will there be a Star Citizen Free Fly event in Q3 2026?',
    'CIG has traditionally run at least two free-fly events per year. Will Q3 (July–September) 2026 see one?',
    'community_events',
    'Resolves YES if CIG runs any free-fly or free-to-play access event to Star Citizen between 1 July and 30 September 2026.',
    '2026-09-30T23:59:59Z',
    'pending_review',
    u.id,
    80
FROM users u WHERE u.scid_sub = 'seed:system';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM markets WHERE created_by = (SELECT id FROM users WHERE scid_sub = 'seed:system');
DELETE FROM users WHERE scid_sub = 'seed:system';
-- +goose StatementEnd
