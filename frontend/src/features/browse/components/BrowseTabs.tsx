import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { MovieStatus } from '@/types';
import { FOCUS_RING, INK_65, INK_BORDER_10, TRANSITION_FAST } from '@/theme/customerTw';

export type BrowseTab = Extract<MovieStatus, 'showing' | 'coming_soon'> | 'special';

interface BrowseTabsProps {
  tab: BrowseTab;
  onChange: (tab: BrowseTab) => void;
  /** Rendered at the right end of the tab bar, outside the tablist. The home puts its
   *  "View All ->" link here so it shares the bar the way the QVisionShow frame draws it;
   *  /films passes nothing and is unchanged. */
  trailing?: ReactNode;
}

const TABS: Array<{ key: BrowseTab; labelKey: string }> = [
  { key: 'showing', labelKey: 'customer.nowShowing' },
  { key: 'coming_soon', labelKey: 'customer.comingSoon' },
  { key: 'special', labelKey: 'customer.special' },
];

/** Now/Coming/Special underline tabs, shared by Home and /films so one change hits both. */
export const BrowseTabs = ({ tab, onChange, trailing }: BrowseTabsProps) => {
  const { t } = useTranslation();

  // `role="tablist"` moved onto the inner row: `trailing` is a link, not a tab, and a non-tab child
  // of a tablist is an ARIA violation.
  return (
    <div className={`mb-5 flex items-center gap-6 border-b ${INK_BORDER_10}`}>
      <div
        className="flex items-center gap-6 overflow-x-auto"
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

      {trailing ? <div className="ml-auto flex-none pb-3">{trailing}</div> : null}
    </div>
  );
};

export default BrowseTabs;
