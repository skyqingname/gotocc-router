/**
 * 模型广场按模型的展示信息（GoToCC）：后台填写的简介、厂商、用途，覆盖 models.dev 的自动值。
 * 键为小写模型名。
 */
import { apiClient } from '../client'

export interface PlazaModelOverride {
  description: string
  vendor: string
  purposes: string[]
}

export type PlazaModelOverrides = Record<string, PlazaModelOverride>

export async function getModelPlazaOverrides(): Promise<PlazaModelOverrides> {
  const { data } = await apiClient.get<PlazaModelOverrides>('/admin/model-plaza/overrides')
  return data
}

export async function updateModelPlazaOverrides(overrides: PlazaModelOverrides): Promise<PlazaModelOverrides> {
  const { data } = await apiClient.put<PlazaModelOverrides>('/admin/model-plaza/overrides', overrides)
  return data
}
