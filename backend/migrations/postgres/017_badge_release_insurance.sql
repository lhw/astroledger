-- +goose Up

-- Add cosmetic insurance tier to badge releases and badge definitions.
-- Values: '6w' (6 Weeks), '120w' (120 Weeks), 'lti' (Lifetime Insurance).
ALTER TABLE badge_releases    ADD COLUMN IF NOT EXISTS insurance TEXT NOT NULL DEFAULT '';
ALTER TABLE badge_definitions ADD COLUMN IF NOT EXISTS insurance TEXT NOT NULL DEFAULT '';

-- Retroactively grant LTI to all earned badges (purchase_price = 0 means not purchased from store).
UPDATE user_badges SET insurance = 'lti' WHERE purchase_price = 0 AND insurance = '';

-- +goose Down

UPDATE user_badges SET insurance = '' WHERE purchase_price = 0 AND insurance = 'lti';
ALTER TABLE badge_definitions DROP COLUMN IF EXISTS insurance;
ALTER TABLE badge_releases    DROP COLUMN IF EXISTS insurance;
