<script setup lang="ts">
// pe-2: PageEditorView — 拖拽式落地页编辑器
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { VueDraggable } from 'vue-draggable-plus';
import {
  PhArrowLeft, PhFloppyDisk, PhEye, PhTrash, PhPlus,
  PhHouse, PhTextT, PhImageSquare, PhCursorClick,
  PhStar, PhClipboardText, PhMinus, PhArrowsVertical,
  PhLayout, PhFile,
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
  { type: 'features', label: t('pages.features'), defaultProps: { items: [{ icon: 'star', title: '特性1', description: '描述' }], columns: 3 } },
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

// ── section icon map (Phosphor, no emoji) ──
const sectionIcons: Record<string, any> = {
  hero: PhHouse,
  text: PhTextT,
  image: PhImageSquare,
  button: PhCursorClick,
  features: PhStar,
  form: PhClipboardText,
  divider: PhMinus,
  spacer: PhArrowsVertical,
  columns: PhLayout,
};

function getSectionTitle(s: Section): string {
  if (s.type === 'hero' && s.props?.title) return String(s.props.title).slice(0, 30);
  if (s.type === 'text' && s.props?.content) return String(s.props.content).slice(0, 30);
  if (s.type === 'button' && s.props?.label) return String(s.props.label).slice(0, 20);
  return s.type;
}


</script>

<template>
  <div class="editor">
    <!-- 顶栏 -->
    <header class="editor-topbar">
      <div class="topbar-left">
        <button class="pc-btn pc-btn--ghost" @click="goBack"><PhArrowLeft :size="18" /> {{ store.current?.title || props.slug }}</button>
        <span :class="store.current?.status === 'published' ? 'pc-badge pc-badge--success' : 'pc-badge pc-badge--accent'">
          {{ t(`pages.${store.current?.status || 'draft'}`) }}
        </span>
      </div>
      <div class="topbar-right">
        <button class="pc-btn pc-btn--ghost" @click="handlePreview"><PhEye :size="16" /> {{ t('pages.preview') }}</button>
        <button class="pc-btn pc-btn--ghost" @click="handlePublish">
          {{ store.current?.status === 'published' ? t('pages.unpublish') : t('pages.publish') }}
        </button>
        <button class="pc-btn pc-btn--primary" :disabled="!dirty || store.saving" @click="save">
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
              <span class="section-icon"><component :is="sectionIcons[section.type] || PhFile" :size="14" /></span>
              <span class="section-type">{{ section.type }}</span>
              <span class="section-title">{{ getSectionTitle(section) }}</span>
              <button class="pc-btn pc-btn--icon" @click.stop="removeSection(idx)" :title="t('pages.delete_section')">
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
                  <button class="pc-btn pc-btn--icon" @click="removeArrayItem(key, ai)">
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
                class="pc-btn pc-btn--ghost add-item-btn"
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
.editor { display: flex; flex-direction: column; height: calc(100vh - var(--pc-topbar-height)); }
.editor-topbar { display: flex; justify-content: space-between; align-items: center; padding: 8px var(--pc-space-card); border-bottom: 1px solid var(--pc-color-border-default); background: var(--pc-color-bg-surface); flex-shrink: 0; }
.topbar-left, .topbar-right { display: flex; align-items: center; gap: var(--pc-space-field); }
.add-item-btn { font-size: var(--pc-text-xs); padding: 4px var(--pc-space-field); margin-top: 4px; }
.editor-body { display: flex; flex: 1; overflow: hidden; }
.panel { width: 260px; flex-shrink: 0; overflow-y: auto; border-right: 1px solid var(--pc-color-border-default); padding: var(--pc-space-card); }
.panel-right { border-right: none; border-left: 1px solid var(--pc-color-border-default); }
.panel-title { font-size: var(--pc-text-sm); font-weight: var(--pc-weight-semibold); text-transform: uppercase; letter-spacing: .5px; color: var(--pc-color-text-muted); margin: 0 0 var(--pc-space-component); }
.component-list { display: flex; flex-direction: column; gap: 4px; }
.component-item { display: flex; align-items: center; gap: var(--pc-space-field); padding: 8px var(--pc-space-component); border: 1px solid var(--pc-color-border-default); border-radius: var(--pc-radius-md); background: var(--pc-color-bg-surface); cursor: pointer; font-size: var(--pc-text-sm); font-family: inherit; color: var(--pc-color-text-primary); }
.component-item:hover { border-color: var(--pc-color-accent-default); background: var(--pc-color-bg-subtle); }
.canvas { flex: 1; overflow-y: auto; padding: var(--pc-space-section); background: var(--pc-color-bg-canvas); }
.canvas-empty { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--pc-color-text-muted); }
.canvas-list { display: flex; flex-direction: column; gap: 6px; min-height: 200px; }
.canvas-section { background: var(--pc-color-bg-surface); border: 2px solid transparent; border-radius: var(--pc-radius-md); padding: 10px 14px; cursor: pointer; box-shadow: var(--pc-shadow-xs); }
.canvas-section:hover { border-color: var(--pc-color-border-default); }
.canvas-section.selected { border-color: var(--pc-color-accent-default); }
.section-header { display: flex; align-items: center; gap: var(--pc-space-field); }
.drag-handle { cursor: grab; color: var(--pc-color-text-muted); font-size: 16px; user-select: none; }
.section-icon { font-size: 16px; display: inline-flex; align-items: center; }
.section-type { font-size: var(--pc-text-xs); font-weight: var(--pc-weight-semibold); color: var(--pc-color-accent-default); text-transform: uppercase; }
.section-title { font-size: var(--pc-text-sm); color: var(--pc-color-text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.ghost { opacity: .4; background: var(--pc-color-bg-subtle); }
.panel-empty { padding: var(--pc-space-section) 0; text-align: center; color: var(--pc-color-text-muted); font-size: var(--pc-text-sm); }
.props-form { display: flex; flex-direction: column; gap: 10px; }
.prop-field { display: flex; flex-direction: column; gap: 4px; }
.prop-field > label { font-size: 11px; font-weight: var(--pc-weight-semibold); color: var(--pc-color-text-muted); text-transform: uppercase; letter-spacing: .3px; }
.input { width: 100%; padding: 6px 10px; border: 1px solid var(--pc-color-border-default); border-radius: var(--pc-radius-sm); font-size: var(--pc-text-sm); font-family: inherit; background: var(--pc-color-bg-surface); color: var(--pc-color-text-primary); }
.input:focus { outline: none; border-color: var(--pc-color-accent-default); box-shadow: var(--pc-focus-ring); }
select.input { cursor: pointer; }
.textarea { min-height: 80px; resize: vertical; }
.color-row { display: flex; gap: 6px; align-items: center; }
.color-row input[type="color"] { width: 32px; height: 32px; padding: 2px; border: 1px solid var(--pc-color-border-default); border-radius: var(--pc-radius-sm); cursor: pointer; }
.color-row .input { flex: 1; }
.checkbox { width: 18px; height: 18px; cursor: pointer; }
.array-editor { display: flex; flex-direction: column; gap: var(--pc-space-field); }
.array-item { border: 1px solid var(--pc-color-border-default); border-radius: var(--pc-radius-sm); padding: var(--pc-space-field); background: var(--pc-color-bg-subtle); }
.array-item-header { display: flex; justify-content: space-between; align-items: center; font-size: 11px; font-weight: var(--pc-weight-semibold); color: var(--pc-color-text-secondary); margin-bottom: 6px; }
.array-field { margin-bottom: 4px; }
.nested-label { font-size: 11px; color: var(--pc-color-text-muted); display: block; margin-bottom: 1px; }
.prop-complex { font-size: var(--pc-text-xs); color: var(--pc-color-text-muted); font-style: italic; }
</style>
