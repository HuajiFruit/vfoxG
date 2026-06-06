export type AppTab = 'sdk' | 'sync' | 'plugin' | 'settings';

export type NavTransition = 'slide-up' | 'slide-down';

export const appTabOrder: AppTab[] = ['sdk', 'sync', 'plugin', 'settings'];

export const getNavTransition = (currentTab: AppTab, nextTab: AppTab): NavTransition => {
  const currentIndex = appTabOrder.indexOf(currentTab);
  const nextIndex = appTabOrder.indexOf(nextTab);
  return nextIndex >= currentIndex ? 'slide-up' : 'slide-down';
};
