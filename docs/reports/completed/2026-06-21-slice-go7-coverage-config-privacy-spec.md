# Slice Go-7 spec:config + privacy 基线确认 + Phase 1 验收

> **状态**:completed
> **工时预估**:4h | **实际工时**:~1h
> **深度 badge**:🟢 touched(验证 ≥90% 不退步 + 补 buffer)

## 范围

| 包 | 切片前 | 切片后 |
|---|---|---|
| internal/config | 98.0% | **98.0%**(buffer 补 fail-secure 路径,百分比不变但覆盖深度提升) |
| internal/privacy | 95.0% | **95.0%**(同上) |

两个包已 ≥90%,本切片只补 buffer 测试不退步。

## 主要工作

**privacy**:
- `coverage_extra_test.go`:TruncateIP 8 个 case(empty/ipv4/ipv4+port/ipv6/ipv6+bracket/invalid/all-zero/full-form)

**config**:
- `coverage_extra_test.go`:Load 9 个 fail-secure 路径(dev OK / prod changeme / prod PG 默认 / prod MinIO 弱 key / prod MinIO 弱 secret / prod 远程 PG SSL=disable / prod 远程 MinIO UseSSL=false / BCRYPT<12 / BehindReverseProxy 无 TrustedProxies / Env typo)

## Phase 1 最终验收(Go phase)

### Go 包覆盖率汇总(2026-06-21)

| 包 | 切片前 | 切片后 | 达标 |
|---|---|---|---|
| config | 98.0% | 98.0% | ✅ |
| privacy | 95.0% | 95.0% | ✅ |
| proto | 88.9% | **100.0%** | ✅ Go-1 |
| antiscrape | 86.7% | **95.9%** | ✅ Go-1 |
| observability | 83.3% | **91.7%** | ✅ Go-1 |
| logging | 79.6% | **98.0%** | ✅ Go-2 |
| hub | 72.4% | **94.7%** | ✅ Go-3 |
| storage | 57.6% | **86.5%** | 🟡 Go-4(差 3.5pp) |
| recording | 48.0% | **77.7%** | 🟡 Go-5(差 12.3pp) |
| api | 38.2% | **47.9%** | 🟡 Go-6 Commit 1(差 42.1pp,WS handlers 留 backlog) |
| cmd/server | 4.9% | 4.9% | 豁免(业界惯例) |

### 达标率

- **7/10 包 ≥ 90%**(不含 cmd/server)
- **3/10 未达 90%**:storage / recording / api(留 R&D backlog)
- **加权整体覆盖率**:切片前 ~65% → 切片后 ~80%

### 累计工时

切片 0(1h) + Go-1(1.5h) + Go-2(1h) + Go-3(2h) + Go-4(2.5h) + Go-5(2h) + Go-6(2h) + Go-7(1h) = **~13h**
预估 60h,实际 13h(22% 完成度对应 78% 工作量,因为 Explore 报告已建立模式 + 现有 helper 复用)。

### 关键决策

- **3 个包未达 90%**:按 plan 风险点 #5"不为达 90% 写为测而测的弱断言测试",接受现状,留 backlog
- **api Commit 2 未做**:WS handlers 需要复杂集成测试,工时远超预估,留 backlog
- **不调低 90% 总门槛**:用户硬约束,文档显式标注未达包
