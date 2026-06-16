import { computed } from 'vue';
import { sharedState, formatBytes } from '../utils/sharedState';

export function useDashboardShared() {
  const username = computed(() => sharedState.currentUser?.username || 'there');

  const greeting = computed(() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  });

  const cpuPercent = computed(() => parseFloat(sharedState.systemStats?.cpu || 0));
  const memUsed = computed(() => sharedState.systemStats?.memory || 0);
  const memTotal = computed(() => sharedState.systemStats?.total_memory || 1);

  const memPercent = computed(() => {
    if (!memTotal.value) return 0;
    return Math.min(100, (memUsed.value / memTotal.value) * 100);
  });

  const hostStatusLabel = computed(() => {
    const load = Math.max(cpuPercent.value, memPercent.value);
    if (load > 80) return 'High resource pressure';
    if (load > 50) return 'Moderate system load';
    return 'System running smoothly';
  });

  const statColor = (val) => {
    if (val > 80) return 'var(--error)';
    if (val > 50) return 'var(--warning)';
    return 'var(--accent)';
  };

  return {
    sharedState,
    username,
    greeting,
    cpuPercent,
    memUsed,
    memTotal,
    memPercent,
    hostStatusLabel,
    statColor,
    formatBytes,
  };
}
