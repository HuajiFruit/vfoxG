import {
  EventsOn,
  WindowSetDarkTheme,
  WindowSetLightTheme,
  WindowSetSystemDefaultTheme,
} from '../../wailsjs/runtime/runtime';

export type RuntimeEventName =
  | 'migration-progress'
  | 'sdk-list-changed'
  | 'system-sdks-ready'
  | 'vfox-busy'
  | 'vfox-home-changed'
  | 'vfox-log';

export const onRuntimeEvent = <TPayload>(
  eventName: RuntimeEventName,
  handler: (payload: TPayload) => void
) => EventsOn(eventName, handler);

export const setWindowThemeMode = (theme: 'auto' | 'light' | 'dark') => {
  if (theme === 'auto') {
    return WindowSetSystemDefaultTheme();
  }
  if (theme === 'light') {
    return WindowSetLightTheme();
  }
  return WindowSetDarkTheme();
};
