# Tasks

## 1. Plan Schedule Editing
- [x] Add default planned time and study-weekday controls to plan detail.
- [x] Add optional selected-weekday override controls with range validation.
- [x] Preserve loaded date overrides in every schedule update payload.
- [x] Confirm pending/running task recalculation and completed-task preservation before save.

## 2. Plan Detail Actions
- [x] Establish a fixed primary lifecycle and secondary maintenance hierarchy.
- [x] Move delay and destructive delete into a focused more-actions sheet.
- [x] Add lifecycle confirmation without restoring configurable actions.

## 3. Reminder Settings
- [x] Add typed template metadata and subscription records.
- [x] Map reminder names and purposes in the frontend.
- [x] Show authorization only when reminder type and current template ID both match.
- [x] Request and persist one template authorization per direct tap with server-verified reminder/template binding.
- [x] Refresh after authorization and cancel-all operations.
- [x] Explain one-time consumption, re-authorization, and H5 limitations.

## 4. Verification
- [x] Strictly validate this OpenSpec change.
- [x] Run frontend type-check.
- [x] Build H5 and mp-weixin targets.
