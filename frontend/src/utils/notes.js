/**
 * 课程个人备注工具
 * 使用 localStorage 持久化，按 uid 隔离
 */

const STORAGE_KEY = 'jw_notes_v1'

function getNotes() {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

function saveNotes(notes) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(notes))
}

/**
 * 获取某门课的备注
 * @param {string} courseName - 课程名
 * @param {number} dayOfWeek - 星期几
 * @param {number} periodStart - 起始节次
 */
export function getNote(courseName, dayOfWeek, periodStart) {
  const key = makeKey(courseName, dayOfWeek, periodStart)
  return getNotes()[key] || ''
}

/**
 * 保存某门课的备注
 */
export function saveNote(courseName, dayOfWeek, periodStart, note) {
  const notes = getNotes()
  const key = makeKey(courseName, dayOfWeek, periodStart)
  if (note.trim()) {
    notes[key] = note.trim()
  } else {
    delete notes[key]
  }
  saveNotes(notes)
}

function makeKey(courseName, dayOfWeek, periodStart) {
  return `${courseName}__${dayOfWeek}__${periodStart}`
}

/**
 * 检查课程是否有备注
 */
export function hasNote(courseName, dayOfWeek, periodStart) {
  return !!getNote(courseName, dayOfWeek, periodStart)
}
