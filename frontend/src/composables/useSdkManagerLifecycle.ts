import { onMounted, watch, type Ref } from 'vue';

type SidebarAction = { id: number; type: 'display' };

type UseSdkManagerLifecycleOptions = {
  activeView: Ref<'list' | 'detail'>;
  sidebarAction: Ref<SidebarAction | null | undefined>;
  loadPlatformInfo: () => Promise<void>;
  closeDetail: () => void;
  emitSidebarActionDone: (actionId: number) => void;
};

export const useSdkManagerLifecycle = (options: UseSdkManagerLifecycleOptions) => {
  onMounted(async () => {
    await options.loadPlatformInfo();
  });

  watch(
    options.sidebarAction,
    (action) => {
      if (!action) return;
      if (options.activeView.value === 'detail') {
        options.closeDetail();
      }
      options.emitSidebarActionDone(action.id);
    },
    { immediate: true }
  );
};
