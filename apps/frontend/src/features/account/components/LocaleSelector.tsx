import { useTranslation } from 'react-i18next'
import { setLocale, type SupportedLocale } from '../../../shared/i18n'
import { cn } from '../../../shared/ui/cn'

type LocaleOption = {
  value: SupportedLocale
  labelKey: 'languagePortuguese' | 'languageEnglish'
}

const OPTIONS: readonly LocaleOption[] = [
  { value: 'pt-BR', labelKey: 'languagePortuguese' },
  { value: 'en', labelKey: 'languageEnglish' },
]

export function LocaleSelector() {
  const { t, i18n } = useTranslation('account')
  const active: SupportedLocale = i18n.language === 'en' ? 'en' : 'pt-BR'

  return (
    <fieldset className="space-y-2">
      <legend className="text-sm font-medium text-primary">{t('interfaceLanguage')}</legend>
      <div className="flex flex-col gap-2 sm:flex-row">
        {OPTIONS.map((option) => {
          const isActive = active === option.value
          return (
            <button
              key={option.value}
              type="button"
              className={cn(
                'rounded-lg border px-4 py-2 text-left text-sm transition-colors',
                isActive
                  ? 'border-cobalt/40 bg-cobalt/10 text-primary'
                  : 'border-line bg-white/[0.02] text-secondary hover:bg-white/[0.04]',
              )}
              aria-pressed={isActive}
              onClick={() => {
                if (!isActive) {
                  void setLocale(option.value)
                }
              }}
            >
              {t(option.labelKey)}
            </button>
          )
        })}
      </div>
    </fieldset>
  )
}
