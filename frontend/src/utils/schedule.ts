export interface ScheduleInterval {
  id: string | number
  title: string
  date: string
  start: string
  end: string
}

export interface CoveredInterval {
  start: string
  end: string
}

export interface ScheduleConflict {
  id: string | number
  title: string
  plan_title?: string
  date: string
  covered_minutes: number
  covered_intervals: CoveredInterval[]
  conflicting_tasks: Array<{ task_id?: number; plan_id?: number; plan_title?: string; title: string; start: string; end: string }>
}

function toMinutes(value: string): number | null {
  const match = /^(\d{2}):(\d{2})$/.exec(String(value || '').trim())
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour > 23 || minute > 59) return null
  return hour * 60 + minute
}

function toTime(minutes: number): string {
  return `${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`
}

export function validateScheduleUnion(intervals: ScheduleInterval[]): ScheduleConflict[] {
  const valid = intervals.map(interval => ({ interval, start: toMinutes(interval.start), end: toMinutes(interval.end) }))
  const conflicts: ScheduleConflict[] = []
  for (const current of valid) {
    if (current.start === null || current.end === null || current.end <= current.start) continue
    const intersections: Array<[number, number]> = []
    const conflictingTasks: ScheduleConflict['conflicting_tasks'] = []
    for (const other of valid) {
      if (other === current || other.interval.date !== current.interval.date || other.start === null || other.end === null || other.end <= other.start) continue
      const start = Math.max(current.start, other.start)
      const end = Math.min(current.end, other.end)
      if (end > start) {
        intersections.push([start, end])
        conflictingTasks.push({ title: other.interval.title, start: toTime(start), end: toTime(end) })
      }
    }
    intersections.sort((a, b) => a[0] - b[0])
    const merged: Array<[number, number]> = []
    for (const range of intersections) {
      const previous = merged[merged.length - 1]
      if (!previous || range[0] > previous[1]) merged.push([...range])
      else previous[1] = Math.max(previous[1], range[1])
    }
    const coveredMinutes = merged.reduce((sum, range) => sum + range[1] - range[0], 0)
    if (coveredMinutes >= 60) {
      conflicts.push({
        id: current.interval.id,
        title: current.interval.title,
        date: current.interval.date,
        covered_minutes: coveredMinutes,
        covered_intervals: merged.map(range => ({ start: toTime(range[0]), end: toTime(range[1]) })),
        conflicting_tasks: conflictingTasks,
      })
    }
  }
  return conflicts
}

export function formatScheduleConflicts(conflicts: ScheduleConflict[]): string {
  return conflicts.map(item => {
    const currentTitle = item.plan_title || item.title
    const peers = (item.conflicting_tasks || []).map(peer => `计划「${peer.plan_title || peer.title}」(${peer.start}-${peer.end})`).join('、')
    return `${item.date} 计划「${currentTitle}」与 ${peers || '其他计划'} 重叠；「${currentTitle}」累计被覆盖 ${item.covered_minutes} 分钟（${item.covered_intervals.map(range => `${range.start}-${range.end}`).join('、')}）`
  }).join('\n')
}
