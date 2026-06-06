import {
  AddNonVfoxSdk,
  CheckPluginPathOverride,
  DetectSdkPathVersion,
  GetActiveCustomSdk,
  GetAllSdks,
  GetInstalledSdks,
  GetNonVfoxSdks,
  GetPlatformInfo,
  GetSdkDetail,
  GetVersionPath,
  HijackPluginSystemPath,
  InstallVersion,
  RemoveNonVfoxSdk,
  RemovePluginWithOptions,
  RestorePluginSystemPath,
  SearchVersions,
  UninstallVersion,
  UnuseVersion,
  UseCustomSdk,
  UseVersion,
} from '../../wailsjs/go/app/App';
import { ClipboardSetText } from '../../wailsjs/runtime/runtime';

export const fetchAllSdks = () => GetAllSdks();

export const fetchInstalledSdks = () => GetInstalledSdks();

export const fetchSdkDetail = (name: string) => GetSdkDetail(name);

export const useVfoxVersion = (name: string, version: string) => UseVersion(name, version);

export const unuseSdk = (name: string) => UnuseVersion(name);

export const installVfoxVersion = (name: string, version: string) => InstallVersion(name, version);

export const uninstallVfoxVersion = (name: string, version: string) => UninstallVersion(name, version);

export const searchSdkVersions = (name: string) => SearchVersions(name);

export const fetchVersionPath = (name: string, version: string) => GetVersionPath(name, version);

export const removePluginWithChoice = (name: string, chosenPath: string) =>
  RemovePluginWithOptions(name, chosenPath);

export const fetchNonVfoxSdks = () => GetNonVfoxSdks();

export const addNonVfoxSdk = (name: string, sdkPath: string, version: string) =>
  AddNonVfoxSdk(name, sdkPath, version);

export const removeNonVfoxSdk = (name: string, sdkPath: string) => RemoveNonVfoxSdk(name, sdkPath);

export const useNonVfoxSdk = (name: string, sdkPath: string) => UseCustomSdk(name, sdkPath);

export const detectSdkVersion = (name: string, sdkPath: string) => DetectSdkPathVersion(name, sdkPath);

export const enablePluginPathOverride = (name: string) => HijackPluginSystemPath(name);

export const disablePluginPathOverride = (name: string) => RestorePluginSystemPath(name);

export const checkPluginPathOverride = (name: string) => CheckPluginPathOverride(name);

export const fetchActiveCustomSdk = (name: string) => GetActiveCustomSdk(name);

export const fetchPlatformInfo = () => GetPlatformInfo();

export const copyTextToClipboard = (text: string) => ClipboardSetText(text);
