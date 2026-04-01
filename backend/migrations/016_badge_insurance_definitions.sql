-- +goose Up

-- ── Badge Definitions Table ───────────────────────────────────────────────────
-- Central registry of all badge metadata — both seeded-from-code (is_hardcoded=1)
-- and admin-created custom badges (is_hardcoded=0). Admins can edit any row.
CREATE TABLE badge_definitions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    key          TEXT    NOT NULL UNIQUE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL,
    tier         INTEGER NOT NULL CHECK(tier BETWEEN 1 AND 5),
    icon         TEXT    NOT NULL DEFAULT '',  -- Custom symbol/emoji; empty = tier default (▲●◆◈★)
    is_hardcoded INTEGER NOT NULL DEFAULT 0,   -- 1 = seeded from code, 0 = admin-created
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Seed all non-admiral-rank badge definitions from AllBadges.
-- Admiral rank badges are auto-awarded by spend threshold, not released in the store.
INSERT INTO badge_definitions (key, title, description, tier, is_hardcoded) VALUES
-- ── Earned ────────────────────────────────────────────────────────────────────
('first_blood',       'First Blood',        'Made your first ever trade. The journey begins.',                                        1, 1),
('quick_shot',        'Quick Shot',          '10 trades in. Getting the hang of this.',                                              1, 1),
('market_maven',      'Market Maven',        '50 trades. Your mouse button is suffering.',                                           2, 1),
('seasoned_trader',   'Seasoned Trader',     '100 trades. At this point it''s a lifestyle.',                                         2, 1),
('market_obsessed',   'Market Obsessed',     '250 trades. We''re not judging. We''re in awe.',                                       3, 1),
('galaxy_brained',    'Galaxy Brained',      '500 trades. When does this become a job?',                                             4, 1),
('bug_prophet',       'Bug Prophet',         'Correctly predicted 5 markets. The ''verse bends to your will.',                      3, 1),
('skeptic',           'Skeptic',             'Correctly predicted 10 markets. You''ve seen this before.',                            2, 1),
('oracle',            'Oracle',              '25 correct predictions. Are you actually clairvoyant?',                                4, 1),
('eternal_optimist',  'Eternal Optimist',    'Active in 10 or more markets. Hope springs eternal.',                                  2, 1),
('doomsayer',         'Doomsayer',           'Predicted against the crowd in 10 or more markets.',                                   2, 1),
('portfolio_manager', 'Portfolio Manager',   'Positions in 25 markets. Hedged everywhere, profitable nowhere.',                     2, 1),
('universe_citizen',  'Universe Citizen',    'Positions in 50 markets. You have no free time and we respect that.',                  3, 1),
('market_founder',    'Market Founder',      'Posted a market that made it live. It was probably about bugs.',                       1, 1),
('serial_founder',    'Serial Founder',      '5 live markets. You''re practically running the ''verse.',                             2, 1),
-- ── FOMO Store — general/unlimited ────────────────────────────────────────────
('citizen_backer',           'Citizen Backer',              'A true believer. Been here since before it was cool.',                                                          1, 1),
('professional_bug_finder',  'Professional Bug Finder',     'It''s not a bug, it''s a stretch goal.',                                                                        1, 1),
('aurora_pilot',             'Aurora Pilot',                'Started small, dreamed big. The Aurora is a classic.',                                                          1, 1),
('roadmap_reader',           'Roadmap Reader',              'Holds 47 open tabs of schedule promises. Refreshes hourly.',                                                    1, 1),
('warp_speed',               'Warp Speed',                  'Will jump to conclusions faster than quantum drive.',                                                           1, 1),
('mostly_backer',            'Mostly Backer',               'In since 2012. Still waiting. Mostly fine about it.',                                                           2, 1),
('hangar_queen',             'Hangar Queen',                'The ships sit in the hangar. The pilots sit at the keyboard. Aspirational.',                                    2, 1),
('tech_preview_survivor',    'Tech Preview Survivor',       'I''ve seen things you wouldn''t believe. Then they wiped the servers.',                                         2, 1),
('star_gazer',               'Star Gazer',                  'Watched the CitizenCon stream live every year. No regrets.',                                                    2, 1),
('alpha_tester',             'Alpha Tester',                'Tested things that clearly weren''t ready. Knew it. Did it anyway.',                                            2, 1),
('space_whale',              'Space Whale',                 'Your wallet is canon-sized. The CIG shareholders send their regards.',                                          2, 1),
('bugged_not_broken',        'Bugged, Not Broken',          'It''s a feature. Definitely a feature. A very intentional feature.',                                            2, 1),
('verse_veteran',            '''Verse Veteran',              'A seasoned survivor of the persistent universe. Emphasis on survivor.',                                         2, 1),
('persistent_citizen',       'Persistent Universe Citizen', 'Was there for server meshing. Both times. Survived both wipes.',                                                3, 1),
('org_leader',               'Org Leader',                  'Commands fleets. Herds cats. Usually the same thing.',                                                          3, 1),
('900i_enjoyer',             '900i Enjoyer',                'Luxury is a lifestyle, not a budget.',                                                                          3, 1),
-- ── Hull Limited ──────────────────────────────────────────────────────────────
('system_colonist',      'System Colonist',  'Reserved a plot in a star system that may never ship. We admire the commitment.',                     3, 1),
('idris_captain',        'Idris Captain',    'Full sovereignty. Even in a game about bugs.',                                                        4, 1),
('backer_royalty',       'Backer Royalty',   'The original true believers. You were there before it was ironic.',                                   4, 1),
('fleet_commander_badge','Fleet Commander',  'Controls more ships than actual crew. The spreadsheet is enormous.',                                  4, 1),
('golden_ticket',        'Golden Ticket',    'The most prestigious flex in the ''verse. Pure gold.',                                                5, 1),
('unobtainium',          'Unobtainium Tier', 'You spent HOW much on fake badges? We are genuinely speechless.',                                     5, 1),
-- ── Rotating (time-limited) ───────────────────────────────────────────────────
('alpha_optimist',       'Alpha Optimist',      'Believed the patch notes. Every time. Bless your heart.',                                           2, 1),
('q4_enjoyer',           'Q4 Enjoyer',           'Adjusted expectations quarterly since 2015 and feels great about it.',                             2, 1),
('citizencon_pilgrim',   'CitizenCon Pilgrim',   'Was there. Watched the stream. Followed every thread. No regrets.',                                3, 1);

-- Admiral rank badges are intentionally excluded from badge_definitions.
-- They are auto-awarded by lifetime FOMO spend thresholds (see service/badges.go).

-- ── Insurance Column ──────────────────────────────────────────────────────────
-- Stores the cosmetic insurance tier chosen when a FOMO badge was purchased.
-- Possible values: '6w' (6 Weeks), '120w' (120 Weeks), 'lti' (LTI — Lifetime Insurance),
-- or '' (empty string) for earned-only badges.
ALTER TABLE user_badges ADD COLUMN insurance TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE user_badges DROP COLUMN insurance;
DROP TABLE badge_definitions;
