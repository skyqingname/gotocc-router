import type { GroupPlatform } from '@/types'
import {
  ccSwitchProtocolIssue,
  defaultCcSwitchAppId,
  ccSwitchApp,
  gatewayRoot,
  type CcSwitchApiFormat
} from '@/utils/ccswitchImport'

// Environment-variable snippets shown right after a key is created. Each client
// speaks one protocol; only clients the key's group can serve are offered, and
// the full per-client configuration stays in the "Use key" dialog.

export type EnvShell = 'unix' | 'powershell'

export interface EnvClient {
  id: 'claude' | 'openai' | 'gemini'
  format: CcSwitchApiFormat
}

const ENV_CLIENTS: EnvClient[] = [
  { id: 'claude', format: 'anthropic' },
  { id: 'openai', format: 'openai_chat' },
  { id: 'gemini', format: 'gemini_native' }
]

/** Clients the group can serve, the group platform's own protocol first. */
export function envClientsFor(platform: GroupPlatform | null | undefined, allowMessagesDispatch: boolean): EnvClient[] {
  const preferred = ccSwitchApp(defaultCcSwitchAppId(platform)).api_format
  const preferredFamily = preferred === 'openai_responses' ? 'openai_chat' : preferred
  return ENV_CLIENTS
    .filter(client => ccSwitchProtocolIssue(client.format, platform, allowMessagesDispatch) === null)
    .sort((a, b) => Number(b.format === preferredFamily) - Number(a.format === preferredFamily))
}

function envVariables(client: EnvClient, baseUrl: string, platform: GroupPlatform | null | undefined, apiKey: string): [string, string][] {
  const root = gatewayRoot(baseUrl)
  const prefixed = platform === 'antigravity' ? `${root}/antigravity` : root
  switch (client.id) {
    case 'claude':
      return [['ANTHROPIC_BASE_URL', prefixed], ['ANTHROPIC_AUTH_TOKEN', apiKey]]
    case 'gemini':
      return [['GOOGLE_GEMINI_BASE_URL', prefixed], ['GEMINI_API_KEY', apiKey]]
    case 'openai':
      return [['OPENAI_BASE_URL', `${root}/v1`], ['OPENAI_API_KEY', apiKey]]
  }
}

export function envSnippet(
  client: EnvClient,
  shell: EnvShell,
  baseUrl: string,
  platform: GroupPlatform | null | undefined,
  apiKey: string
): string {
  return envVariables(client, baseUrl, platform, apiKey)
    .map(([name, value]) => (shell === 'unix' ? `export ${name}="${value}"` : `$env:${name}="${value}"`))
    .join('\n')
}
