import { useTranslation } from 'react-i18next';
import type { MovieStatus } from '@/types';
import { FOCUS_RING, INK_65, INK_BORDER_10, TRANSITION_FAST } from '@/theme/customerTw';

export type BrowseTab = Extract<MovieStatus, 'showing' | 'coming_soon'> | 'special';

interface BrowseTabsProps {
  tab: BrowseTab;
  onChange: (tab: BrowseTab) => void;
}

const TABS: Array<{ key: BrowseTab; labelKey: string }> = [
  { key: 'showing', labelKey: 'customer.nowShowing' },
  { key: 'coming_soon', labelKey: 'customer.comingSoon' },
  { key: 'special', labelKey: 'customer.special' },
];

/** Now/Coming/Special underline tabs, shared by Home and /films so one change hits both. */
export const BrowseTabs = ({ tab, onChange }: BrowseTabsProps) => {
  const { t } = useTranslation();

  return (
    <div
      className={`mb-5 flex items-center gap-6 overflow-x-auto border-b ${INK_BORDER_10}`}
      role="tablist"
      aria-label={t('customer.browseTabsLabel')}
    >
      {TABS.map(({ key, labelKey }) => {
        const active = tab === key;
        return (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(key)}
            className={[
              '-mb-px flex-none border-none border-b-2 bg-transparent px-0.5 pb-3 text-base font-bold sm:text-lg',
              TRANSITION_FAST,
              FOCUS_RING,
              active ? 'border-brand text-brand' : `border-transparent ${INK_65}`,
            ].join(' ')}
          >
            {t(labelKey)}
          </button>
        );
      })}
    </div>
  );
};

export default BrowseTabs;
