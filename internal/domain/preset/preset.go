// Package preset describes the migration phase combinations exposed via
// `wp2emdash run --preset <name>`. Each preset is a list of high-level
// "intents" that the runner translates into concrete tasks.
package preset

import "fmt"

// Name is the public identifier passed to the CLI.
type Name string

const (
	Minimal         Name = "minimal"
	SmallProduction Name = "small-production"
	SEOProduction   Name = "seo-production"
	MediaHeavy      Name = "media-heavy"
	CustomRebuild   Name = "custom-rebuild"
)

// Step is one logical action inside a preset. The runner maps each Kind to a
// concrete implementation; unknown kinds are reported as "not implemented yet".
type Step struct {
	Kind    string
	Summary string
}

// Preset is a phase/step plan keyed by Name.
type Preset struct {
	Name        Name
	Description string
	Phases      []Phase
}

type Phase struct {
	Name  string
	Steps []Step
}

// All known presets, in roughly increasing complexity / scope.
func All() []Preset {
	return []Preset{
		{
			Name:        Minimal,
			Description: "PoC: WordPress 複雑度を測り EmDash 移行可否レポートを出すだけ",
			Phases: []Phase{
				{Name: "audit", Steps: []Step{
					{Kind: "doctor", Summary: "外部ツール (wp / wrangler / git) の存在確認"},
					{Kind: "audit", Summary: "WP-CLI による 14 観点の計測"},
					{Kind: "media-scan-sample", Summary: "uploads の最初 200 ファイル分の manifest（hash なし）"},
					{Kind: "report", Summary: "summary.json + risk-report.md を生成"},
				}},
			},
		},
		{
			Name:        SmallProduction,
			Description: "小規模ブログ/LPの本番移行: 投稿/固定ページ/uploads/standard SEO",
			Phases: []Phase{
				{Name: "audit", Steps: []Step{
					{Kind: "audit", Summary: "WP-CLI 全観点"},
					{Kind: "media-scan", Summary: "uploads 全量 manifest（hash なし）"},
				}},
				{Name: "plan", Steps: []Step{
					{Kind: "report", Summary: "summary.json + risk-report.md"},
					{Kind: "todo", Summary: "TODO: db plan / env generate / deploy staging（v0.2 以降）"},
				}},
			},
		},
		{
			Name:        SEOProduction,
			Description: "SEO を落とさない本番移行: meta / canonical / redirect / OGP",
			Phases: []Phase{
				{Name: "audit", Steps: []Step{
					{Kind: "audit", Summary: "WP-CLI 全観点 + SEO meta 集計"},
					{Kind: "media-scan", Summary: "uploads 全量 manifest（hash なし）"},
					{Kind: "todo", Summary: "TODO: seo extract-meta / seo extract-redirects（v0.4）"},
				}},
				{Name: "plan", Steps: []Step{
					{Kind: "report", Summary: "summary.json + risk-report.md"},
					{Kind: "todo", Summary: "TODO: URL map / Cloudflare Rules 計画（v0.4）"},
				}},
			},
		},
		{
			Name:        MediaHeavy,
			Description: "大量画像/PDF/動画を R2 に安全移送するシナリオ",
			Phases: []Phase{
				{Name: "audit", Steps: []Step{
					{Kind: "audit", Summary: "WP-CLI 全観点"},
					{Kind: "media-scan-hash", Summary: "uploads 全量 manifest + SHA-256"},
					{Kind: "todo", Summary: "TODO: media sync wrapper / route generate（v0.3）"},
				}},
			},
		},
		{
			Name:        CustomRebuild,
			Description: "functions.php/plugins/mu-plugins/外部連携を含む再構築案件",
			Phases: []Phase{
				{Name: "audit", Steps: []Step{
					{Kind: "audit", Summary: "WP-CLI 全観点"},
					{Kind: "todo", Summary: "TODO: theme/plugins/mu-plugins/api/shortcode/custom-fields 詳細解析（v0.5）"},
				}},
				{Name: "plan", Steps: []Step{
					{Kind: "report", Summary: "再構築計画レポート（v0.5 で詳細化）"},
				}},
			},
		},
	}
}

// Lookup returns the preset for n, or an error.
func Lookup(n Name) (Preset, error) {
	for _, p := range All() {
		if p.Name == n {
			return p, nil
		}
	}
	return Preset{}, fmt.Errorf("unknown preset %q", n)
}

// Names returns the canonical preset names — used by the CLI for auto-help.
func Names() []string {
	all := All()
	out := make([]string, 0, len(all))
	for _, p := range all {
		out = append(out, string(p.Name))
	}
	return out
}
