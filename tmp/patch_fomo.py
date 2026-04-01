#!/usr/bin/env python3
"""Patch the fomo/+page.svelte for insurance picker"""

target = '/Users/lhw/src/astroledger/frontend/src/routes/fomo/+page.svelte'

with open(target, 'r') as f:
    content = f.read()

# Replacement 1: Add {:else if insurancePicking} clause and change buy() -> startPurchase()
old1 = "						{:else if !$isLoggedIn}\n							<a href=\"/auth/login\" class=\"badge-btn\">Login to Buy</a>\n						{:else}\n							<button\n								class=\"badge-btn\"\n								disabled={purchasing === badge.badge_key || !canBuy(badge)}\n								onclick={() => buy(badge)}\n							>\n								{purchasing === badge.badge_key ? 'Buying\u2026' : 'Buy'}\n							</button>\n						{/if}"

new1 = "						{:else if !$isLoggedIn}\n							<a href=\"/auth/login\" class=\"badge-btn\">Login to Buy</a>\n						{:else if insurancePicking === badge.badge_key}\n							<span class=\"insurance-hint\">Choose insurance below</span>\n						{:else}\n							<button\n								class=\"badge-btn\"\n								disabled={purchasing === badge.badge_key || !canBuy(badge)}\n								onclick={() => startPurchase(badge)}\n							>\n								{purchasing === badge.badge_key ? 'Buying\u2026' : 'Buy'}\n							</button>\n						{/if}"

if old1 in content:
    content = content.replace(old1, new1, 1)
    print("Replacement 1: OK")
else:
    # Try with tabs
    print("Replacement 1: NOT FOUND, dumping nearby content")
    idx = content.find("{:else if !$isLoggedIn}")
    print(repr(content[max(0,idx-10):idx+200]))

# Replacement 2: Insurance picker block after badge-footer div
old2 = "\t\t\t\t\t</div>\n\t\t\t\t\t{#if successKey === badge.badge_key}"
new2 = "\t\t\t\t\t</div>\n\t\t\t\t\t{#if insurancePicking === badge.badge_key}\n\t\t\t\t\t\t<div class=\"insurance-picker\">\n\t\t\t\t\t\t\t<p class=\"insurance-label\">Pick your insurance tier (cosmetic only):</p>\n\t\t\t\t\t\t\t<div class=\"insurance-pills\">\n\t\t\t\t\t\t\t\t<button\n\t\t\t\t\t\t\t\t\tclass=\"ins-pill none\"\n\t\t\t\t\t\t\t\t\tclass:selected={chosenInsurance[badge.badge_key] === ''}\n\t\t\t\t\t\t\t\t\tonclick={() => (chosenInsurance[badge.badge_key] = '')}\n\t\t\t\t\t\t\t\t>None</button>\n\t\t\t\t\t\t\t\t<button\n\t\t\t\t\t\t\t\t\tclass=\"ins-pill purple\"\n\t\t\t\t\t\t\t\t\tclass:selected={chosenInsurance[badge.badge_key] === '6w'}\n\t\t\t\t\t\t\t\t\tonclick={() => (chosenInsurance[badge.badge_key] = '6w')}\n\t\t\t\t\t\t\t\t>6 Weeks</button>\n\t\t\t\t\t\t\t\t<button\n\t\t\t\t\t\t\t\t\tclass=\"ins-pill orange\"\n\t\t\t\t\t\t\t\t\tclass:selected={chosenInsurance[badge.badge_key] === '120w'}\n\t\t\t\t\t\t\t\t\tonclick={() => (chosenInsurance[badge.badge_key] = '120w')}\n\t\t\t\t\t\t\t\t>120 Months</button>\n\t\t\t\t\t\t\t\t<button\n\t\t\t\t\t\t\t\t\tclass=\"ins-pill red\"\n\t\t\t\t\t\t\t\t\tclass:selected={chosenInsurance[badge.badge_key] === 'lti'}\n\t\t\t\t\t\t\t\t\tonclick={() => (chosenInsurance[badge.badge_key] = 'lti')}\n\t\t\t\t\t\t\t\t>LTI</button>\n\t\t\t\t\t\t\t</div>\n\t\t\t\t\t\t\t<div class=\"insurance-actions\">\n\t\t\t\t\t\t\t\t<button class=\"ins-cancel\" onclick={() => (insurancePicking = null)}>Cancel</button>\n\t\t\t\t\t\t\t\t<button\n\t\t\t\t\t\t\t\t\tclass=\"badge-btn ins-confirm\"\n\t\t\t\t\t\t\t\t\tonclick={() => buy(badge, chosenInsurance[badge.badge_key] ?? '')}\n\t\t\t\t\t\t\t\t>Confirm Purchase</button>\n\t\t\t\t\t\t\t</div>\n\t\t\t\t\t\t</div>\n\t\t\t\t\t{/if}\n\t\t\t\t\t{#if successKey === badge.badge_key}"

if old2 in content:
    content = content.replace(old2, new2, 1)
    print("Replacement 2: OK")
else:
    # Try spaces variant
    old2_spaces = "                                        </div>\n                                        {#if successKey === badge.badge_key}"
    if old2_spaces in content:
        # Need to repeat for spaces
        new2_spaces = old2_spaces  # placeholder - will handle below
        print("Replacement 2: file uses spaces, not tabs")
        idx = content.find("</div>\n")
        # find the one before successKey
        idx = content.find("{#if successKey === badge.badge_key}")
        print(repr(content[max(0,idx-100):idx+20]))
    else:
        print("Replacement 2: NOT FOUND")
        idx = content.find("{#if successKey === badge.badge_key}")
        print(repr(content[max(0,idx-100):idx+20]))

with open(target, 'w') as f:
    f.write(content)
print("Done")
