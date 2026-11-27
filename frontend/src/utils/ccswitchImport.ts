import type { GroupPlatform } from '@/types'
import clientAccessDefaults from '../../../client-access-defaults.json'
import keyConsole from '../../../key-console.json'
import { creditUnit } from '@/utils/credits'

export const OPENAI_CC_SWITCH_CODEX_MODEL = clientAccessDefaults.openai_model
export const OPENAI_CODEX_DEFAULT_CLIENT_TAB = clientAccessDefaults.openai_codex_client_tab

export type CcSwitchApiFormat = 'anthropic' | 'openai_responses' | 'openai_chat' | 'gemini_native'
export type CcSwitchModelField = 'model' | 'haikuModel' | 'sonnetModel' | 'opusModel'

export interface CcSwitchApp {
  id: string
  label: string
  default_name: string
  api_format: CcSwitchApiFormat
  /** `root`: site root without `/v1`; `v1`: site root plus `/v1`. */
  endpoint: 'root' | 'v1'
  /** PlatformIcon id; empty shows the app's initial. */
  icon: GroupPlatform | ''
  model_fields: CcSwitchModelField[]
}

const ccswitch = keyConsole.ccswitch

/** Every application CC Switch can import a provider into, in display order (root key-console.json). */
export const CC_SWITCH_APPS = ccswitch.apps as CcSwitchApp[]

export function ccSwitchApp(id: string): CcSwitchApp {
  return CC_SWITCH_APPS.find(app => app.id === id) ?? CC_SWITCH_APPS[0]
}

/** The application preselected for a key: its group platform's own client, otherwise the default app. */
export function defaultCcSwitchAppId(platform: GroupPlatform | null | undefined): string {
  const byPlatform = ccswitch.default_app_by_platform as Record<string, string>
  return (platform && byPlatform[platform]) || ccswitch.default_app
}

/** Model preselected for an app when the key's model list contains it (client-access-defaults.json). */
export function defaultCcSwitchModel(appId: string): string {
  const field = (ccswitch.default_model_by_app as Record<string, keyof typeof clientAccessDefaults>)[appId]
  return field ? clientAccessDefaults[field] : ''
}

/**
 * Why a key's group cannot serve an app's protocol, mirroring the gateway's routing
 * (backend internal/server/routes/gateway.go):
 * - video groups only serve the video endpoints; TypeSafe groups only System One;
 * - Gemini native (/v1beta) is served by Gemini groups, and by Antigravity groups under /antigravity;
 * - /v1/messages on an OpenAI group needs the group's `allow_messages_dispatch`
 *   (Grok and multi-protocol API-key providers are exempt).
 * Keys without a fixed group (auto routing, aggregate) and composite groups resolve
 * the platform per request, so nothing is known in advance.
 */
export type CcSwitchProtocolIssue = 'videoOnly' | 'systemOneOnly' | 'geminiNativeOnly' | 'messagesDispatchOff'

export function ccSwitchProtocolIssue(
  format: CcSwitchApiFormat,
  platform: GroupPlatform | null | undefined,
  allowMessagesDispatch: boolean
): CcSwitchProtocolIssue | null {
  if (!platform || platform === 'composite') return null
  if (platform === 'video') return 'videoOnly'
  if (platform === 'typesafe') return 'systemOneOnly'
  if (format === 'gemini_native' && platform !== 'gemini' && platform !== 'antigravity') return 'geminiNativeOnly'
  if (format === 'anthropic' && platform === 'openai' && !allowMessagesDispatch) return 'messagesDispatchOff'
  return null
}

/** Base URL without trailing slashes and without a trailing `/v1`. */
export function gatewayRoot(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '').replace(/\/v1$/, '')
}

/**
 * Endpoint CC Switch stores for the app. Antigravity groups serve Messages and
 * Gemini native under `/antigravity`; their OpenAI-compatible routes stay at `/v1`.
 */
export function ccSwitchEndpoint(app: CcSwitchApp, baseUrl: string, platform: GroupPlatform | null | undefined): string {
  const root = gatewayRoot(baseUrl)
  if (app.endpoint === 'v1') return `${root}/v1`
  return platform === 'antigravity' ? `${root}/antigravity` : root
}

/**
 * Balance query CC Switch runs against the imported provider. CC Switch fills
 * `{{baseUrl}}` with the provider's endpoint as stored — some apps carry a
 * trailing `/v1`, others do not, and users may edit it either way afterwards —
 * so the URL strips an existing `/v1` instead of blindly appending one.
 */
export const CC_SWITCH_USAGE_SCRIPT = `({
    request: {
      url: "{{baseUrl}}".replace(/\\/+$/, "").replace(/\\/v1$/, "") + "/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? ${JSON.stringify(creditUnit())};
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`

function utf8Base64(text: string): string {
  const bytes = new TextEncoder().encode(text)
  return btoa(Array.from(bytes, byte => String.fromCharCode(byte)).join(''))
}

export interface CcSwitchImportDeeplinkInput {
  app: CcSwitchApp
  baseUrl: string
  homepage: string
  platform: GroupPlatform | null | undefined
  name: string
  apiKey: string
  models: Partial<Record<CcSwitchModelField, string>>
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', input.app.id],
    ['apiFormat', input.app.api_format],
    ['name', input.name],
    ['endpoint', ccSwitchEndpoint(input.app, input.baseUrl, input.platform)],
    ['apiKey', input.apiKey]
  ]
  for (const field of input.app.model_fields) {
    const value = input.models[field]?.trim()
    if (value) entries.push([field, value])
  }
  entries.push(
    ['homepage', gatewayRoot(input.homepage)],
    ['enabled', 'true'],
    ['usageEnabled', 'true'],
    ['usageScript', utf8Base64(CC_SWITCH_USAGE_SCRIPT)],
    ['usageAutoInterval', String(ccswitch.usage_auto_interval_minutes)]
  )
  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
