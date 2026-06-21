# Slice 0 spec:.env rename + docker 自动建库(覆盖率提升前置)

> **状态**:in_progress
> **工时预估**:2h
> **依赖**:无(后续 Go-4/5/6 硬依赖本切片)
> **深度目标**:🟢 touched(配置修复 + 验证 docker 集成测试可重复跑)

## 背景与动机

[`audits/2026-06-20-coverage-assessment.md`](../audits/2026-06-20-coverage-assessment.md) §6.5 发现 rename 重构 5 步(`f461b59..234fa06`)漏改 `.env` 和 `.env.example`:

- `PG_DB=marketing_monitor`(应 `pinconsole`)
- `MINIO_BUCKET=marketing-monitor`(应 `pinconsole`)
- 注释和 `ADMIN_EMAIL=admin@marketing-monitor.local`(应 `pinconsole.local`)

**实际影响**:

- `docker-compose.yml` 默认值是 `${PG_DB:-pinconsole}`(已对),但 `.env` 优先覆盖
- docker 启动的 PG 自动建 `marketing_monitor` 数据库
- 测试代码 `server/internal/storage/erasure_test.go:34` 硬编码 `localhost:5432/pinconsole`
- 名字不匹配 → 所有 storage/recording 集成测试在本地默认 `docker compose up` 后 skip → 显示假低覆盖率(如 storage 1.5% vs 真实 57.6%)

**为什么必须先修**:后续 Go-4(storage)/Go-5(recording)/Go-6(api) 三个大切片依赖 PG/Redis/MinIO 集成测试,不修则切片验收无法跑真实覆盖率。

## 范围

### 必做

1. **`.env`**:`PG_DB` / `MINIO_BUCKET` / 顶部注释改为 pinconsole
2. **`.env.example`**:同上 + `ADMIN_EMAIL` 改为 `admin@pinconsole.local`
3. **`README.md`**:在"快速开始"段加迁移说明(已有 marketing_monitor 数据的老开发者需手动迁移)+ 改用 `make docker-up` 等命令(现有 Makefile 已就绪)
4. **验证**:`docker compose down -v && docker compose up -d` + 手动应用 migrations + `make test-go` 全绿;覆盖率与 [`audits/2026-06-20-coverage-assessment.md`](../audits/2026-06-20-coverage-assessment.md) §2 一致

### 不做

- 不改测试代码硬编码(`erasure_test.go:34`)—— `.env` 改对后,新建数据库自动叫 `pinconsole`,测试可连
- 不写 init script—— `postgres:16-alpine` 启动时自动执行 `CREATE DATABASE ${POSTGRES_DB}`,无需额外脚本
- 不新建 Makefile—— **项目已有完善 Makefile(183 行,含 docker-up/test-go/migrate-up-legacy 等全套 target)**,无需重复造轮子
- 不批量清理已完成的 reports 中的 marketing_monitor 字面量(那是历史快照,2026-06-20 文档整理时已加注)

## 决策

| # | 决策 | 选择 | 理由 |
|---|---|---|---|
| 1 | 数据库迁移策略 | 仅修 .env,README 加说明 | 老开发者 pg-data volume 已有 marketing_monitor 数据,新 docker compose up 不会再建 pinconsole;README 说明手动 `CREATE DATABASE pinconsole` 或 `docker compose down -v` 重来 |
| 2 | Makefile vs scripts/ | Makefile | 项目根尚无 Makefile,创建一个统一入口比散在 scripts/ 更易发现;CI 也可直接 `make test-go` |
| 3 | test-go-integration target 设计 | 不用 build tag,默认全跑 | 项目当前没有 `//go:build integration` tag,所有测试默认依赖 docker;Makefile 仅做"启动 docker → 跑测试"的封装 |

## 验收

```bash
# 1. 干净重启 docker
docker compose down -v
docker compose up -d postgres redis minio

# 2. 等 PG ready
until docker compose exec -T postgres pg_isready -U mm 2>/dev/null | grep -q accepting; do sleep 1; done

# 3. PG 应自动建 pinconsole 数据库(不是 marketing_monitor)
docker compose exec -T postgres psql -U mm -l | grep pinconsole
# 预期:pinconsole 数据库存在,marketing_monitor 不存在

# 4. 跑 Go 测试
make test-go
# 预期:全绿,storage 包覆盖率 ≥ 57.6%

# 5. 覆盖率报告
make coverage-go
# 预期:逐包覆盖率与 audit §2 一致
```

## 风险

- 🟡 老开发者本地 pg-data volume 已有 marketing_monitor 数据,改 .env 后 docker 不会自动迁移。**缓解**:README 加一句"`docker compose down -v` 清理 volume 后重启,或手动 `psql -c 'CREATE DATABASE pinconsole OWNER mm'`"
- 🟢 Makefile 新建不会破坏现有 CI(CI 仍用 raw 命令);后续 CI-1 切片再统一切换到 `make`
