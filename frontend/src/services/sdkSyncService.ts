import {
  ExportCurrentEnvironmentSdks,
  ImportSdkEnvironmentFromTxt,
  PreviewCurrentEnvironmentSdks,
} from '../../wailsjs/go/app/App';

export const previewCurrentEnvironmentSdks = () => PreviewCurrentEnvironmentSdks();

export const exportCurrentEnvironmentSdks = () => ExportCurrentEnvironmentSdks();

export const importSdkEnvironmentFromTxt = () => ImportSdkEnvironmentFromTxt();
