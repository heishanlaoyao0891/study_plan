## Why

旧计划记录可以保存空的日程数组，API 将其序列化为 `null`。小程序计划页按数组渲染学习日，因而在已成功创建计划后仍会因 `null.length` 崩溃，无法展示计划卡片。

## What Changes

- 计划查询 API 将缺失的学习日、学习日期和日程覆盖统一返回为空数组。
- 计划列表与详情界面对历史 `null` 数据进行防御性数组归一化。
- 用回归测试锁定 API 数据合同和小程序渲染保护。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `plan-management`: 计划读取接口与客户端必须安全处理历史缺失日程数组。

## Impact

- 后端计划列表与详情响应。
- 小程序计划列表、计划详情的日程摘要与编辑流程。
- 后端和小程序回归测试。
