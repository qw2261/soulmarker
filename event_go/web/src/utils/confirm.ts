import { ElMessageBox, type ElMessageBoxOptions } from 'element-plus'

export function confirmAction(message: string, title: string, options: ElMessageBoxOptions = {}) {
  return ElMessageBox.confirm(message, title, {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    ...options,
  })
}
