package audit

type Audit struct {
	Site          SiteInfo     `json:"site"`
	Content       ContentStats `json:"content"`
	Uploads       UploadsStats `json:"uploads"`
	Theme         ThemeStats   `json:"theme"`
	Plugins       PluginsStats `json:"plugins"`
	Customization CustomStats  `json:"customization"`
}

type SiteInfo struct {
	HomeURL     string `json:"home_url"`
	SiteURL     string `json:"site_url"`
	WPVersion   string `json:"wp_version"`
	PHPVersion  string `json:"php_version"`
	DBPrefix    string `json:"db_prefix"`
	IsMultisite string `json:"is_multisite"`
}

type ContentStats struct {
	Posts            int `json:"posts"`
	Pages            int `json:"pages"`
	Drafts           int `json:"drafts"`
	PrivatePosts     int `json:"private_posts"`
	Categories       int `json:"categories"`
	Tags             int `json:"tags"`
	Users            int `json:"users"`
	ApprovedComments int `json:"approved_comments"`
}

type UploadsStats struct {
	Exists                bool   `json:"exists"`
	Size                  string `json:"size"`
	FileCount             int    `json:"file_count"`
	PostsWithUploadsPaths int    `json:"posts_with_uploads_paths"`
	PostsWithHTTPURLs     int    `json:"posts_with_http_urls"`
}

type ThemeStats struct {
	ActiveTheme           string `json:"active_theme"`
	PHPFiles              int    `json:"php_files"`
	CSSFiles              int    `json:"css_files"`
	JSFiles               int    `json:"js_files"`
	PageTemplates         int    `json:"page_templates"`
	HookLikeOccurrences   int    `json:"hook_like_occurrences"`
	JQueryLikeOccurrences int    `json:"jquery_like_occurrences"`
}

type PluginsStats struct {
	ActiveCount     int  `json:"active_count"`
	HasACF          bool `json:"has_acf"`
	HasWooCommerce  bool `json:"has_woocommerce"`
	HasSEO          bool `json:"has_seo"`
	HasForm         bool `json:"has_form"`
	HasRedirect     bool `json:"has_redirect"`
	HasMember       bool `json:"has_member"`
	HasMultilingual bool `json:"has_multilingual"`
	HasCache        bool `json:"has_cache"`
}

type CustomStats struct {
	CustomPostTypeCount            int `json:"custom_post_type_count"`
	CustomTaxonomyCount            int `json:"custom_taxonomy_count"`
	MUPluginCount                  int `json:"mu_plugin_count"`
	MUPluginHookLikeOccurrences    int `json:"mu_plugin_hook_like_occurrences"`
	ShortcodePostCount             int `json:"shortcode_post_count"`
	SEOMetaCount                   int `json:"seo_meta_count"`
	SerializedMetaCount            int `json:"serialized_meta_count"`
	HtaccessRedirectLikeLines      int `json:"htaccess_redirect_like_lines"`
	CodeRedirectLikeOccurrences    int `json:"code_redirect_like_occurrences"`
	ExternalIntegrationOccurrences int `json:"external_integration_like_occurrences"`
}
