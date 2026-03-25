-- name: GetEnabledAutofilterRules :many
SELECT rule_type, value FROM autofilter_rules WHERE enabled = 1;
