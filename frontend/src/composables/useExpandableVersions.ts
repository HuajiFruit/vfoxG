import { ref } from 'vue';

export const useExpandableVersions = () => {
  const expandedVersions = ref<Record<string, boolean>>({});

  const toggleExpand = (versionKey: string) => {
    expandedVersions.value[versionKey] = !expandedVersions.value[versionKey];
  };

  return {
    expandedVersions,
    toggleExpand,
  };
};
