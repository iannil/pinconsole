<script setup lang="ts">
// pe-2: PageEditorView — 拖拽式落地页编辑器
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { VueDraggable } from 'vue-draggable-plus';
import {
  PhArrowLeft, PhFloppyDisk, PhEye, PhTrash, PhPlus,
} from '@phosphor-icons/vue';
import type { Section, PageSchema, SectionType } from '@pinconsole/proto';
import { usePagesStore } from '../stores/pages';

const props = defineProps<{ slug: string }>();
const { t } = useI18n();
const router = useRouter();
const store = usePagesStore();

// ── state ──
const sections = ref<Section[]>([]);
const selectedIdx = ref<number | null>(null);
const dirty = ref(false);
const selectedSection = computed(() =>
  selectedIdx.value !== null ? sections.value[selectedIdx.value] : null
);

// ── 组件面板 ──
interface ComponentDef {
  type: SectionType;
  label: string;
  defaultProps: Record<string, any>;
}

const componentPalette: ComponentDef[] = [
  { type: 'hero', label: t('pages.hero'), defaultProps: { title: '标题', subtitle: '副标题', cta_label: '开始', cta_url: '', align: 'center' } },
  { type: 'text', label: t('pages.text'), defaultProps: { content: '文本内容' } },
  { type: 'image', label: t('pages.image'), defaultProps: { src: 'https://placehold.co/800x400', alt: '', max_width: '100%', align: 'center' } },
  { type: 'button', label: t('pages.button'), defaultProps: { label: '按钮', url: '', size: 'md' } },
  { type: 'features', label: t('pages.features'), defaultProps: { items: [{ icon: '✨', title: '特性1', description: '描述' }], columns: 3 } },
  { type: 'form', label: t('pages.form'), defaultProps: { fields: [{ name: 'email', type: 'email', label: '邮箱', required: true }], submit_label: '提交', submit_action: 'store' } },
  { type: 'divider', label: t('pages.divider'), defaultProps: {} },
  { type: 'spacer', label: t('pages.spacer'), defaultProps: { height: 48 } },
  { type: 'columns', label: t('pages.columns'), defaultProps: { columns: [{ width: 1, sections: [] }, { width: 1, sections: [] }], gap: 24 } },
];

// ── 生命周期 ──
onMounted(async () => {
  try {
    await store.loadPage(props.slug);
    if (store.current) {
      const schema = store.current.schema;
      sections.value = schema?.sections || [];
    }
  } catch {
    router.push('/pages');
  }
});

// ── 节操 ──
function addSection(def: ComponentDef) {
  const id = `s-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
  sections.value.push({ id, type: def.type, props: JSON.parse(JSON.stringify(def.defaultProps)) });
  selectedIdx.value = sections.value.length - 1;
  dirty.value = true;
}

function removeSection(idx: number) {
  sections.value.splice(idx, 1);
  if (selectedIdx.value === idx) selectedIdx.value = null;
  else if (selectedIdx.value !== null && selectedIdx.value > idx) selectedIdx.value--;
  dirty.value = true;
}

function selectSection(idx: number) {
  selectedIdx.value = idx;
}

function updateProp(key: string, value: any) {
  if (selectedIdx.value === null || !selectedSection.value) return;
  (selectedSection.value.props as Record<string, any>)[key] = value;
  dirty.value = true;
}

function updateNestedProp(keys: string[], value: any) {
  if (selectedIdx.value === null || !selectedSection.value) return;
  const props = selectedSection.value.props as Record<string, any>;
  let target = props;
  for (let i = 0; i < keys.length - 1; i++) {
    target = target[keys[i]];
  }
  target[keys[keys.length - 1]] = value;
  dirty.value = true;
}

function addArrayItem(key: string, template: Record<string, any>) {
  if (selectedIdx.value === null || !selectedSection.value) return;
  const arr = ((selectedSection.value.props as Record<string, any>)[key]) as any[];
  arr.push(JSON.parse(JSON.stringify(template)));
  dirty.value = true;
}

function removeArrayItem(key: string, idx: number) {
  if (selectedIdx.value === null || !selectedSection.value) return;
  const arr = ((selectedSection.value.props as Record<string, any>)[key]) as any[];
  arr.splice(idx, 1);
  dirty.value = true;
}

async function save() {
  const meta = store.current?.schema?.meta || { title: store.current?.title || props.slug };
  const style = store.current?.schema?.style || { primary_color: '#0f766e', background: '#ffffff' };
  const schema: PageSchema = { meta, style, sections: sections.value };
  await store.saveSchema(props.slug, schema);
  dirty.value = false;
}

async function handlePublish() {
  const target = store.current?.status === 'published' ? 'draft' : 'published';
  await store.publish(props.slug, target);
}

function handlePreview() {
  window.open(`/p/${props.slug}`, '_blank');
}

function goBack() {
  router.push('/pages');
}

// ── 属性渲染辅助 ──
const selectFields: Record<string, string[]> = {
  align: ['left', 'center'],
  size: ['sm', 'md', 'lg'],
  submit_action: ['store', 'redirect'],
  columns: ['2', '3', '4'],
};

const colorFields = new Set(['primary_color', 'color', 'background', 'text_color', 'bubble_color', 'header_color']);

function isSelectField(key: string): boolean { return key in selectFields; }
function isColorField(key: string): boolean { return colorFields.has(key); }

function getSectionTitle(s: Section): string {
  if (s.type === 'hero' && s.props?.title) return String(s.props.title).slice(0, 30);
  if (s.type === 'text' && s.props?.content) return String(s.props.content).slice(0, 30);
  if (s.type === 'button' && s.props?.label) return String(s.props.label).slice(0, 20);
  return s.type;
}

function sectionIcon(s: Section): string {
  const icons: Record<string, string> = {
    hero: '🏠', text: '📝', image: '🖼', button: '🔘',
    features: '✨', form: '📋', divider: '➖', spacer: '⏹', columns: '📐',
  };
  return icons[s.type] || '📄';
}
</script>

<template>
  <div class="editor">
    <!-- 顶栏 -->
    <header class="editor-topbar">
      <div class="topbar-left">
        <button class="btn-text" @click="goBack"><PhArrowLeft :size="18" /> {{ store.current?.title || props.slug }}</button>
        <span class="status-badge" :class="store.current?.status === 'published' ? 'published' : 'draft'">
          {{ t(`pages.${store.current?.status || 'draft'}`) }}
        </span>
      </div>
      <div class="topbar-right">
        <button class="btn-text" @click="handlePreview"><PhEye :size="16" /> {{ t('pages.preview') }}</button>
        <button class="btn-text" @click="handlePublish">
          {{ store.current?.status === 'published' ? t('pages.unpublish') : t('pages.publish') }}
        </button>
        <button class="btn btn-primary" :disabled="!dirty || store.saving" @click="save">
          <PhFloppyDisk :size="16" /> {{ store.saving ? t('pages.saving') : t('pages.save') }}
        </button>
      </div>
    </header>

    <div class="editor-body">
      <!-- 左侧：组件面板 -->
      <aside class="panel panel-left">
        <h3 class="panel-title">{{ t('pages.components') }}</h3>
        <div class="component-list">
          <button
            v-for="def in componentPalette"
            :key="def.type"
            class="component-item"
            @click="addSection(def)"
          >
            <PhPlus :size="14" />
            {{ def.label }}
          </button>
        </div>
      </aside>

      <!-- 中间：画布 -->
      <main class="canvas">
        <div v-if="sections.length === 0" class="canvas-empty">
          <p>{{ t('pages.no_pages') }}</p>
        </div>
        <VueDraggable
          v-model="sections"
          class="canvas-list"
          ghost-class="ghost"
          :animation="200"
          handle=".drag-handle"
          @change="dirty = true"
        >
          <div
            v-for="(section, idx) in sections"
            :key="section.id"
            class="canvas-section"
            :class="{ selected: selectedIdx === idx }"
            @click="selectSection(idx)"
          >
            <div class="section-header">
              <span class="drag-handle">⠿</span>
              <span class="section-icon">{{ sectionIcon(section) }}</span>
              <span class="section-type">{{ section.type }}</span>
              <span class="section-title">{{ getSectionTitle(section) }}</span>
              <button class="btn-icon btn-icon-danger" @click.stop="removeSection(idx)" :title="t('pages.delete_section')">
                <PhTrash :size="14" />
              </button>
            </div>
          </div>
        </VueDraggable>
      </main>

      <!-- 右侧：属性编辑 -->
      <aside class="panel panel-right">
        <h3 class="panel-title">{{ t('pages.properties') }}</h3>
        <div v-if="!selectedSection" class="panel-empty">
          <p>← {{ t('pages.add_section') }}</p>
        </div>
        <div v-else class="props-form">
          <div
            v-for="(value, key) in (selectedSection.props as Record<string, any>)"
            :key="key"
            class="prop-field"
          >
            <label>{{ key }}</label>

            <!-- select enum -->
            <select
              v-if="isSelectField(key) && typeof value === 'string'"
              :value="value"
              class="input"
              @change="updateProp(key, ($event.target as HTMLSelectElement).value)"
            >
              <option v-for="opt in selectFields[key]" :key="opt" :value="opt">{{ opt }}</option>
            </select>

            <!-- color -->
            <div v-else-if="isColorField(key)" class="color-row">
              <input
                type="color"
                :value="value || '#000000'"
                @input="updateProp(key, ($event.target as HTMLInputElement).value)"
              />
              <input
                type="text"
                :value="value || ''"
                class="input"
                @input="updateProp(key, ($event.target as HTMLInputElement).value)"
              />
            </div>

            <!-- textarea for long text -->
            <textarea
              v-else-if="typeof value === 'string' && (key === 'content' || key === 'body' || key === 'description')"
              :value="value"
              class="input textarea"
              @input="updateProp(key, ($event.target as HTMLTextAreaElement).value)"
            ></textarea>

            <!-- string/number input -->
            <input
              v-else-if="typeof value === 'string' || typeof value === 'number'"
              :type="typeof value === 'number' ? 'number' : 'text'"
              :value="value"
              class="input"
              @input="updateProp(key, ($event.target as HTMLInputElement).value)"
            />

            <!-- boolean -->
            <input
              v-else-if="typeof value === 'boolean'"
              type="checkbox"
              :checked="value"
              class="checkbox"
              @change="updateProp(key, ($event.target as HTMLInputElement).checked)"
            />

            <!-- array of objects -->
            <div v-else-if="Array.isArray(value)" class="array-editor">
              <div v-for="(item, ai) in value" :key="ai" class="array-item">
                <div class="array-item-header">
                  <span>{{ key }} #{{ ai + 1 }}</span>
                  <button class="btn-icon btn-icon-danger" @click="removeArrayItem(key, ai)">
                    <PhTrash :size="12" />
                  </button>
                </div>
                <div v-for="(fv, fk) in item" :key="fk" class="array-field">
                  <label class="nested-label">{{ fk }}</label>
                  <input
                    v-if="typeof fv === 'string' || typeof fv === 'number'"
                    :type="typeof fv === 'number' ? 'number' : 'text'"
                    :value="fv"
                    class="input"
                    @input="updateNestedProp([key, ai, fk], ($event.target as HTMLInputElement).value)"
                  />
                  <input
                    v-else-if="typeof fv === 'boolean'"
                    type="checkbox"
                    :checked="fv"
                    @change="updateNestedProp([key, ai, fk], ($event.target as HTMLInputElement).checked)"
                  />
                  <span v-else class="prop-complex">{{ typeof fv }}</span>
                </div>
              </div>
              <!-- add item button for features items / form fields -->
              <button
                v-if="key === 'items' || key === 'fields'"
                class="btn-text btn-add"
                @click="addArrayItem(key, key === 'items' ? { icon: '', title: '', description: '' } : { name: '', type: 'text', label: '', required: false })"
              >
                <PhPlus :size="12" /> {{ t('pages.add_section') }}
              </button>
            </div>

            <!-- fallback -->
            <span v-else class="prop-complex">{{ typeof value }}</span>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.editor { display: flex; flex-direction: column; height: calc(100vh - 56px); }
.editor-topbar { display: flex; justify-content: space-between; align-items: center; padding: 8px 16px; border-bottom: 1px solid var(--color-border-default, #e7e5e4); background: var(--color-bg-surface, #fff); flex-shrink: 0; }
.topbar-left, .topbar-right { display: flex; align-items: center; gap: 8px; }
.btn-text { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border: none; background: transparent; cursor: pointer; border-radius: 6px; font-size: 14px; font-family: inherit; color: var(--color-text-secondary, #57534e); }
.btn-text:hover { background: var(--color-bg-subtle, #f5f1ec); }
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 6px 14px; border-radius: 8px; font-size: 14px; font-weight: 500; cursor: pointer; border: none; font-family: inherit; }
.btn-primary { background: var(--color-accent-default, #0f766e); color: #fff; }
.btn-primary:hover { opacity: .9; }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-icon { display: inline-flex; align-items: center; justify-content: center; padding: 4px; border: none; background: transparent; cursor: pointer; border-radius: 4px; color: var(--color-text-muted, #78716c); }
.btn-icon-danger:hover { color: #dc2626; }
.btn-add { font-size: 12px; padding: 4px 8px; margin-top: 4px; }
.status-badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 500; }
.status-badge.published { background: #d1fae5; color: #065f46; }
.status-badge.draft { background: #f5f5f4; color: #44403c; }
.editor-body { display: flex; flex: 1; overflow: hidden; }
.panel { width: 260px; flex-shrink: 0; overflow-y: auto; border-right: 1px solid var(--color-border-default, #e7e5e4); padding: 16px; }
.panel-right { border-right: none; border-left: 1px solid var(--color-border-default, #e7e5e4); }
.panel-title { font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: .5px; color: var(--color-text-muted, #78716c); margin: 0 0 12px; }
.component-list { display: flex; flex-direction: column; gap: 4px; }
.component-item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 8px; background: var(--color-bg-surface, #fff); cursor: pointer; font-size: 13px; font-family: inherit; color: var(--color-text-primary, #1c1917); }
.component-item:hover { border-color: var(--color-accent-default, #0f766e); background: var(--color-bg-subtle, #f5f1ec); }
.canvas { flex: 1; overflow-y: auto; padding: 24px; background: #fafaf9; }
.canvas-empty { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-muted, #78716c); }
.canvas-list { display: flex; flex-direction: column; gap: 6px; min-height: 200px; }
.canvas-section { background: #fff; border: 2px solid transparent; border-radius: 8px; padding: 10px 14px; cursor: pointer; box-shadow: 0 1px 3px rgba(0,0,0,.06); }
.canvas-section:hover { border-color: var(--color-border-default, #e7e5e4); }
.canvas-section.selected { border-color: var(--color-accent-default, #0f766e); }
.section-header { display: flex; align-items: center; gap: 8px; }
.drag-handle { cursor: grab; color: var(--color-text-muted, #78716c); font-size: 16px; user-select: none; }
.section-icon { font-size: 16px; }
.section-type { font-size: 12px; font-weight: 600; color: var(--color-accent-default, #0f766e); text-transform: uppercase; }
.section-title { font-size: 13px; color: var(--color-text-primary, #1c1917); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.ghost { opacity: .4; background: var(--color-bg-subtle, #f5f1ec); }
.panel-empty { padding: 24px 0; text-align: center; color: var(--color-text-muted, #78716c); font-size: 13px; }
.props-form { display: flex; flex-direction: column; gap: 10px; }
.prop-field { display: flex; flex-direction: column; gap: 4px; }
.prop-field > label { font-size: 11px; font-weight: 600; color: var(--color-text-muted, #78716c); text-transform: uppercase; letter-spacing: .3px; }
.input { width: 100%; padding: 6px 10px; border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 6px; font-size: 13px; font-family: inherit; background: var(--color-bg-surface, #fff); }
.input:focus { outline: none; border-color: var(--color-accent-default, #0f766e); }
select.input { cursor: pointer; }
.textarea { min-height: 80px; resize: vertical; }
.color-row { display: flex; gap: 6px; align-items: center; }
.color-row input[type="color"] { width: 32px; height: 32px; padding: 2px; border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 6px; cursor: pointer; }
.color-row .input { flex: 1; }
.checkbox { width: 18px; height: 18px; cursor: pointer; }
.array-editor { display: flex; flex-direction: column; gap: 8px; }
.array-item { border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 6px; padding: 8px; background: var(--color-bg-subtle, #f5f1ec); }
.array-item-header { display: flex; justify-content: space-between; align-items: center; font-size: 11px; font-weight: 600; color: var(--color-text-secondary, #57534e); margin-bottom: 6px; }
.array-field { margin-bottom: 4px; }
.nested-label { font-size: 11px; color: var(--color-text-muted, #78716c); display: block; margin-bottom: 1px; }
.prop-complex { font-size: 12px; color: var(--color-text-muted, #78716c); font-style: italic; }
</style>
