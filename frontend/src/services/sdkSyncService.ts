import {
  ExportCurrentEnvironmentSdks,
  ImportSdkEnvironmentFromTxt,
  PreviewCurrentEnvironmentSdks,
} from '../../wailsjs/go/main/App';

export const previewCurrentEnvironmentSdks = () => PreviewCurrentEnvironmentSdks();

export const exportCurrentEnvironmentSdks = () => ExportCurrentEnvironmentSdks();

export const importSdkEnvironmentFromTxt = () => ImportSdkEnvironmentFromTxt();
