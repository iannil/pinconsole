# Slice 0 implementation:.env rename + docker 自动建库

> **状态**:completed
> **对应 spec**:[spec](./2026-06-20-slice-0-env-rename-fix-spec.md)
> **深度 badge**:🟢 touched(配置修复 + 实测验证 docker 集成测试可重复跑)
> **实际工时**:~1h(预估 2h)
> **Disclaimer**:本切片只验证"docker 启动 + 测试连接 + 覆盖率显示真实值",不验证业务逻辑正确性。

## 实施清单

### 1. `.env`(根目录)

| 行 | 改前 | 改后 |
|---|---|---|
| 1 | `# marketing-monitor 本地开发环境配置` | `# pinconsole 本地开发环境配置` |
| 15 | `PG_DB=marketing_monitor` | `PG_DB=pinconsole` |
| 32 | `MINIO_BUCKET=marketing-monitor` | `MINIO_BUCKET=pinconsole` |

### 2. `.env.example`

| 行 | 改前 | 改后 |
|---|---|---|
| 1 | `# marketing-monitor 本地开发环境配置` | `# pinconsole 本地开发环境配置` |
| 25 | `ADMIN_EMAIL=admin@marketing-monitor.local` | `ADMIN_EMAIL=admin@pinconsole.local` |
| 43 | `PG_DB=marketing_monitor` | `PG_DB=pinconsole` |
| 61 | `MINIO_BUCKET=marketing-monitor` | `MINIO_BUCKET=pinconsole` |

### 3. `Makefile` — **保持原样,未修改**

切片前预估"新建 Makefile",实测发现项目已有完善 Makefile(切片 1a 时建立,183 行,含 docker-up / docker-down / test-go / test-js / test-e2e / lint / migrate-up(deprecated)/ migrate-up-legacy / build / install-tools / clean 等全套 target)。

**反思**:plan 阶段误判"Makefile 不存在"。实际验证 `make help` 输出 30+ target,基础设施完备。本切片**无需新建**,后续切片(Go-1~7 / TS-1~5 / CI-1)直接用 `make test-go` / `make docker-up` 即可。

后续 CI-1 切片可考虑把 CI workflow 切换到 `make` 命令统一入口,但不在本切片范围。

### 4. `README.md`

- "快速开始"段改用 `make docker-up` / `make build-frontend` / `make build`,加 "Make 命令清单"提示
- 新增"老开发者迁移"段:解释 `.env` 改了但本地 PG volume 有 `marketing_monitor` 数据库的情况,提供两种迁移方式(清空 volume / 手动 CREATE DATABASE)

## 验证

```bash
# 1. 干净重启 docker(删 volume,确保新 .env 生效)
docker compose down -v
make docker-up   # 或 docker compose up -d postgres redis minio

# 2. 验证 PG 自动建了 pinconsole 数据库
docker compose exec -T postgres psql -U mm -l | grep pinconsole
# 预期:pinconsole 数据库存在,marketing_monitor 不存在

# 3. 应用 migrations(server 启动时自动跑,或测试场景手动 psql)
for f in server/migrations/*.up.sql; do
  docker compose exec -T postgres psql -U mm -d pinconsole < "$f"
done

# 4. 跑 Go 测试
make test-go
# 预期:全绿

# 5. 覆盖率验证
cd server && go test -count=1 -cover ./...
# 预期:逐包覆盖率显示真实数字(storage 57.6% 而非 1.5%)
```

**实测结果**(2026-06-20):

- ✅ `docker compose down -v && up -d` 后 PG 自动建 pinconsole 数据库(postgres:16-alpine 标准行为)
- ✅ 7 张表通过手动 psql 应用 5 个 migration 文件成功建好
- ✅ `make test-go` 全绿(11 个包 + cmd/server + migrations)
- ✅ 覆盖率显示 storage **57.6%** / api 38.2% / antiscrape 86.7% / recording 48.0% 等真实数字(与 [`audits/2026-06-20-coverage-assessment.md`](../audits/2026-06-20-coverage-assessment.md) §2 完全一致)

## 实测的逐包覆盖率(本切片验证后)

| 包 | 覆盖率 | 评级 |
|---|---|---|
| config | 98.0% | 🟢 |
| privacy | 95.0% | 🟢 |
| proto | 88.9% | 🟢 |
| antiscrape | 86.7% | 🟢 |
| observability | 83.3% | 🟢 |
| logging | 79.6% | 🟢 |
| hub | 73.0% | 🟢 |
| storage | 57.6% | 🟡 |
| recording | 48.0% | 🟡 |
| api | 38.2% | 🟡 |
| cmd/server | 4.9% | 🔴(豁免) |

(数据与 [`audits/2026-06-20-coverage-assessment.md`](../audits/2026-06-20-coverage-assessment.md) §2 完全一致,证明 audit 数据可信。)

## 副作用与回归

- ✅ `docker compose down -v` 后再 `up -d`,PG 自动建 pinconsole 数据库(postgres:16-alpine 标准行为)
- ✅ 测试代码 `erasure_test.go:34` 硬编码 `localhost:5432/pinconsole` 现在与 docker 实际数据库名一致,无需改测试代码
- ✅ Makefile 新建不破坏现有 CI(CI 仍用 raw `go test` 命令;CI-1 切片再统一切换)
- ⚠️ 老开发者本地 `marketing_monitor` 数据库需手动迁移(README 已说明)

## 后续切片解锁

本切片完成,以下切片可启动:

- **Go-4**(storage 适配器补测)— 依赖真实 PG/Redis/MinIO,本切片前在干净 docker 上必失败
- **Go-5**(recording 包补测)— 同上,需 MinIO 集成
- **Go-6**(api 包补测)— 依赖 hub/storage/recording,通过 hub 间接依赖本切片

Go-1/Go-2/Go-3 不依赖本切片(纯单测或仅 mock),可并行启动。

## 同步文档

- ✅ `docs/progress/` → 完成后移到 `docs/reports/completed/`(spec + impl 一起)
- ✅ `memory/daily/2026-06-20.md` 追加切片 0 记录
- ✅ `docs/project-status.md` §2.1 不需要改(覆盖率数字本来就是实测口径)
