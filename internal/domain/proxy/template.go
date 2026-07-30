package proxy

import (
	"fmt"
	"time"
)

func newRuleID() string {
	return fmt.Sprintf("rule_%d", time.Now().UnixNano())
}

// RuleTemplate bundles a set of rules that users can apply as a group.
type RuleTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // streaming, ai, social, gaming, work, china
	Icon        string `json:"icon"`
	Rules       []Rule `json:"rules"`
}

// PredefinedTemplates returns the built-in rule templates.
func PredefinedTemplates() []RuleTemplate {
	now := time.Now()
	return []RuleTemplate{
		{
			ID: "ai-openai", Name: "OpenAI / ChatGPT", Category: "ai",
			Description: "路由 OpenAI 和 ChatGPT 流量到代理",
			Icon:        "brain",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "openai.com", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 10, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "chatgpt.com", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 11, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "auth0.com", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 12, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "oaistatic.com", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 13, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "claude.ai", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 14, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "anthropic.com", TargetGroup: "PROXY", Enabled: true, Category: "ai", TemplateID: "ai-openai", Priority: 15, CreatedAt: now},
			},
		},
		{
			ID: "streaming-netflix", Name: "Netflix", Category: "streaming",
			Description: "路由 Netflix 流量到代理",
			Icon:        "tv",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "netflix.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-netflix", Priority: 20, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "nflxvideo.net", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-netflix", Priority: 21, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "nflxext.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-netflix", Priority: 22, CreatedAt: now},
			},
		},
		{
			ID: "streaming-youtube", Name: "YouTube", Category: "streaming",
			Description: "路由 YouTube 流量到代理",
			Icon:        "play",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "youtube.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-youtube", Priority: 30, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "googlevideo.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-youtube", Priority: 31, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "ytimg.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-youtube", Priority: 32, CreatedAt: now},
			},
		},
		{
			ID: "streaming-disney", Name: "Disney+", Category: "streaming",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "disneyplus.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-disney", Priority: 40, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "disney.com", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-disney", Priority: 41, CreatedAt: now},
			},
		},
		{
			ID: "streaming-abema", Name: "Abema TV", Category: "streaming",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "abema.tv", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-abema", Priority: 50, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "abema.io", TargetGroup: "PROXY", Enabled: true, Category: "streaming", TemplateID: "streaming-abema", Priority: 51, CreatedAt: now},
			},
		},
		{
			ID: "social-telegram", Name: "Telegram", Category: "social",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "telegram.org", TargetGroup: "PROXY", Enabled: true, Category: "social", TemplateID: "social-telegram", Priority: 60, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "t.me", TargetGroup: "PROXY", Enabled: true, Category: "social", TemplateID: "social-telegram", Priority: 61, CreatedAt: now},
			},
		},
		{
			ID: "social-twitter", Name: "X / Twitter", Category: "social",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "twitter.com", TargetGroup: "PROXY", Enabled: true, Category: "social", TemplateID: "social-twitter", Priority: 70, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "x.com", TargetGroup: "PROXY", Enabled: true, Category: "social", TemplateID: "social-twitter", Priority: 71, CreatedAt: now},
			},
		},
		{
			ID: "work-github", Name: "GitHub", Category: "work",
			Rules: []Rule{
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "github.com", TargetGroup: "PROXY", Enabled: true, Category: "work", TemplateID: "work-github", Priority: 80, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "githubusercontent.com", TargetGroup: "PROXY", Enabled: true, Category: "work", TemplateID: "work-github", Priority: 81, CreatedAt: now},
			},
		},
		{
			ID: "china-direct", Name: "国内直连", Category: "china",
			Description: "中国网站直连，不经过代理",
			Icon:        "flag",
			Rules: []Rule{
				{ID: newRuleID(), Type: "GEOIP", Value: "CN", TargetGroup: "DIRECT", Enabled: true, Category: "china", TemplateID: "china-direct", Priority: 0, CreatedAt: now},
				{ID: newRuleID(), Type: "DOMAIN-SUFFIX", Value: "cn", TargetGroup: "DIRECT", Enabled: true, Category: "china", TemplateID: "china-direct", Priority: 1, CreatedAt: now},
			},
		},
	}
}
