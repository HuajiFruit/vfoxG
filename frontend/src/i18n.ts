import { ref, watch } from 'vue';

const getInitialLang = () => {
  const saved = localStorage.getItem('vfox-lang');
  if (saved && (saved === 'en' || saved === 'zh')) {
    return saved;
  }
  return navigator.language.startsWith('zh') ? 'zh' : 'en';
};

export const currentLang = ref<'en' | 'zh'>(getInitialLang());

watch(currentLang, (newLang) => {
  localStorage.setItem('vfox-lang', newLang);
});

type TranslationDict = Record<string, string>;

const en: TranslationDict = {
  // Navigation
  'nav.installed': 'Installed',
  'nav.market': 'Plugin Market',
  'nav.settings': 'Settings',

  // Settings
  'settings.title': 'Settings',
  'settings.appearance': 'Appearance',
  'settings.theme': 'Theme',
  'settings.theme.desc': 'Choose your preferred interface theme',
  'settings.theme.light': 'Light',
  'settings.theme.dark': 'Dark',
  'settings.theme.auto': 'Auto',
  'settings.language': 'Language',
  'settings.language.desc': 'Choose interface language',
  'settings.language.en': 'English',
  'settings.language.zh': '中文',
  'settings.system': 'System Integration',
  'settings.system.path': 'System PATH',
  'settings.system.path.desc': 'Add the `vfox` command to your global system PATH',
  'settings.system.path.add': 'Add to PATH',
  'settings.system.path.remove': 'Remove from PATH',

  // SdkManager
  'sdk.installed.title': 'Installed SDKs',
  'sdk.installed.empty': 'No SDKs found on this system.',
  'sdk.vfox.title': 'Vfox SDKs',
  'sdk.nonevfox.title': 'Custom SDKs',
  'sdk.version': 'version',
  'sdk.versions': 'versions',
  'sdk.version_list': 'Versions',
  'sdk.back': 'Back',
  'sdk.add_system_path': 'Add to System PATH',
  'sdk.add_system_path.tooltip': 'Force this SDK to override the Machine PATH (Fixes Microsoft Store alias conflicts). Requires Administrator privileges.',
  'sdk.remove_system_path': 'Remove from System PATH',
  'sdk.remove_system_path.tooltip': 'Remove this SDK from the Machine PATH to restore original behavior. Requires Administrator privileges.',
  'sdk.add_system_path.hint': 'Use this if version changes don\'t take effect',
  'sdk.remove_plugin': 'Remove Plugin',
  'sdk.available_versions': 'Available Versions',
  'sdk.loading': 'Loading versions...',
  'sdk.current': 'Current',
  'sdk.custom': 'Custom',
  'sdk.use': 'Use',
  'sdk.unset': 'Unset',
  'sdk.uninstall': 'Uninstall',
  'sdk.remove': 'Remove',
  'sdk.install_path': 'Installation Path',
  'sdk.exe_path': 'Executable Path',
  'sdk.install_new': 'Install New Version',
  'sdk.add_custom': 'Add Custom Path',
  'sdk.cancel': 'Cancel',
  'sdk.add': 'Add',
  'sdk.custom_path.placeholder': 'Custom exe path (e.g. C:\\Python39\\python.exe)',
  'sdk.version.placeholder': 'Name',
  'sdk.confirm.remove_plugin': 'Are you sure you want to completely remove the plugin',
  'sdk.confirm.uninstall_version': 'Are you sure you want to uninstall version',
  'sdk.confirm.remove_custom': 'Are you sure you want to remove the custom SDK',
  'sdk.confirm.note': 'Note: This only removes the reference from Vfox.',
  'sdk.remove_reference': 'Remove Reference',
  'sdk.remove_plugin.choose_sdk': 'This plugin has custom SDK paths. Choose which one to keep as the system SDK, or remove everything:',
  'sdk.remove_plugin.clean': 'Clean removal (remove all)',
  'sdk.remove_plugin.clean_desc': 'Remove the plugin and all SDK references. No environment variables will remain.',

  // PluginMarket
  'market.title': 'Plugin Market',
  'market.search.placeholder': 'Search plugins...',
  'market.no_results': 'No plugins found matching your search.',
  'market.loading': 'Loading plugins from registry...',
  'market.install': 'Install',
  'market.installed': 'Installed',
  'market.homepage': 'Homepage',
};

const zh: TranslationDict = {
  // Navigation
  'nav.installed': '已安装',
  'nav.market': '插件市场',
  'nav.settings': '设置',

  // Settings
  'settings.title': '设置',
  'settings.appearance': '外观',
  'settings.theme': '主题',
  'settings.theme.desc': '选择你偏好的界面主题',
  'settings.theme.light': '浅色',
  'settings.theme.dark': '深色',
  'settings.theme.auto': '跟随系统',
  'settings.language': '语言',
  'settings.language.desc': '选择界面语言',
  'settings.language.en': 'English',
  'settings.language.zh': '中文',
  'settings.system': '系统集成',
  'settings.system.path': '系统 PATH',
  'settings.system.path.desc': '将 `vfox` 命令添加到全局系统 PATH 环境变量',
  'settings.system.path.add': '添加到 PATH',
  'settings.system.path.remove': '从 PATH 移除',

  // SdkManager
  'sdk.installed.title': '已安装 SDK',
  'sdk.installed.empty': '系统上未找到已安装的 SDK。',
  'sdk.vfox.title': 'Vfox SDK',
  'sdk.nonevfox.title': '自定义 SDK',
  'sdk.version': '个版本',
  'sdk.versions': '个版本',
  'sdk.version_list': '版本列表',
  'sdk.back': '返回',
  'sdk.add_system_path': '添加到系统 PATH',
  'sdk.add_system_path.tooltip': '强制此 SDK 覆盖全局 Machine PATH（解决微软商店别名冲突等问题）。需要管理员权限。',
  'sdk.remove_system_path': '从系统 PATH 移除',
  'sdk.remove_system_path.tooltip': '将此 SDK 从全局 Machine PATH 中移除以恢复原始行为。需要管理员权限。',
  'sdk.add_system_path.hint': '如果版本切换未生效，请使用此功能接管底层环境变量',
  'sdk.remove_plugin': '移除插件',
  'sdk.available_versions': '可用版本',
  'sdk.loading': '正在加载版本...',
  'sdk.current': '当前',
  'sdk.custom': '自定义',
  'sdk.use': '应用',
  'sdk.unset': '取消',
  'sdk.uninstall': '卸载',
  'sdk.remove': '移除',
  'sdk.install_path': '安装路径',
  'sdk.exe_path': '可执行文件路径',
  'sdk.install_new': '安装新版本',
  'sdk.add_custom': '添加自定义路径',
  'sdk.cancel': '取消',
  'sdk.add': '添加',
  'sdk.custom_path.placeholder': '自定义可执行文件路径 (例: C:\\Python39\\python.exe)',
  'sdk.version.placeholder': '名称',
  'sdk.confirm.remove_plugin': '确定要彻底移除插件以及卸载其所有版本吗：',
  'sdk.confirm.uninstall_version': '确定要卸载该版本吗：',
  'sdk.confirm.remove_custom': '确定要移除此自定义 SDK 吗：',
  'sdk.confirm.note': '注意：这仅仅是从 Vfox 中移除引用，并不会删除磁盘上的物理文件。',
  'sdk.remove_reference': '移除引用',
  'sdk.remove_plugin.choose_sdk': '该插件关联了自定义 SDK 路径。请选择保留哪个作为系统 SDK，或全部清除：',
  'sdk.remove_plugin.clean': '彻底清除（不保留任何环境）',
  'sdk.remove_plugin.clean_desc': '移除插件并清除所有 SDK 引用和环境变量，不留任何残留。',

  // PluginMarket
  'market.title': '插件市场',
  'market.search.placeholder': '搜索插件...',
  'market.no_results': '未找到匹配的插件。',
  'market.loading': '正在从注册表加载插件...',
  'market.install': '安装',
  'market.installed': '已安装',
  'market.homepage': '主页',
};

const messages = { en, zh };

export const t = (key: string): string => {
  const dict = messages[currentLang.value] || messages.en;
  return dict[key] || key;
};
