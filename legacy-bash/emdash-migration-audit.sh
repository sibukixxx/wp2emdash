#!/usr/bin/env bash
#
# EmDash migration audit — quantify WordPress complexity and surface migration risks.
#
# Run this **on the WordPress server** (or anywhere with WP-CLI + DB access),
# from the WordPress install root (the directory that contains wp-config.php).
#
# Output (default: ./emdash-audit-output/):
#   - summary.json        machine-readable digest of every metric
#   - risk-report.md      human-readable migration brief
#   - plugins.json        active/inactive plugins
#   - themes.json         installed themes
#   - post-types.json     registered post types
#   - taxonomies.json     registered taxonomies
#   - postmeta-top.csv    top 100 postmeta keys by row count
#   - uploads-extensions.txt  file-extension histogram of wp-content/uploads
#
# Risk score → cost band mapping is a *starting point* for sales conversations,
# not a fixed quote. Always sanity-check against the underlying data.
#
# Usage:
#   chmod +x scripts/audit/emdash-migration-audit.sh
#   ./scripts/audit/emdash-migration-audit.sh [output-dir]
#
# Requirements:
#   - bash 4+
#   - wp-cli (https://wp-cli.org)
#   - read access to the WordPress DB via wp-cli
#   - read access to wp-content/themes, plugins, mu-plugins, uploads

set -euo pipefail

OUT_DIR="${1:-emdash-audit-output}"
mkdir -p "$OUT_DIR"

if ! command -v wp >/dev/null 2>&1; then
  echo "ERROR: wp command not found. Install WP-CLI first." >&2
  exit 1
fi

if [ ! -f "wp-config.php" ]; then
  echo "ERROR: Run this script from the WordPress root directory (no wp-config.php here)." >&2
  exit 1
fi

DB_PREFIX=$(wp db prefix)
HOME_URL=$(wp option get home 2>/dev/null || echo "")
SITE_URL=$(wp option get siteurl 2>/dev/null || echo "")
ACTIVE_THEME=$(wp theme list --status=active --field=name 2>/dev/null || echo "")
THEME_DIR="wp-content/themes/${ACTIVE_THEME}"

echo "Running EmDash migration audit..."
echo "Output: $OUT_DIR"

# -------------------------
# Basic WordPress info
# -------------------------
WP_VERSION=$(wp core version 2>/dev/null || echo "unknown")
PHP_VERSION=$(php -r 'echo PHP_VERSION;' 2>/dev/null || echo "unknown")
IS_MULTISITE=$(wp eval 'echo is_multisite() ? "yes" : "no";' 2>/dev/null || echo "unknown")

POST_COUNT=$(wp post list --post_type=post --post_status=publish --format=count 2>/dev/null || echo 0)
PAGE_COUNT=$(wp post list --post_type=page --post_status=publish --format=count 2>/dev/null || echo 0)
DRAFT_COUNT=$(wp post list --post_status=draft --format=count 2>/dev/null || echo 0)
PRIVATE_COUNT=$(wp post list --post_status=private --format=count 2>/dev/null || echo 0)
CATEGORY_COUNT=$(wp term list category --format=count 2>/dev/null || echo 0)
TAG_COUNT=$(wp term list post_tag --format=count 2>/dev/null || echo 0)
USER_COUNT=$(wp user list --format=count 2>/dev/null || echo 0)
COMMENT_COUNT=$(wp comment list --status=approve --format=count 2>/dev/null || echo 0)

# -------------------------
# Uploads
# -------------------------
UPLOADS_DIR="wp-content/uploads"
UPLOADS_EXISTS="no"
UPLOADS_SIZE="0"
UPLOADS_FILE_COUNT=0

if [ -d "$UPLOADS_DIR" ]; then
  UPLOADS_EXISTS="yes"
  UPLOADS_SIZE=$(du -sh "$UPLOADS_DIR" 2>/dev/null | awk '{print $1}')
  UPLOADS_FILE_COUNT=$(find "$UPLOADS_DIR" -type f 2>/dev/null | wc -l | tr -d ' ')
  find "$UPLOADS_DIR" -type f 2>/dev/null \
    | sed 's/.*\.//' \
    | tr '[:upper:]' '[:lower:]' \
    | sort \
    | uniq -c \
    | sort -nr > "$OUT_DIR/uploads-extensions.txt" || true
fi

POSTS_WITH_UPLOADS=$(wp db query "
SELECT COUNT(*) FROM ${DB_PREFIX}posts
WHERE post_content LIKE '%wp-content/uploads%';
" --skip-column-names 2>/dev/null || echo 0)

POSTS_WITH_HTTP=$(wp db query "
SELECT COUNT(*) FROM ${DB_PREFIX}posts
WHERE post_content LIKE '%http://%';
" --skip-column-names 2>/dev/null || echo 0)

# -------------------------
# Plugins
# -------------------------
wp plugin list --format=json > "$OUT_DIR/plugins.json" 2>/dev/null || echo "[]" > "$OUT_DIR/plugins.json"
ACTIVE_PLUGIN_COUNT=$(wp plugin list --status=active --format=count 2>/dev/null || echo 0)

PLUGIN_NAMES=$(wp plugin list --status=active --field=name 2>/dev/null || true)

HAS_ACF=0
HAS_WOOCOMMERCE=0
HAS_SEO=0
HAS_FORM=0
HAS_REDIRECT=0
HAS_MEMBER=0
HAS_MULTILINGUAL=0
HAS_CACHE=0

echo "$PLUGIN_NAMES" | grep -Eiq "advanced-custom-fields|acf" && HAS_ACF=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "woocommerce" && HAS_WOOCOMMERCE=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "wordpress-seo|seo-by-rank-math|all-in-one-seo|aioseo" && HAS_SEO=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "contact-form-7|mw-wp-form|wpforms|ninja-forms|gravityforms" && HAS_FORM=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "redirection|safe-redirect-manager" && HAS_REDIRECT=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "ultimate-member|paid-memberships-pro|memberpress|simple-membership" && HAS_MEMBER=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "wpml|polylang|translatepress|multilingualpress" && HAS_MULTILINGUAL=1 || true
echo "$PLUGIN_NAMES" | grep -Eiq "autoptimize|wp-rocket|w3-total-cache|litespeed-cache|wp-super-cache" && HAS_CACHE=1 || true

# -------------------------
# Themes
# -------------------------
wp theme list --format=json > "$OUT_DIR/themes.json" 2>/dev/null || echo "[]" > "$OUT_DIR/themes.json"

THEME_PHP_COUNT=0
THEME_CSS_COUNT=0
THEME_JS_COUNT=0
THEME_TEMPLATE_COUNT=0
THEME_HOOK_COUNT=0
THEME_JQUERY_COUNT=0

if [ -d "$THEME_DIR" ]; then
  THEME_PHP_COUNT=$(find "$THEME_DIR" -type f -name "*.php" | wc -l | tr -d ' ')
  THEME_CSS_COUNT=$(find "$THEME_DIR" -type f -name "*.css" | wc -l | tr -d ' ')
  THEME_JS_COUNT=$(find "$THEME_DIR" -type f -name "*.js" | wc -l | tr -d ' ')
  THEME_TEMPLATE_COUNT=$(grep -R "Template Name:" "$THEME_DIR" 2>/dev/null | wc -l | tr -d ' ')
  THEME_HOOK_COUNT=$(grep -R "add_action\|add_filter\|register_post_type\|register_taxonomy\|add_shortcode\|register_rest_route\|add_meta_box\|wp_schedule_event\|wp_remote_" "$THEME_DIR" 2>/dev/null | wc -l | tr -d ' ')
  THEME_JQUERY_COUNT=$(grep -R "jquery\|admin-ajax.php\|slick\|swiper\|owlCarousel" "$THEME_DIR" 2>/dev/null | wc -l | tr -d ' ')
fi

# -------------------------
# MU plugins
# -------------------------
MU_PLUGIN_COUNT=0
MU_PLUGIN_HOOK_COUNT=0

if [ -d "wp-content/mu-plugins" ]; then
  MU_PLUGIN_COUNT=$(find wp-content/mu-plugins -type f -name "*.php" | wc -l | tr -d ' ')
  MU_PLUGIN_HOOK_COUNT=$(grep -R "add_action\|add_filter\|register_post_type\|register_taxonomy\|wp_remote_\|wp_redirect\|register_rest_route" wp-content/mu-plugins 2>/dev/null | wc -l | tr -d ' ')
fi

# -------------------------
# Post types / taxonomies
# -------------------------
wp post-type list --format=json > "$OUT_DIR/post-types.json" 2>/dev/null || echo "[]" > "$OUT_DIR/post-types.json"
wp taxonomy list --format=json > "$OUT_DIR/taxonomies.json" 2>/dev/null || echo "[]" > "$OUT_DIR/taxonomies.json"

CUSTOM_POST_TYPE_COUNT=$(wp post-type list --field=name 2>/dev/null \
  | grep -Ev "^(post|page|attachment|revision|nav_menu_item|custom_css|customize_changeset|oembed_cache|user_request|wp_block|wp_template|wp_template_part|wp_global_styles|wp_navigation)$" \
  | wc -l | tr -d ' ')

CUSTOM_TAXONOMY_COUNT=$(wp taxonomy list --field=name 2>/dev/null \
  | grep -Ev "^(category|post_tag|nav_menu|link_category|post_format|wp_theme)$" \
  | wc -l | tr -d ' ')

# -------------------------
# Postmeta / SEO / serialized
# -------------------------
wp db query "
SELECT meta_key, COUNT(*) AS count
FROM ${DB_PREFIX}postmeta
GROUP BY meta_key
ORDER BY count DESC
LIMIT 100;
" --format=csv > "$OUT_DIR/postmeta-top.csv" 2>/dev/null || true

SEO_META_COUNT=$(wp db query "
SELECT COUNT(*) FROM ${DB_PREFIX}postmeta
WHERE meta_key LIKE '%yoast%'
   OR meta_key LIKE '%rank_math%'
   OR meta_key LIKE '%aioseo%';
" --skip-column-names 2>/dev/null || echo 0)

SERIALIZED_META_COUNT=$(wp db query "
SELECT COUNT(*) FROM ${DB_PREFIX}postmeta
WHERE meta_value LIKE 'a:%'
   OR meta_value LIKE 'O:%';
" --skip-column-names 2>/dev/null || echo 0)

SHORTCODE_POST_COUNT=$(wp db query "
SELECT COUNT(*) FROM ${DB_PREFIX}posts
WHERE post_content REGEXP '\\\\[[a-zA-Z0-9_-]+';
" --skip-column-names 2>/dev/null || echo 0)

# -------------------------
# Redirect / htaccess / external integrations
# -------------------------
HTACCESS_REDIRECT_COUNT=0
if [ -f ".htaccess" ]; then
  HTACCESS_REDIRECT_COUNT=$(grep -Ei "redirect|rewrite" .htaccess 2>/dev/null | wc -l | tr -d ' ')
fi

CODE_REDIRECT_COUNT=$(grep -R "wp_redirect\|header('Location\|header(\"Location" wp-content/themes wp-content/plugins wp-content/mu-plugins 2>/dev/null | wc -l | tr -d ' ' || echo 0)

EXTERNAL_INTEGRATION_COUNT=$(grep -R "wp_remote_get\|wp_remote_post\|curl_init\|admin-ajax.php\|register_rest_route\|webhook\|stripe\|line\|slack\|mailchimp" wp-content/themes wp-content/plugins wp-content/mu-plugins 2>/dev/null | wc -l | tr -d ' ' || echo 0)

# -------------------------
# Score
# -------------------------
SCORE=0
RISK_ITEMS=()

add_score() {
  local points="$1"
  local message="$2"
  SCORE=$((SCORE + points))
  RISK_ITEMS+=("+${points}: ${message}")
}

[ "$POST_COUNT" -gt 100 ] && add_score 5 "投稿数が100件超"
[ "$POST_COUNT" -gt 500 ] && add_score 10 "投稿数が500件超"
[ "$PAGE_COUNT" -gt 20 ] && add_score 5 "固定ページが20件超"
[ "$ACTIVE_PLUGIN_COUNT" -gt 10 ] && add_score 5 "有効プラグインが10個超"
[ "$ACTIVE_PLUGIN_COUNT" -gt 20 ] && add_score 10 "有効プラグインが20個超"
[ "$HAS_ACF" -eq 1 ] && add_score 15 "ACF/カスタムフィールド系プラグインあり"
[ "$HAS_WOOCOMMERCE" -eq 1 ] && add_score 30 "WooCommerceあり"
[ "$HAS_MEMBER" -eq 1 ] && add_score 25 "会員系プラグインあり"
[ "$HAS_MULTILINGUAL" -eq 1 ] && add_score 20 "多言語系プラグインあり"
[ "$CUSTOM_POST_TYPE_COUNT" -gt 0 ] && add_score 10 "カスタム投稿タイプあり"
[ "$CUSTOM_POST_TYPE_COUNT" -gt 2 ] && add_score 15 "カスタム投稿タイプが3個以上"
[ "$CUSTOM_TAXONOMY_COUNT" -gt 0 ] && add_score 10 "カスタムタクソノミーあり"
[ "$SHORTCODE_POST_COUNT" -gt 20 ] && add_score 10 "ショートコード利用投稿が多い"
[ "$THEME_HOOK_COUNT" -gt 50 ] && add_score 10 "テーマ/functions.php周辺のhookが多い"
[ "$MU_PLUGIN_COUNT" -gt 0 ] && add_score 10 "mu-pluginsあり"
[ "$EXTERNAL_INTEGRATION_COUNT" -gt 0 ] && add_score 10 "外部連携/API/Ajaxらしきコードあり"
[ "$HAS_REDIRECT" -eq 1 ] && add_score 10 "リダイレクト系プラグインあり"
[ "$HAS_SEO" -eq 1 ] && add_score 5 "SEOプラグインあり"
[ "$SEO_META_COUNT" -gt 100 ] && add_score 10 "SEO metaが100件超"
[ "$SERIALIZED_META_COUNT" -gt 100 ] && add_score 10 "serialized postmetaが多い"
[ "$HTACCESS_REDIRECT_COUNT" -gt 10 ] && add_score 10 ".htaccess rewrite/redirectが多い"
[ "$CODE_REDIRECT_COUNT" -gt 0 ] && add_score 10 "コード内redirectあり"
[ "$THEME_JQUERY_COUNT" -gt 20 ] && add_score 10 "jQuery/admin-ajax等の依存が多い"

if [ "$SCORE" -le 20 ]; then
  LEVEL="Simple"
  ESTIMATE="5万〜20万円"
elif [ "$SCORE" -le 50 ]; then
  LEVEL="Standard"
  ESTIMATE="20万〜60万円"
elif [ "$SCORE" -le 90 ]; then
  LEVEL="Complex"
  ESTIMATE="60万〜150万円"
elif [ "$SCORE" -le 130 ]; then
  LEVEL="High Risk"
  ESTIMATE="150万〜300万円"
else
  LEVEL="Rebuild Project"
  ESTIMATE="300万円〜 / 個別見積り"
fi

# -------------------------
# JSON summary
# -------------------------
cat > "$OUT_DIR/summary.json" <<JSON
{
  "site": {
    "home_url": "$HOME_URL",
    "site_url": "$SITE_URL",
    "wp_version": "$WP_VERSION",
    "php_version": "$PHP_VERSION",
    "db_prefix": "$DB_PREFIX",
    "is_multisite": "$IS_MULTISITE"
  },
  "content": {
    "posts": $POST_COUNT,
    "pages": $PAGE_COUNT,
    "drafts": $DRAFT_COUNT,
    "private_posts": $PRIVATE_COUNT,
    "categories": $CATEGORY_COUNT,
    "tags": $TAG_COUNT,
    "users": $USER_COUNT,
    "approved_comments": $COMMENT_COUNT
  },
  "uploads": {
    "exists": "$UPLOADS_EXISTS",
    "size": "$UPLOADS_SIZE",
    "file_count": $UPLOADS_FILE_COUNT,
    "posts_with_uploads_paths": $POSTS_WITH_UPLOADS,
    "posts_with_http_urls": $POSTS_WITH_HTTP
  },
  "theme": {
    "active_theme": "$ACTIVE_THEME",
    "php_files": $THEME_PHP_COUNT,
    "css_files": $THEME_CSS_COUNT,
    "js_files": $THEME_JS_COUNT,
    "page_templates": $THEME_TEMPLATE_COUNT,
    "hook_like_occurrences": $THEME_HOOK_COUNT,
    "jquery_like_occurrences": $THEME_JQUERY_COUNT
  },
  "plugins": {
    "active_count": $ACTIVE_PLUGIN_COUNT,
    "has_acf": $HAS_ACF,
    "has_woocommerce": $HAS_WOOCOMMERCE,
    "has_seo": $HAS_SEO,
    "has_form": $HAS_FORM,
    "has_redirect": $HAS_REDIRECT,
    "has_member": $HAS_MEMBER,
    "has_multilingual": $HAS_MULTILINGUAL,
    "has_cache": $HAS_CACHE
  },
  "customization": {
    "custom_post_type_count": $CUSTOM_POST_TYPE_COUNT,
    "custom_taxonomy_count": $CUSTOM_TAXONOMY_COUNT,
    "mu_plugin_count": $MU_PLUGIN_COUNT,
    "mu_plugin_hook_like_occurrences": $MU_PLUGIN_HOOK_COUNT,
    "shortcode_post_count": $SHORTCODE_POST_COUNT,
    "seo_meta_count": $SEO_META_COUNT,
    "serialized_meta_count": $SERIALIZED_META_COUNT,
    "htaccess_redirect_like_lines": $HTACCESS_REDIRECT_COUNT,
    "code_redirect_like_occurrences": $CODE_REDIRECT_COUNT,
    "external_integration_like_occurrences": $EXTERNAL_INTEGRATION_COUNT
  },
  "risk": {
    "score": $SCORE,
    "level": "$LEVEL",
    "rough_estimate": "$ESTIMATE"
  }
}
JSON

# -------------------------
# Markdown report
# -------------------------
{
  echo "# EmDash Migration Audit Report"
  echo
  echo "## Summary"
  echo
  echo "- URL: $HOME_URL"
  echo "- WordPress: $WP_VERSION"
  echo "- PHP: $PHP_VERSION"
  echo "- Active theme: $ACTIVE_THEME"
  echo "- Risk score: $SCORE"
  echo "- Level: $LEVEL"
  echo "- Rough estimate: $ESTIMATE"
  echo
  echo "## Content"
  echo
  echo "- Posts: $POST_COUNT"
  echo "- Pages: $PAGE_COUNT"
  echo "- Drafts: $DRAFT_COUNT"
  echo "- Private posts: $PRIVATE_COUNT"
  echo "- Categories: $CATEGORY_COUNT"
  echo "- Tags: $TAG_COUNT"
  echo "- Users: $USER_COUNT"
  echo "- Approved comments: $COMMENT_COUNT"
  echo
  echo "## Uploads"
  echo
  echo "- Exists: $UPLOADS_EXISTS"
  echo "- Size: $UPLOADS_SIZE"
  echo "- File count: $UPLOADS_FILE_COUNT"
  echo "- Posts with wp-content/uploads paths: $POSTS_WITH_UPLOADS"
  echo "- Posts with http URLs: $POSTS_WITH_HTTP"
  echo
  echo "## Plugins"
  echo
  echo "- Active plugins: $ACTIVE_PLUGIN_COUNT"
  echo "- ACF: $HAS_ACF"
  echo "- WooCommerce: $HAS_WOOCOMMERCE"
  echo "- SEO plugin: $HAS_SEO"
  echo "- Form plugin: $HAS_FORM"
  echo "- Redirect plugin: $HAS_REDIRECT"
  echo "- Member plugin: $HAS_MEMBER"
  echo "- Multilingual plugin: $HAS_MULTILINGUAL"
  echo
  echo "## Theme"
  echo
  echo "- PHP files: $THEME_PHP_COUNT"
  echo "- CSS files: $THEME_CSS_COUNT"
  echo "- JS files: $THEME_JS_COUNT"
  echo "- Page templates: $THEME_TEMPLATE_COUNT"
  echo "- Hook-like occurrences: $THEME_HOOK_COUNT"
  echo "- jQuery-like occurrences: $THEME_JQUERY_COUNT"
  echo
  echo "## Customization"
  echo
  echo "- Custom post types: $CUSTOM_POST_TYPE_COUNT"
  echo "- Custom taxonomies: $CUSTOM_TAXONOMY_COUNT"
  echo "- MU plugin files: $MU_PLUGIN_COUNT"
  echo "- Shortcode posts: $SHORTCODE_POST_COUNT"
  echo "- SEO meta count: $SEO_META_COUNT"
  echo "- Serialized meta count: $SERIALIZED_META_COUNT"
  echo "- .htaccess redirect/rewrite lines: $HTACCESS_REDIRECT_COUNT"
  echo "- Code redirect occurrences: $CODE_REDIRECT_COUNT"
  echo "- External integration occurrences: $EXTERNAL_INTEGRATION_COUNT"
  echo
  echo "## Risk Items"
  echo
  if [ "${#RISK_ITEMS[@]}" -eq 0 ]; then
    echo "- No major risk items detected by this script."
  else
    for item in "${RISK_ITEMS[@]}"; do
      echo "- $item"
    done
  fi
  echo
  echo "## Recommended Next Actions"
  echo
  echo "1. Review plugins.json"
  echo "2. Review postmeta-top.csv"
  echo "3. Review uploads-extensions.txt"
  echo "4. Manually inspect functions.php and mu-plugins"
  echo "5. Decide whether old /wp-content/uploads URLs should be preserved"
  echo "6. Decide whether SEO metadata should be migrated fully or partially"
} > "$OUT_DIR/risk-report.md"

echo "Done."
echo "Report:  $OUT_DIR/risk-report.md"
echo "Summary: $OUT_DIR/summary.json"
