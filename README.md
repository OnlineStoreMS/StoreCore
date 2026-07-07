# StoreCore — OSMS 门店管理系统

StoreCore 是 **OSMS（Offline Store Management System）** 平台下的门店管理应用，与 ProductCore、UserCore、SupplyCore 采用相同技术栈与部署方式。

## 功能模块

| 模块 | 说明 |
|------|------|
| 收银台 | 同步 ProductCore 商品/SKU，即时零售结算，多种支付方式，电子小票 |
| 销售订单 | 线下顾客订货、提货、送货上门、发快递等非即时零售场景 |
| 服务工单 | 中高端自行车维修、预约服务等 |
| 库存 | 门店库存（OSMS 库存子集），引用中央 SKU |
| 采购 | 销售订单驱动采购、门店备货，供应商来自 SupplyCore |
| 监控 | 门店室内外监控设备、实时预览与录像查阅（预留对接） |

## 端口

| 服务 | 端口 |
|------|------|
| API | 8094 |
| Web | 5179 |

## 本地开发

```bash
# 后端
make run

# 前端
cd web && npm install && npm run dev
```

登录：从 UserCore 应用中心（`:5174`）进入「门店管理」。

## 集成

- **UserCore**：JWT 鉴权、应用中心入口
- **ProductCore**：SKU/商品搜索（`/api/v1/admin/product-skus/search`）
- **SupplyCore**：供应商数据（后续对接）

## Docker / ACR

镜像名：`storecore-api`、`storecore-web`，CI 推送到阿里云 ACR（见 `.github/workflows/docker-push-acr.yml`）。

平台编排见 `/home/asialeaf/projects/deploy`。
