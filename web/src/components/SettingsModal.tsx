import { useEffect, useState } from 'react'
import {
  AlertCircle,
  Bot,
  Check,
  CheckCircle2,
  ExternalLink,
  Eye,
  EyeOff,
  Key,
  Loader2,
  RefreshCw,
  Sparkles,
  X,
} from 'lucide-react'
import { fetchAvailableModels, testSettings } from '../api/client'
import { useSettings } from '../hooks/useSettings'
import { cn } from '../lib/utils'
import type { ModelItem } from '../types'
import { Button } from './cm/primitives'

interface SettingsModalProps {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { settings, updateSettings, isUpdating } = useSettings()

  const [provider, setProvider] = useState<string>('gemini')
  const [model, setModel] = useState<string>('gemini-2.0-flash')
  const [isCustomModel, setIsCustomModel] = useState(false)
  const [customModelName, setCustomModelName] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [showApiKey, setShowApiKey] = useState(false)

  // Live models state
  const [liveModels, setLiveModels] = useState<ModelItem[] | null>(null)
  const [isFetchingModels, setIsFetchingModels] = useState(false)
  const [modelsError, setModelsError] = useState<string | null>(null)

  // Test connection state
  const [isTesting, setIsTesting] = useState(false)
  const [testResult, setTestResult] = useState<{
    success?: boolean
    message?: string
    error?: string
  } | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)

  // Static fallback providers & models
  const providers = settings?.supportedProviders ?? [
    {
      id: 'gemini',
      name: 'Google Gemini',
      defaultModel: 'gemini-2.0-flash',
      models: [
        'gemini-2.0-flash',
        'gemini-2.0-flash-lite',
        'gemini-1.5-flash',
        'gemini-1.5-flash-8b',
        'gemini-1.5-pro',
      ],
      keyHelpUrl: 'https://aistudio.google.com/apikey',
      keyPrefix: 'AIza',
    },
    {
      id: 'openai',
      name: 'OpenAI',
      defaultModel: 'gpt-4o',
      models: ['gpt-4o', 'gpt-4o-mini', 'o3-mini', 'gpt-4-turbo'],
      keyHelpUrl: 'https://platform.openai.com/api-keys',
      keyPrefix: 'sk-',
    },
  ]

  const activeProviderMeta = providers.find((p) => p.id === provider) ?? providers[0]

  // Initialize form state when settings are loaded or modal opens
  useEffect(() => {
    if (!isOpen || !settings) return

    const initialProvider = settings.provider || 'gemini'
    setProvider(initialProvider)

    const activeMeta = settings.supportedProviders.find((p) => p.id === initialProvider)
    const currentModel = settings.model || activeMeta?.defaultModel || 'gemini-2.0-flash'

    if (activeMeta && !activeMeta.models.includes(currentModel)) {
      setIsCustomModel(true)
      setCustomModelName(currentModel)
      setModel('custom')
    } else {
      setIsCustomModel(false)
      setModel(currentModel)
    }

    setApiKey('')
    setShowApiKey(false)
    setTestResult(null)
    setSaveSuccess(false)
    setLiveModels(null)
    setModelsError(null)

    // If API key is available for the current provider, auto-fetch live models
    if (settings.hasApiKey && initialProvider === settings.provider) {
      void fetchModelsFor(initialProvider, undefined)
    }
  }, [isOpen, settings])

  // Handle ESC key to close
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  const isCurrentProvider = settings?.provider === provider
  const currentKeyAvailable = isCurrentProvider && settings?.hasApiKey

  const fetchModelsFor = async (targetProvider: string, explicitKey?: string) => {
    const keyToUse = explicitKey !== undefined ? explicitKey : apiKey.trim() || undefined
    // Check if key is available
    const hasKey = Boolean(keyToUse || (settings?.provider === targetProvider && settings?.hasApiKey))
    if (!hasKey) {
      setLiveModels(null)
      return
    }

    setIsFetchingModels(true)
    setModelsError(null)

    try {
      const res = await fetchAvailableModels({
        provider: targetProvider,
        apiKey: keyToUse,
      })
      if (res.models && res.models.length > 0) {
        setLiveModels(res.models)
        // If current model is not in the live list and not custom, switch to the first live model
        if (!isCustomModel) {
          const exists = res.models.some((m) => m.id === model)
          if (!exists) {
            setModel(res.models[0].id)
          }
        }
      }
    } catch (err) {
      setModelsError(err instanceof Error ? err.message : String(err))
    } finally {
      setIsFetchingModels(false)
    }
  }

  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider)
    const targetMeta = providers.find((p) => p.id === newProvider)
    if (targetMeta) {
      setIsCustomModel(false)
      setModel(targetMeta.defaultModel)
    }
    setTestResult(null)
    setSaveSuccess(false)
    setLiveModels(null)
    setModelsError(null)

    // Auto-fetch if key is available for the new provider
    if (settings?.provider === newProvider && settings?.hasApiKey) {
      void fetchModelsFor(newProvider, undefined)
    }
  }

  const handleModelChange = (selected: string) => {
    if (selected === 'custom') {
      setIsCustomModel(true)
      setModel('custom')
      if (!customModelName && activeProviderMeta) {
        setCustomModelName(activeProviderMeta.defaultModel)
      }
    } else {
      setIsCustomModel(false)
      setModel(selected)
    }
    setTestResult(null)
    setSaveSuccess(false)
  }

  const effectiveModel = isCustomModel ? customModelName.trim() : model

  const handleFetchLiveModelsClick = () => {
    void fetchModelsFor(provider, apiKey.trim() || undefined)
  }

  const handleTest = async () => {
    setIsTesting(true)
    setTestResult(null)
    setSaveSuccess(false)

    try {
      const res = await testSettings({
        provider,
        model: effectiveModel,
        apiKey: apiKey.trim() || undefined,
      })
      setTestResult(res)
    } catch (err) {
      setTestResult({
        success: false,
        error: err instanceof Error ? err.message : String(err),
      })
    } finally {
      setIsTesting(false)
    }
  }

  const handleSave = async () => {
    setTestResult(null)
    try {
      await updateSettings({
        provider,
        model: effectiveModel,
        apiKey: apiKey.trim() || undefined,
      })
      setSaveSuccess(true)
      setTimeout(() => {
        onClose()
      }, 700)
    } catch (err) {
      setTestResult({
        success: false,
        error: err instanceof Error ? err.message : 'Failed to save settings',
      })
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/70 backdrop-blur-xs transition-opacity"
        onClick={onClose}
      />

      {/* Modal Dialog */}
      <div className="relative z-10 w-full max-w-lg rounded-lg border border-border-strong bg-surface shadow-2xl">
        {/* Header */}
        <div className="flex h-12 items-center justify-between border-b px-5">
          <div className="flex items-center gap-2.5">
            <div className="grid size-7 place-items-center rounded-sm bg-primary/15 text-primary">
              <Bot className="size-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold tracking-tight">AI & Model Settings</h2>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="grid size-7 place-items-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X className="size-4" />
          </button>
        </div>

        {/* Body */}
        <div className="space-y-5 p-5 text-[13px]">
          {/* Provider Selection */}
          <div>
            <label className="label-xs mb-2 block">AI Provider</label>
            <div className="grid grid-cols-2 gap-2">
              {providers.map((p) => {
                const isSelected = provider === p.id
                return (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => handleProviderChange(p.id)}
                    className={cn(
                      'flex items-center justify-between rounded-md border p-3 text-left transition-colors',
                      isSelected
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'border-border-strong bg-surface-2 text-muted-foreground hover:bg-accent hover:text-foreground',
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <Sparkles
                        className={cn('size-4', isSelected ? 'text-primary' : 'text-muted-foreground')}
                      />
                      <span className="font-medium">{p.name}</span>
                    </div>
                    {isSelected && <Check className="size-4 text-primary" />}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Model Selection */}
          <div>
            <div className="mb-2 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <label htmlFor="model-select" className="label-xs">
                  Model
                </label>
                {liveModels && liveModels.length > 0 && (
                  <span className="inline-flex items-center gap-1 rounded-sm border border-healthy/40 bg-healthy/10 px-1.5 py-0.5 font-mono text-[10px] text-healthy">
                    <span className="size-1 rounded-full bg-healthy" />
                    Live ({liveModels.length} models)
                  </span>
                )}
              </div>

              {/* Real-time Fetch Live Models Button */}
              <button
                type="button"
                onClick={handleFetchLiveModelsClick}
                disabled={isFetchingModels || (!currentKeyAvailable && !apiKey.trim())}
                title={
                  !currentKeyAvailable && !apiKey.trim()
                    ? 'Enter an API key to fetch real-time available models'
                    : 'Fetch latest available models in real-time from API'
                }
                className="inline-flex items-center gap-1 text-[11px] font-medium text-primary hover:underline disabled:cursor-not-allowed disabled:text-muted-foreground"
              >
                <RefreshCw className={cn('size-3', isFetchingModels && 'animate-spin')} />
                <span>{isFetchingModels ? 'Fetching...' : 'Fetch Live Models'}</span>
              </button>
            </div>

            <select
              id="model-select"
              value={isCustomModel ? 'custom' : model}
              onChange={(e) => handleModelChange(e.target.value)}
              className="h-9 w-full rounded-md border border-border-strong bg-surface-2 px-3 font-mono text-xs text-foreground focus:border-primary focus:outline-hidden"
            >
              {liveModels && liveModels.length > 0
                ? liveModels.map((m) => (
                    <option key={m.id} value={m.id} title={m.description}>
                      {m.displayName && m.displayName !== m.id
                        ? `${m.displayName} (${m.id})`
                        : m.id}
                    </option>
                  ))
                : activeProviderMeta.models.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
              <option value="custom">Custom model (enter manually)...</option>
            </select>

            {modelsError && (
              <p className="mt-1.5 flex items-center gap-1 text-[11px] text-critical">
                <AlertCircle className="size-3 shrink-0" />
                <span>{modelsError}</span>
              </p>
            )}

            {isCustomModel && (
              <div className="mt-2">
                <input
                  type="text"
                  placeholder="e.g. gemini-2.0-flash or gpt-4o"
                  value={customModelName}
                  onChange={(e) => setCustomModelName(e.target.value)}
                  className="h-8 w-full rounded-md border border-border-strong bg-surface-2 px-2.5 font-mono text-xs text-foreground focus:border-primary focus:outline-hidden"
                />
              </div>
            )}
          </div>

          {/* API Key */}
          <div>
            <div className="mb-2 flex items-center justify-between">
              <label htmlFor="api-key-input" className="label-xs">
                API Key
              </label>
              {currentKeyAvailable ? (
                <span className="inline-flex items-center gap-1 font-mono text-[11px] text-healthy">
                  <span className="size-1.5 rounded-full bg-healthy" />
                  Configured ({settings.maskedApiKey})
                </span>
              ) : (
                <span className="font-mono text-[11px] text-muted-foreground">Not configured</span>
              )}
            </div>

            <div className="relative">
              <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-muted-foreground">
                <Key className="size-3.5" />
              </div>
              <input
                id="api-key-input"
                type={showApiKey ? 'text' : 'password'}
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={
                  currentKeyAvailable
                    ? 'Leave blank to keep current key, or enter new key'
                    : `Enter ${activeProviderMeta.name} API Key (${activeProviderMeta.keyPrefix}...)`
                }
                className="h-9 w-full rounded-md border border-border-strong bg-surface-2 pl-8 pr-9 font-mono text-xs text-foreground focus:border-primary focus:outline-hidden"
              />
              <button
                type="button"
                onClick={() => setShowApiKey(!showApiKey)}
                tabIndex={-1}
                aria-label={showApiKey ? 'Hide key' : 'Show key'}
                className="absolute inset-y-0 right-0 flex items-center pr-2.5 text-muted-foreground hover:text-foreground"
              >
                {showApiKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </button>
            </div>

            <div className="mt-1.5 flex items-center justify-between">
              <a
                href={activeProviderMeta.keyHelpUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 text-[11px] text-primary hover:underline"
              >
                Get {activeProviderMeta.name} API Key
                <ExternalLink className="size-2.5" />
              </a>
              <span className="text-[11px] text-muted-foreground">
                Stored in-memory on backend
              </span>
            </div>
          </div>

          {/* Test / Save Alerts */}
          {testResult && (
            <div
              className={cn(
                'flex items-start gap-2 rounded-md border p-3 text-xs leading-relaxed',
                testResult.success
                  ? 'border-healthy/40 bg-healthy/10 text-healthy'
                  : 'border-critical/40 bg-critical/10 text-critical',
              )}
            >
              {testResult.success ? (
                <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-healthy" />
              ) : (
                <AlertCircle className="mt-0.5 size-4 shrink-0 text-critical" />
              )}
              <div className="flex-1">
                <p className="font-medium">
                  {testResult.success ? 'Connection Successful' : 'Connection Failed'}
                </p>
                <p className="mt-0.5 opacity-90">
                  {testResult.success ? testResult.message : testResult.error}
                </p>
              </div>
            </div>
          )}

          {saveSuccess && (
            <div className="flex items-center gap-2 rounded-md border border-healthy/40 bg-healthy/10 p-2.5 text-xs text-healthy">
              <CheckCircle2 className="size-4 shrink-0" />
              <span>Settings saved successfully! Closing...</span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t px-5 py-3">
          <Button
            type="button"
            variant="secondary"
            onClick={handleTest}
            disabled={isTesting || (!currentKeyAvailable && !apiKey.trim())}
            className="text-xs"
          >
            {isTesting && <Loader2 className="size-3.5 animate-spin" />}
            Test Connection
          </Button>

          <div className="flex items-center gap-2">
            <Button type="button" variant="ghost" onClick={onClose} className="text-xs">
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick={handleSave}
              disabled={isUpdating || saveSuccess}
              className="text-xs"
            >
              {isUpdating && <Loader2 className="size-3.5 animate-spin" />}
              Save Settings
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
