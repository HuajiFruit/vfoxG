import {
  AddNonVfoxSdk,
  DetectSdkPathVersion,
  GetAddedPlugins,
  GetAvailablePlugins,
  GetCachedSystemSdks,
  GetNonVfoxSdks,
  GetSdkDetail,
  HijackSystemPath,
  InstallVersion,
  RefreshAvailablePlugins,
  RemovePluginWithOptions,
  RunVfoxWithProgress,
  ScanSystemSdks,
  SearchVersions,
  UseCustomSdk,
} from '../../wailsjs/go/app/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

export const fetchAvailablePlugins = () => GetAvailablePlugins();

export const refreshAvailablePlugins = () => RefreshAvailablePlugins();

export const fetchAddedPlugins = () => GetAddedPlugins();

export const addVfoxPlugin = (name: string) => RunVfoxWithProgress(['add', name]);

export const fetchSearchVersions = (name: string) => SearchVersions(name);

export const fetchSdkDetail = (name: string) => GetSdkDetail(name);

export const installPluginVersion = (pluginName: string, version: string) =>
  InstallVersion(pluginName, version);

export const removePlugin = (pluginName: string, chosenPath: string) =>
  RemovePluginWithOptions(pluginName, chosenPath);

export const refreshSystemSdkCache = () => ScanSystemSdks();

export const fetchCachedSystemSdks = () => GetCachedSystemSdks();

export const detectSystemSdkVersion = (name: string, sdkPath: string) =>
  DetectSdkPathVersion(name, sdkPath);

export const addCustomSdkReference = (name: string, sdkPath: string, version: string) =>
  AddNonVfoxSdk(name, sdkPath, version);

export const useCustomSdkReference = (name: string, sdkPath: string) =>
  UseCustomSdk(name, sdkPath);

export const hijackSystemSdkPath = (name: string, sdkPath: string) =>
  HijackSystemPath(name, sdkPath);

export const fetchNonVfoxSdks = () => GetNonVfoxSdks();

export const openExternalUrl = (url: string) => BrowserOpenURL(url);
