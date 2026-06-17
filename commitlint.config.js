// commitlint 配置：强制 conventional commits
// 详见 docs/standards/naming-conventions.md §6
export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',     // 新功能
        'fix',      // bug 修复
        'docs',     // 仅文档
        'style',    // 不影响代码语义的变更（空格、格式、分号等）
        'refactor', // 既不是 feat 也不是 fix
        'perf',     // 提升性能
        'test',     // 添加/修改测试
        'build',    // 影响构建系统或依赖的变更
        'ci',       // CI 配置
        'chore',    // 不属于以上类别的杂项
        'revert',   // 回滚之前的 commit
      ],
    ],
    'subject-case': [0], // 允许中文 subject
    'header-max-length': [2, 'always', 100],
  },
};
