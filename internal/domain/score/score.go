// Package score turns an audit struct into a raw risk score plus reasons.
// Business-facing labels and price hints are applied outside this package via
// a policy file so the core can remain publication-friendly.
package score

import (
	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
)

// Level groups the numeric score into the five sales-facing bands.
type Level string

const (
	LevelSimple   Level = "Simple"
	LevelStandard Level = "Standard"
	LevelComplex  Level = "Complex"
	LevelHighRisk Level = "High Risk"
	LevelRebuild  Level = "Rebuild Project"
)

// Reason is one entry in the audit's "why was this scored that way" list.
type Reason struct {
	Points int    `json:"points"`
	Code   string `json:"code"`
	Text   string `json:"text"`
}

// Result is the public output of the scorer.
type Result struct {
	Score    int      `json:"score"`
	Level    Level    `json:"level"`
	Estimate string   `json:"estimate"`
	Reasons  []Reason `json:"reasons"`
}

// Compute applies the rubric to an audit and returns the rolled-up result.
func Compute(a audit.Audit) Result {
	res := Result{}
	add := func(points int, code, text string, when bool) {
		if !when {
			return
		}
		res.Score += points
		res.Reasons = append(res.Reasons, Reason{Points: points, Code: code, Text: text})
	}

	add(5, "posts.gt100", "投稿数が100件超", a.Content.Posts > 100)
	add(10, "posts.gt500", "投稿数が500件超", a.Content.Posts > 500)
	add(5, "pages.gt20", "固定ページが20件超", a.Content.Pages > 20)

	add(5, "plugins.gt10", "有効プラグインが10個超", a.Plugins.ActiveCount > 10)
	add(10, "plugins.gt20", "有効プラグインが20個超", a.Plugins.ActiveCount > 20)

	add(15, "plugin.acf", "ACF/カスタムフィールド系プラグインあり", a.Plugins.HasACF)
	add(30, "plugin.woo", "WooCommerceあり", a.Plugins.HasWooCommerce)
	add(25, "plugin.member", "会員系プラグインあり", a.Plugins.HasMember)
	add(20, "plugin.multilingual", "多言語系プラグインあり", a.Plugins.HasMultilingual)
	add(10, "plugin.redirect", "リダイレクト系プラグインあり", a.Plugins.HasRedirect)
	add(5, "plugin.seo", "SEOプラグインあり", a.Plugins.HasSEO)

	add(10, "cpt.any", "カスタム投稿タイプあり", a.Customization.CustomPostTypeCount > 0)
	add(15, "cpt.gte3", "カスタム投稿タイプが3個以上", a.Customization.CustomPostTypeCount >= 3)
	add(10, "tax.any", "カスタムタクソノミーあり", a.Customization.CustomTaxonomyCount > 0)

	add(10, "shortcode.gt20", "ショートコード利用投稿が多い", a.Customization.ShortcodePostCount > 20)
	add(10, "theme.hooks.gt50", "テーマ/functions.php周辺のhookが多い", a.Theme.HookLikeOccurrences > 50)
	add(10, "muplugins.any", "mu-pluginsあり", a.Customization.MUPluginCount > 0)
	add(10, "external.any", "外部連携/API/Ajaxらしきコードあり", a.Customization.ExternalIntegrationOccurrences > 0)
	add(10, "seo.meta.gt100", "SEO metaが100件超", a.Customization.SEOMetaCount > 100)
	add(10, "serialized.gt100", "serialized postmetaが多い", a.Customization.SerializedMetaCount > 100)
	add(10, "htaccess.gt10", ".htaccess rewrite/redirectが多い", a.Customization.HtaccessRedirectLikeLines > 10)
	add(10, "code.redirect.any", "コード内redirectあり", a.Customization.CodeRedirectLikeOccurrences > 0)
	add(10, "theme.jquery.gt20", "jQuery/admin-ajax等の依存が多い", a.Theme.JQueryLikeOccurrences > 20)

	return res
}
