<template>
  <div class="page-view health-container">
    <section class="page-hero animate-slide-up">
      <div class="page-hero-body">
        <div class="page-hero-copy">
          <span class="page-hero-eyebrow">Diagnostics</span>
          <h1>System health</h1>
          <p class="page-hero-sub">
            Historical CPU and memory utilization
            <span v-if="isPartialData" class="coverage-hint">
              · Showing {{ formatDuration(availableHours) }} of data
            </span>
          </p>
        </div>
        <div class="page-hero-actions">
          <div class="page-filter-pills">
            <button
              v-for="f in filters"
              :key="f.label"
              @click="activeFilter = f.value"
              :class="[
                'page-filter-pill',
                {
                  active: activeFilter === f.value,
                  'is-partial': availableHours < f.value,
                },
              ]"
              :data-tooltip="availableHours < f.value ? `${f.note} (partial data)` : f.note"
            >
              {{ f.short }}
            </button>
            <button
              class="page-filter-pill"
              @click="showCustomModal = true"
              :class="{ active: activeFilter === 'custom' }"
              data-tooltip="Select specific dates"
            >
              Custom
            </button>
          </div>
        </div>
      </div>
      <div class="page-hero-mesh" aria-hidden="true"></div>
    </section>

    <section class="page-metrics animate-slide-up" style="animation-delay: 0.05s">
      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon">
            <AppIcon name="activity" />
          </div>
          <span class="badge badge-dim">Average</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Avg CPU load</span>
          <span class="stat-value">{{ avgCpu }}%</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon error">
            <AppIcon name="bell" />
          </div>
          <span class="badge badge-error">Peak</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Peak CPU spike</span>
          <span class="stat-value">{{ maxCpu }}%</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon success">
            <AppIcon name="server" />
          </div>
          <span class="badge badge-success">Memory</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Avg memory</span>
          <span class="stat-value">{{ avgMem }} GB</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon warning">
            <AppIcon name="containers" />
          </div>
          <span class="badge badge-warning">Max</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Peak memory</span>
          <span class="stat-value">{{ maxMem }} GB</span>
        </div>
      </div>
    </section>

    <section class="health-grid animate-slide-up" style="animation-delay: 0.1s">
      <div class="chart-section">
        <div class="chart-header">
          <h4>CPU Utilization History</h4>
          <span class="unit-badge">% Load</span>
        </div>
        <div class="chart-container">
          <Line
            v-if="chartData.cpu.labels.length"
            :data="chartData.cpu"
            :options="cpuChartOptions"
          />
          <div v-else class="chart-placeholder">
            <div class="shimmer"></div>
          </div>
        </div>
      </div>

      <div class="chart-section">
        <div class="chart-header">
          <h4>Memory Consumption</h4>
          <span class="unit-badge">Gigabytes</span>
        </div>
        <div class="chart-container">
          <Line
            v-if="chartData.mem.labels.length"
            :data="chartData.mem"
            :options="memChartOptions"
          />
          <div v-else class="chart-placeholder">
            <div class="shimmer"></div>
          </div>
        </div>
      </div>
    </section>

    <!-- Custom Range Modal -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showCustomModal" class="modal-overlay">
          <div class="modal-content shadow-2xl">
            <div class="modal-header">
              <h3>Custom Range</h3>
            </div>
            <div class="modal-body">
              <div class="input-group">
                <label>Start Date</label>
                <input type="date" v-model="tempStart" :max="today" class="premium-input" />
              </div>
              <div class="input-group">
                <label>End Date</label>
                <input type="date" v-model="tempEnd" :max="today" class="premium-input" />
              </div>
              <p v-if="modalError" class="error-text">{{ modalError }}</p>
            </div>
            <div class="modal-actions">
              <button @click="showCustomModal = false" class="modal-btn cancel">Cancel</button>
              <button @click="applyCustomRange" class="modal-btn confirm">Apply Range</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import AppIcon from "../components/AppIcon.vue";
import { ref, onMounted, onUnmounted, watch, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Line } from "vue-chartjs";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Filler,
} from "chart.js";
import { secureStorage } from "../utils/storage";
import { sharedState } from "../utils/sharedState";
import { apiFetch } from "../utils/apiFetch";

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Filler,
);

const isDark = computed(() => sharedState.theme === 'dark');
const route = useRoute();
const router = useRouter();

const filters = [
  { label: "1H", short: "1H", note: "Last hour", value: 1 },
  { label: "6H", short: "6H", note: "Last 6 hours", value: 6 },
  { label: "12H", short: "12H", note: "Last 12 hours", value: 12 },
  { label: "24H", short: "24H", note: "Last 24 hours", value: 24 },
];

const activeFilter = ref(24);
const customStart = ref("");
const customEnd = ref("");
const tempStart = ref("");
const tempEnd = ref("");
const showCustomModal = ref(false);
const modalError = ref("");
const today = new Date().toISOString().split("T")[0];

const history = ref([]);

const availableHours = computed(() => {
  if (history.value.length === 0) return 0;
  const timestamps = history.value.map(h => new Date(h.timestamp).getTime());
  const oldest = Math.min(...timestamps);
  const now = new Date().getTime();
  return Math.max(0, (now - oldest) / (1000 * 60 * 60));
});

const isPartialData = computed(() => {
  if (activeFilter.value === 'custom') return false;
  // If we have less than 95% of the requested range, call it partial
  return availableHours.value < (activeFilter.value * 0.95);
});

const formatDuration = (hours) => {
  if (hours < 1) return `${Math.round(hours * 60)}m`;
  if (hours < 24) return `${hours.toFixed(1)}h`;
  return `${(hours / 24).toFixed(1)}d`;
};
const chartData = ref({
  cpu: { labels: [], datasets: [] },
  mem: { labels: [], datasets: [] },
});

const activeHistory = computed(() => {
  let start, end;
  if (activeFilter.value === "custom") {
    if (!customStart.value || !customEnd.value) return history.value;
    start = new Date(customStart.value);
    end = new Date(customEnd.value);
    end.setHours(23, 59, 59);
  } else {
    end = new Date();
    start = new Date(end.getTime() - activeFilter.value * 60 * 60 * 1000);
  }
  return history.value.filter((h) => {
    const t = new Date(h.timestamp);
    return t >= start && t <= end;
  });
});

const activeRange = computed(() => {
  if (activeFilter.value === "custom" && customStart.value && customEnd.value) {
    const start = new Date(customStart.value);
    const end = new Date(customEnd.value);
    end.setHours(23, 59, 59);
    return { start, end };
  }
  const end = new Date();
  const start = new Date(end.getTime() - activeFilter.value * 60 * 60 * 1000);
  return { start, end };
});

const avgCpu = computed(() => {
  if (!activeHistory.value.length) return 0;
  return (activeHistory.value.reduce((acc, h) => acc + h.cpu, 0) / activeHistory.value.length).toFixed(1);
});

const maxCpu = computed(() => {
  if (!activeHistory.value.length) return 0;
  return Math.max(...activeHistory.value.map((h) => h.cpu)).toFixed(1);
});

const avgMem = computed(() => {
  if (!activeHistory.value.length) return 0;
  return (activeHistory.value.reduce((acc, h) => acc + h.memory, 0) / (activeHistory.value.length * 1024 * 1024 * 1024)).toFixed(2);
});

const maxMem = computed(() => {
  if (!activeHistory.value.length) return 0;
  return (Math.max(...activeHistory.value.map((h) => h.memory)) / (1024 * 1024 * 1024)).toFixed(2);
});

const applyCustomRange = () => {
  if (!tempStart.value || !tempEnd.value) {
    modalError.value = "Select both dates";
    return;
  }
  customStart.value = tempStart.value;
  customEnd.value = tempEnd.value;
  activeFilter.value = "custom";
  showCustomModal.value = false;
  updateUrl();
};

const syncStateFromUrl = () => {
  const { range, start, end } = route.query;
  if (range) {
    activeFilter.value = range === "custom" ? "custom" : parseInt(range);
  }
  if (start) customStart.value = start;
  if (end) customEnd.value = end;
};

const updateUrl = () => {
  const query = { ...route.query };
  query.range = activeFilter.value;
  if (activeFilter.value === "custom") {
    query.start = customStart.value;
    query.end = customEnd.value;
  } else {
    delete query.start;
    delete query.end;
  }
  router.replace({ query });
};

const makeChartOptions = (unit) => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 1000, easing: "easeOutQuart" },
  interaction: { mode: "index", intersect: false },
  scales: {
    y: {
      beginAtZero: true,
      grid: { color: isDark.value ? "rgba(255, 255, 255, 0.03)" : "rgba(0, 0, 0, 0.03)" },
      ticks: {
        color: isDark.value ? "rgba(255, 255, 255, 0.4)" : "rgba(0, 0, 0, 0.4)",
        font: { size: 10, family: "JetBrains Mono" },
      },
    },
    x: {
      grid: { display: false },
      ticks: {
        color: isDark.value ? "rgba(255, 255, 255, 0.3)" : "rgba(0, 0, 0, 0.3)",
        font: { size: 9, family: "JetBrains Mono" },
        maxRotation: 0,
        maxTicksLimit: 8,
      },
    },
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDark.value ? "#0f172a" : "#ffffff",
      titleColor: isDark.value ? "rgba(255,255,255,0.5)" : "rgba(0,0,0,0.5)",
      bodyColor: isDark.value ? "#fff" : "#000",
      borderColor: "var(--border)",
      borderWidth: 1,
      padding: 12,
      cornerRadius: 12,
      displayColors: false,
      callbacks: {
        label: (item) => `${item.formattedValue} ${unit}`,
      },
    },
  },
  elements: {
    point: { radius: 0, hoverRadius: 6 },
    line: { tension: 0.4 },
  },
});

const cpuChartOptions = computed(() => {
  const opts = makeChartOptions("%");
  const values = chartData.value.cpu.datasets[0]?.data?.filter((v) => v != null) ?? [];
  const dataMax = values.length ? Math.max(...values) : 0;
  if (dataMax > 0) {
    opts.scales.y.suggestedMax = Math.min(100, Math.ceil(dataMax * 1.15));
  }
  return opts;
});
const memChartOptions = computed(() => {
  const opts = makeChartOptions("GB");
  const values = chartData.value.mem.datasets[0]?.data?.filter((v) => v != null) ?? [];
  const dataMax = values.length ? Math.max(...values) : 0;
  if (dataMax > 0) {
    opts.scales.y.suggestedMax = Math.ceil(dataMax * 1.15 * 100) / 100;
  }
  return opts;
});

watch(
  () => sharedState.theme,
  () => {
    updateCharts();
  },
);

const fetchData = async () => {
  try {
    const token = secureStorage.getItem("token");
    const res = await apiFetch("/api/system/history", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      const data = await res.json();
      // Ensure history is sorted ascending (oldest first) for easier processing
      history.value = data.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
      updateCharts();
    }
  } catch (err) {
    console.error(err);
  }
};

const updateCharts = () => {
  const { start: startTime, end: endTime } = activeRange.value;
  const rangeHours = Math.max((endTime.getTime() - startTime.getTime()) / (1000 * 60 * 60), 1 / 60);
  const now = new Date();

  // 1. Generate fixed timeline bins (e.g., 60 bins for any range)
  const binCount = 60;
  const binSizeMs = (rangeHours * 60 * 60 * 1000) / binCount;
  const timeline = [];

  for (let i = 0; i <= binCount; i++) {
    const t = new Date(startTime.getTime() + i * binSizeMs);
    const isToday = t.toDateString() === now.toDateString();
    const label = isToday
      ? t.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
      : `${t.toLocaleDateString([], { month: "short", day: "numeric" })} ${t.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`;

    timeline.push({
      time: t,
      label,
      cpu: null,
      mem: null,
    });
  }

  // 2. Map filtered history to bins (peak CPU + peak memory per bin, matching summary cards)
  activeHistory.value.forEach((h) => {
    const hTime = new Date(h.timestamp);
    const binIndex = Math.floor((hTime.getTime() - startTime.getTime()) / binSizeMs);
    if (binIndex >= 0 && binIndex <= binCount) {
      const bin = timeline[binIndex];
      bin.cpu = bin.cpu === null ? h.cpu : Math.max(bin.cpu, h.cpu);
      bin.mem = bin.mem === null ? h.memory : Math.max(bin.mem, h.memory);
    }
  });

  const labels = timeline.map((t) => t.label);

  chartData.value.cpu = {
    labels,
    datasets: [
      {
        label: "CPU Load",
        data: timeline.map((t) => t.cpu),
        borderColor: "#0891b2",
        backgroundColor: "rgba(8, 145, 178, 0.12)",
        fill: true,
        borderWidth: 3,
        spanGaps: true,
      },
    ],
  };

  chartData.value.mem = {
    labels,
    datasets: [
      {
        label: "Memory Usage",
        data: timeline.map((t) => (t.mem != null ? t.mem / (1024 * 1024 * 1024) : null)),
        borderColor: "#10b981",
        backgroundColor: "rgba(16, 185, 129, 0.1)",
        fill: true,
        borderWidth: 3,
        spanGaps: true,
      },
    ],
  };
};

watch([activeFilter, customStart, customEnd], () => {
  updateCharts();
  updateUrl();
});

onMounted(() => {
  syncStateFromUrl();
  fetchData();
});
</script>

<style scoped>
.health-container {
  gap: 1.25rem;
}

.coverage-hint {
  color: var(--warning);
  font-weight: 700;
}

.page-filter-pill.is-partial {
  opacity: 0.65;
}

.health-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.chart-section {
  padding: 1.25rem;
  border-radius: var(--radius-2xl);
  border: 1px solid var(--border);
  background: var(--bg-card);
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-header h4 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 900;
  color: var(--text-main);
}

.unit-badge {
  font-size: 0.65rem;
  font-weight: 900;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  padding: 0.25rem 0.6rem;
  background: var(--bg-input);
  border-radius: 6px;
}

.chart-container {
  height: 350px;
  position: relative;
}

.chart-placeholder {
  height: 100%;
  background: var(--bg-input);
  border-radius: 16px;
  overflow: hidden;
}

@media (max-width: 1200px) {
  .health-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .chart-container {
    height: 280px;
  }
}

@media (max-width: 480px) {
  .chart-container {
    height: 240px;
  }
}
</style>
