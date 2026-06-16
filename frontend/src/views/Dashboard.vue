<template>
  <DashboardDocker v-if="dashboardMode === 'docker'" />
  <DashboardKubernetes v-else-if="dashboardMode === 'kubernetes'" />
  <DashboardBoth v-else />
</template>

<script setup>
import { computed } from 'vue';
import { dockerEnabled, kubernetesEnabled } from '../utils/sharedState';
import DashboardDocker from './dashboard/DashboardDocker.vue';
import DashboardKubernetes from './dashboard/DashboardKubernetes.vue';
import DashboardBoth from './dashboard/DashboardBoth.vue';

const dashboardMode = computed(() => {
  if (dockerEnabled() && kubernetesEnabled()) return 'both';
  if (kubernetesEnabled()) return 'kubernetes';
  return 'docker';
});
</script>
