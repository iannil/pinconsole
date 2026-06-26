// Page Schema 类型（page-editor pe-1）
// JSON schema 定义单页落地页的结构，由拖拽编辑器生成。
// Go SSR 读取后渲染为 HTML。

// ── 组件类型枚举 ──────────────────────────────────────────

export type SectionType =
  | 'hero'
  | 'text'
  | 'image'
  | 'button'
  | 'features'
  | 'form'
  | 'divider'
  | 'spacer'
  | 'columns';

// ── 各组件 Props ──────────────────────────────────────────

export interface HeroProps {
  title: string;
  subtitle?: string;
  cta_label?: string;
  cta_url?: string;
  background?: string;     // CSS color
  text_color?: string;     // CSS color
  align?: 'left' | 'center';
}

export interface TextProps {
  content: string;         // HTML safe — rendered as textContent in SSR
  text_color?: string;
  background?: string;
}

export interface ImageProps {
  src: string;
  alt: string;
  max_width?: string;      // e.g. "600px"
  align?: 'left' | 'center' | 'right';
}

export interface ButtonProps {
  label: string;
  url: string;
  size?: 'sm' | 'md' | 'lg';
  color?: string;          // CSS color
  full_width?: boolean;
}

export interface FeatureItem {
  icon?: string;           // emoji or SVG URL
  title: string;
  description: string;
}

export interface FeaturesProps {
  items: FeatureItem[];
  columns?: 2 | 3 | 4;
  background?: string;
}

export interface FormField {
  name: string;
  type: 'text' | 'email' | 'tel' | 'textarea';
  label: string;
  required: boolean;
  placeholder?: string;
}

export interface FormProps {
  fields: FormField[];
  submit_label: string;
  submit_action: 'store' | 'redirect';
  redirect_url?: string;
  background?: string;
}

export interface DividerProps {
  color?: string;
  thickness?: number;      // px
  spacing?: number;        // px (上下间距)
}

export interface SpacerProps {
  height: number;          // px
}

export interface Column {
  sections: Section[];
  width?: number;          // flex ratio, default 1
}

export interface ColumnsProps {
  columns: Column[];
  background?: string;
  gap?: number;            // px
}

// ── Section 与 PageSchema ──────────────────────────────────

export interface Section {
  id: string;
  type: SectionType;
  props: HeroProps | TextProps | ImageProps | ButtonProps
       | FeaturesProps | FormProps | DividerProps | SpacerProps
       | ColumnsProps;
  style?: Record<string, string>; // CSS overrides
}

export interface PageMeta {
  title: string;
  description?: string;
  og_image?: string;
}

export interface PageStyle {
  font_family?: string;
  primary_color: string;
  background: string;
}

export interface PageSchema {
  meta: PageMeta;
  style: PageStyle;
  sections: Section[];
}

// ── API 类型 ──────────────────────────────────────────────

export interface PageResponse {
  id: number;
  tenant_id: string;
  slug: string;
  title: string;
  status: 'draft' | 'published';
  schema: PageSchema;
  created_at: string;
  updated_at: string;
}

export interface PageListItem {
  id: number;
  slug: string;
  title: string;
  status: 'draft' | 'published';
  updated_at: string;
}

export interface CreatePageRequest {
  title: string;
  slug?: string;           // 可选，不传则自动生成
}

export interface UpdatePageRequest {
  title?: string;
  schema?: PageSchema;
  status?: 'draft' | 'published';
}

export interface FormSubmission {
  id: number;
  page_slug: string;
  fields: Record<string, string>;
  created_at: string;
}
