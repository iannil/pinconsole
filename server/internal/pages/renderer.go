// Package pages 提供拖拽编辑器落地页的 Go SSR 渲染引擎（page-editor pe-1）。
//
// 使用 Go html/template 将 PageSchema JSON 渲染为完整 HTML 页面。
// 所有模板通过 //go:embed 嵌入二进制，与单二进制部署架构一致。
package pages

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

// RenderData 是 page.gohtml 模板的数据上下文。
type RenderData struct {
	Lang             string
	Schema           PageSchema
	FontFamily       string
	HeroAlign        string
	SDKScript        string
	SectionsRendered []template.HTML // 预渲染的 section HTML
}

// PageSchema 是 JSON schema 的 Go 映射（与 TS 类型对应）。
type PageSchema struct {
	Meta     PageMeta  `json:"meta"`
	Style    PageStyle `json:"style"`
	Sections []Section `json:"sections"`
}

type PageMeta struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	OgImage     string `json:"og_image,omitempty"`
}

type PageStyle struct {
	FontFamily   string `json:"font_family,omitempty"`
	PrimaryColor string `json:"primary_color"`
	Background   string `json:"background"`
}

// Section 是页面中的一个组件块。
type Section struct {
	ID    string              `json:"id"`
	Type  string              `json:"type"`
	Props map[string]any      `json:"props"`
	Style map[string]string   `json:"style,omitempty"`
}

// Renderer 缓存已解析的模板。
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer 加载并解析全部 Go 模板。
func NewRenderer() (*Renderer, error) {
	t := template.New("").Funcs(template.FuncMap{
		"json": func(v any) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"toHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	})

	// 加载所有模板文件
	if err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".gohtml") {
			return nil
		}
		name := filepath.Base(path)
		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", name, err)
		}
		if _, err := t.New(name).Parse(string(content)); err != nil {
			return fmt.Errorf("parse template %s: %w", name, err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk templates: %w", err)
	}

	return &Renderer{tmpl: t}, nil
}

// Render 将 PageSchema 渲染为完整 HTML。
func (r *Renderer) Render(w io.Writer, schema PageSchema, opts RenderOptions) error {
	if r.tmpl == nil {
		return fmt.Errorf("renderer not initialized")
	}

	fontFamily := schema.Style.FontFamily
	if fontFamily == "" {
		fontFamily = "system-ui, sans-serif"
	}

	heroAlign := "center"

	// 预处理 + 预渲染所有 sections
	var rendered []template.HTML
	for _, s := range schema.Sections {
		// hero 对齐检测
		if s.Type == "hero" {
			if align, ok := s.Props["align"].(string); ok {
				heroAlign = align
			}
		}
		// 为 form 注入提交 action URL
		if s.Type == "form" && opts.PublicURL != "" {
			if s.Props == nil {
				s.Props = make(map[string]any)
			}
			s.Props["action"] = opts.PublicURL + "/api/pages/" + opts.Slug + "/form"
		}
		// columns：预渲染子节
		if s.Type == "columns" {
			s = r.preRenderColumns(s)
		}

		// 渲染此 section
		var buf bytes.Buffer
		if err := r.tmpl.ExecuteTemplate(&buf, s.Type, s); err != nil {
			return fmt.Errorf("render section %s (%s): %w", s.ID, s.Type, err)
		}
		rendered = append(rendered, template.HTML(buf.String()))
	}

	data := RenderData{
		Lang:             opts.Lang,
		Schema:           schema,
		FontFamily:       fontFamily,
		HeroAlign:        heroAlign,
		SDKScript:        opts.SDKScript,
		SectionsRendered: rendered,
	}

	return r.tmpl.ExecuteTemplate(w, "page.gohtml", data)
}

// preRenderColumns 递归预渲染 columns 的子节。
func (r *Renderer) preRenderColumns(s Section) Section {
	cols, ok := columnListFromProps(s.Props)
	if !ok {
		return s
	}
	for ci, col := range cols {
		var childHTML bytes.Buffer
		for _, child := range col.Children {
			if err := r.tmpl.ExecuteTemplate(&childHTML, child.Type, child); err != nil {
				// 跳过渲染失败的子节
				continue
			}
		}
		cols[ci].Rendered = childHTML.String()
	}
	if s.Props == nil {
		s.Props = make(map[string]any)
	}
	s.Props["columns"] = cols
	return s
}

// columnItem 对应 columns schema 中的单个列。
type columnItem struct {
	Width    int       `json:"width"`
	Children []Section `json:"sections"`
	Rendered string    `json:"rendered,omitempty"`
}

// columnListFromProps 从 columns 的 props 中提取列列表。
func columnListFromProps(props map[string]any) ([]columnItem, bool) {
	raw, ok := props["columns"]
	if !ok {
		return nil, false
	}
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, false
	}
	var cols []columnItem
	if err := json.Unmarshal(rawJSON, &cols); err != nil {
		return nil, false
	}
	return cols, true
}

// RenderOptions 控制渲染行为的参数。
type RenderOptions struct {
	Lang     string // 页面语言，默认 "zh"
	Slug     string // 页面 slug，用于表单提交
	PublicURL string // 公开访问的基础 URL（不含 /p/:slug）
	SDKScript string // SDK 脚本 URL，为空不注入
}
