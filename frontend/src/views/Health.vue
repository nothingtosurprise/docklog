<template>
  <div class="health-page">
    <div class="health-sub-bar glass">
      <div class="h-left">
        <div class="title-group">
          <h1>System Health</h1>
          <p>Historical resource utilization & diagnostics</p>
        </div>
      </div>
      <div class="h-filters glass">
        <div class="range-copy">
          <span class="range-kicker">TIME WINDOW</span>
          <h3>{{ activeRangeLabel }}</h3>
        </div>

        <div class="filter-pills">
          <button
            v-for="f in filters"
            :key="f.label"
            @click="activeFilter = f.value"
            :class="['range-pill', { active: activeFilter === f.value }]"
          >
            <span class="pill-value">{{ f.short }}</span>
            <span class="pill-note">{{ f.note }}</span>
          </button>
        </div>

        <button
          class="custom-range-btn"
          @click="showCustomModal = true"
          :class="{ active: activeFilter === 'custom' }"
        >
          <svg
            viewBox="0 0 24 24"
            width="14"
            height="14"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <circle cx="12" cy="12" r="9"></circle>
            <path d="M12 7v5l3 3"></path>
          </svg>
          Custom
        </button>
      </div>
    </div>

    <Transition name="fade">
      <div
        v-if="showCustomModal"
        class="modal-overlay flex-center"
        @click.self="showCustomModal = false"
      >
        <div class="custom-modal glass shadow-2xl">
          <div class="modal-header">
            <h3>Custom Time Range</h3>
            <button class="close-btn" @click="showCustomModal = false">
              &times;
            </button>
          </div>
          <div class="modal-body">
            <p class="modal-hint">
              Select a range within the last 30 days to analyze historical
              performance.
            </p>
            <div class="range-grid">
              <div class="date-group">
                <label>START DATE</label>
                <div class="input-box">
                  <input type="date" v-model="tempStart" :max="today" />
                </div>
              </div>
              <div class="date-group">
                <label>END DATE</label>
                <div class="input-box">
                  <input type="date" v-model="tempEnd" :max="today" />
                </div>
              </div>
            </div>
            <div v-if="modalError" class="modal-error fade-in">
              {{ modalError }}
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showCustomModal = false">
              Cancel
            </button>
            <button class="btn-apply" @click="applyCustomRange">
              Apply Custom Range
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <div v-if="history.length" class="stats-summary">
      <div class="summary-card glass">
        <span class="s-lbl">AVG CPU</span>
        <span class="s-val">{{ avgCpu }}%</span>
      </div>
      <div class="summary-card glass">
        <span class="s-lbl">PEAK CPU</span>
        <span class="s-val warning">{{ maxCpu }}%</span>
      </div>
      <div class="summary-card glass">
        <span class="s-lbl">AVG MEMORY</span>
        <span class="s-val success">{{ avgMem }} GB</span>
      </div>
      <div class="summary-card glass">
        <span class="s-lbl">PEAK MEMORY</span>
        <span class="s-val">{{ maxMem }} GB</span>
      </div>
    </div>

    <div class="health-grid">
      <div class="chart-container glass">
        <div class="chart-header">
          <h3>CPU Performance History</h3>
          <span class="unit">Utilization %</span>
        </div>
        <div class="chart-wrapper">
          <Line
            v-if="chartData.cpu.labels.length"
            :data="chartData.cpu"
            :options="cpuChartOptions"
          />
          <div v-else class="chart-placeholder">
            Processing historical CPU data...
          </div>
        </div>
      </div>

      <div class="chart-container glass">
        <div class="chart-header">
          <h3>Memory Footprint</h3>
          <span class="unit">Gigabytes (GB)</span>
        </div>
        <div class="chart-wrapper">
          <Line
            v-if="chartData.mem.labels.length"
            :data="chartData.mem"
            :options="memChartOptions"
          />
          <div v-else class="chart-placeholder">
            Processing historical Memory data...
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from "vue";
import { useRouter } from "vue-router";
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

const router = useRouter();
const isDark = computed(() => sharedState.theme === "dark");

const filters = [
  { label: "1H", short: "1H", note: "Last hour", value: 1 },
  { label: "3H", short: "3H", note: "Last 3 hours", value: 3 },
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

const activeRangeLabel = computed(() => {
  if (activeFilter.value === "custom") return "Custom range";
  const preset = filters.find((f) => f.value === activeFilter.value);
  return preset ? preset.note : "Last 24 hours";
});

let systemStatsInterval = null;

const applyCustomRange = () => {
  if (!tempStart.value || !tempEnd.value) {
    modalError.value = "Please select both start and end dates";
    return;
  }

  const start = new Date(tempStart.value);
  const end = new Date(tempEnd.value);
  const diffDays = (end - start) / (1000 * 60 * 60 * 24);

  if (start > end) {
    modalError.value = "Start date must be before end date";
    return;
  }

  if (diffDays > 30) {
    modalError.value = "Maximum range is 30 days";
    return;
  }

  modalError.value = "";
  customStart.value = tempStart.value;
  customEnd.value = tempEnd.value;
  activeFilter.value = "custom";
  showCustomModal.value = false;
};

const history = ref([]);

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

const formatChartTick = (timestamp) => {
  const date = new Date(timestamp);
  if (activeFilter.value === "custom") {
    return date.toLocaleDateString([], {
      month: "short",
      day: "numeric",
    });
  }
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
};

const avgCpu = computed(() => {
  if (!activeHistory.value.length) return 0;
  const sum = activeHistory.value.reduce((acc, h) => acc + h.cpu, 0);
  return (sum / activeHistory.value.length).toFixed(1);
});

const maxCpu = computed(() => {
  if (!activeHistory.value.length) return 0;
  return Math.max(...activeHistory.value.map((h) => h.cpu)).toFixed(1);
});

const avgMem = computed(() => {
  if (!activeHistory.value.length) return 0;
  const sum = activeHistory.value.reduce((acc, h) => acc + h.memory, 0);
  return (sum / (activeHistory.value.length * 1024 * 1024 * 1024)).toFixed(2);
});

const maxMem = computed(() => {
  if (!activeHistory.value.length) return 0;
  const max = Math.max(...activeHistory.value.map((h) => h.memory));
  return (max / (1024 * 1024 * 1024)).toFixed(2);
});

const makeChartOptions = (unit) =>
  computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    animation: { duration: 800, easing: "easeOutQuart" },
    interaction: { mode: "index", intersect: false },
    scales: {
      y: {
        beginAtZero: true,
        grid: {
          color: isDark.value
            ? "rgba(255, 255, 255, 0.04)"
            : "rgba(0, 0, 0, 0.06)",
        },
        ticks: {
          color: isDark.value
            ? "rgba(255, 255, 255, 0.4)"
            : "rgba(0, 0, 0, 0.4)",
          font: { size: 10, weight: "600", family: "JetBrains Mono" },
        },
      },
      x: {
        type: "linear",
        bounds: "data",
        grid: { display: false },
        ticks: {
          color: isDark.value
            ? "rgba(255, 255, 255, 0.35)"
            : "rgba(0, 0, 0, 0.35)",
          font: { size: 9, weight: "600", family: "JetBrains Mono" },
          maxRotation: 0,
          maxTicksLimit: 12,
          callback: (value) => formatChartTick(Number(value)),
        },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: isDark.value
          ? "rgba(15, 23, 42, 0.95)"
          : "rgba(255, 255, 255, 0.95)",
        titleColor: isDark.value ? "rgba(255,255,255,0.5)" : "rgba(0,0,0,0.4)",
        bodyColor: isDark.value ? "#f1f5f9" : "#0f172a",
        borderColor: isDark.value
          ? "rgba(255,255,255,0.08)"
          : "rgba(0,0,0,0.08)",
        borderWidth: 1,
        padding: { top: 10, bottom: 10, left: 14, right: 14 },
        cornerRadius: 10,
        displayColors: false,
        titleFont: { size: 10, weight: "600", family: "JetBrains Mono" },
        bodyFont: { size: 14, weight: "800", family: "Outfit" },
        titleMarginBottom: 6,
        callbacks: {
          title: (items) =>
            items[0]?.parsed?.x ? formatChartTick(items[0].parsed.x) : "",
          label: (item) => `${item.formattedValue} ${unit}`,
        },
      },
    },
    elements: {
      point: {
        radius: 0,
        hoverRadius: 5,
        hoverBackgroundColor: unit === "%" ? "#6366f1" : "#10b981",
        hoverBorderColor: isDark.value ? "#0f172a" : "#fff",
        hoverBorderWidth: 3,
      },
      line: { tension: 0.12 },
    },
  }));

const cpuChartOptions = makeChartOptions("%");
const memChartOptions = makeChartOptions("GB");

const fetchData = async () => {
  try {
    const token = secureStorage.getItem("token");
    const res = await fetch("/api/system/history", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      const data = await res.json();
      history.value = data.reverse();
      updateCharts();
    }
  } catch (err) {
    console.error(err);
  }
};

const updateCharts = () => {
  const selectedHistory = [...activeHistory.value].sort(
    (a, b) => new Date(a.timestamp) - new Date(b.timestamp),
  );

  const labels = selectedHistory.map((item) => formatChartTick(item.timestamp));
  const cpuData = selectedHistory.map((item) => ({
    x: new Date(item.timestamp).getTime(),
    y: item.cpu,
  }));
  const memData = selectedHistory.map((item) => ({
    x: new Date(item.timestamp).getTime(),
    y: item.memory / (1024 * 1024 * 1024),
  }));

  chartData.value.cpu = {
    labels,
    datasets: [
      {
        label: "CPU %",
        data: cpuData,
        parsing: false,
        borderColor: "#6366f1",
        backgroundColor: (ctx) => {
          const canvas = ctx.chart.ctx;
          const gradient = canvas.createLinearGradient(0, 0, 0, 400);
          gradient.addColorStop(0, "rgba(99, 102, 241, 0.2)");
          gradient.addColorStop(1, "rgba(99, 102, 241, 0)");
          return gradient;
        },
        fill: true,
        borderWidth: 3,
        pointRadius: 0,
        spanGaps: false,
      },
    ],
  };

  chartData.value.mem = {
    labels,
    datasets: [
      {
        label: "Memory (GB)",
        data: memData,
        parsing: false,
        borderColor: "#10b981",
        backgroundColor: (ctx) => {
          const canvas = ctx.chart.ctx;
          const gradient = canvas.createLinearGradient(0, 0, 0, 400);
          gradient.addColorStop(0, "rgba(16, 185, 129, 0.15)");
          gradient.addColorStop(1, "rgba(16, 185, 129, 0)");
          return gradient;
        },
        fill: true,
        borderWidth: 3,
        pointRadius: 0,
        spanGaps: false,
      },
    ],
  };
};

watch([activeFilter, customStart, customEnd], updateCharts);
const handleGlobalClick = () => {
  showUserMenu.value = false;
};

onMounted(() => {
  fetchData();
  window.addEventListener("click", handleGlobalClick);
});

onUnmounted(() => {
  window.removeEventListener("click", handleGlobalClick);
});
</script>

<style scoped>
.health-view {
  padding: 0;
  min-height: 100vh;
  background: var(--bg-main);
  animation: fadeIn 0.6s cubic-bezier(0.23, 1, 0.32, 1);
}

/* Unified Header Styles */
.main-header {
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 3rem;
  background: var(--bg-sidebar);
  border-bottom: 1px solid var(--border);
  position: relative;
  z-index: 1000;
  flex-shrink: 0;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  flex: 1.5;
}
.back-btn-header {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  color: var(--text-dim);
  border: 1px solid var(--border);
  cursor: pointer;
  transition: all 0.2s;
  background: var(--bg-input);
}
.back-btn-header:hover {
  color: var(--text-main);
  border-color: var(--text-mute);
  transform: translateX(-2px);
  background: var(--bg-card);
}
.logo-link {
  text-decoration: none;
  cursor: pointer;
  transition: opacity 0.2s;
  display: flex;
  align-items: center;
}
.logo-link:hover {
  opacity: 0.85;
}
.logo-area {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}
.logo-box {
  width: 34px;
  height: 34px;
  background: linear-gradient(135deg, var(--accent), #4f46e5);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  box-shadow: 0 8px 16px -4px rgba(99, 102, 241, 0.4);
}
.logo-box img {
  width: 20px;
  height: 20px;
  object-fit: contain;
  filter: brightness(0) invert(1) !important;
}
.logo-area h1 {
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  margin: 0;
  color: var(--text-main);
  background: linear-gradient(to bottom, var(--text-main), var(--text-dim));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  line-height: 1.1;
}
.divider {
  width: 1px;
  height: 32px;
  background: var(--border);
  opacity: 0.5;
}
.search-wrapper {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1.5rem;
  border-radius: 16px;
  width: 100%;
  max-width: 400px;
  border: 1px solid var(--border);
  background: var(--bg-input);
}
.search-icon {
  width: 18px;
  height: 18px;
  color: var(--text-mute);
}
.search-input {
  background: transparent;
  border: none;
  color: var(--text-main);
  outline: none;
  font-size: 0.85rem;
  width: 100%;
  font-weight: 600;
}
.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
}
.header-stats-group {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.stat-box {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.6rem 1.25rem;
  border-radius: 14px;
  border: 1px solid var(--border);
  background: var(--bg-input);
}
.stat-label {
  font-size: 0.65rem;
  font-weight: 900;
  color: var(--text-mute);
  letter-spacing: 0.1em;
}
.stat-value {
  font-size: 0.85rem;
  font-weight: 800;
  font-family: var(--font-mono);
  color: var(--text-main);
}
.header-right {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  flex: 1.5;
  justify-content: flex-end;
}
/* Health Sub Bar */
.health-sub-bar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  padding: 1.35rem 3rem;
  background: var(--bg-main);
  border-bottom: 1px solid var(--border);
}
.h-left {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  padding-top: 0.1rem;
  flex: 0 0 auto;
}
.back-btn {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-main);
  transition: all 0.2s;
  text-decoration: none;
}
.back-btn:hover {
  transform: translateX(-4px);
  background: var(--bg-card);
  border-color: var(--text-mute);
}
.title-group h1 {
  font-size: 1.5rem;
  font-weight: 900;
  color: var(--text-main);
  margin: 0 0 0.25rem 0;
  letter-spacing: -0.02em;
}
.title-group p {
  font-size: 0.85rem;
  color: var(--text-dim);
  margin: 0;
  font-weight: 500;
}

.h-filters {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.9rem 1rem;
  padding: 0.8rem 0.9rem;
  border-radius: 22px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(255, 255, 255, 0.72)),
    var(--bg-input);
  border: 1px solid var(--border);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.05);
  min-width: 0;
  flex: 1 1 760px;
}
.range-copy {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 126px;
  padding-right: 0.2rem;
  flex: 0 0 auto;
}
.range-kicker {
  font-size: 0.65rem;
  font-weight: 900;
  letter-spacing: 0.16em;
  color: var(--text-mute);
}
.range-copy h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 900;
  color: var(--text-main);
  letter-spacing: -0.02em;
}
.filter-pills {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: nowrap;
  flex: 1 1 auto;
  min-width: 0;
  overflow-x: auto;
  padding: 0.2rem 0.1rem 0.25rem;
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.filter-pills::-webkit-scrollbar {
  display: none;
}
.range-pill {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.1rem;
  min-width: 92px;
  padding: 0.68rem 0.88rem;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.65);
  color: var(--text-mute);
  text-align: center;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    background 0.2s ease;
}
.range-pill:hover {
  transform: translateY(-1px);
  border-color: rgba(99, 102, 241, 0.35);
  box-shadow: 0 10px 24px rgba(99, 102, 241, 0.08);
}
.range-pill.active {
  background:
    linear-gradient(180deg, rgba(99, 102, 241, 0.16), rgba(99, 102, 241, 0.1)),
    rgba(255, 255, 255, 0.9);
  border-color: rgba(99, 102, 241, 0.46);
  color: var(--accent);
  box-shadow: 0 10px 24px rgba(99, 102, 241, 0.12);
}
.pill-value {
  font-size: 0.92rem;
  font-weight: 950;
  letter-spacing: 0.02em;
}
.pill-note {
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--text-dim);
  white-space: nowrap;
}
.range-pill.active .pill-note {
  color: color-mix(in srgb, var(--accent) 75%, white);
}
.custom-range-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.74rem 0.9rem;
  border-radius: 14px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.78);
  color: var(--text-main);
  font-size: 0.76rem;
  font-weight: 900;
  letter-spacing: 0.03em;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    color 0.2s ease;
}
.custom-range-btn:hover {
  transform: translateY(-1px);
  border-color: rgba(99, 102, 241, 0.35);
  box-shadow: 0 10px 24px rgba(99, 102, 241, 0.08);
}
.custom-range-btn.active {
  border-color: rgba(99, 102, 241, 0.4);
  color: var(--accent);
  box-shadow: 0 10px 24px rgba(99, 102, 241, 0.12);
}

.stats-summary {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
  padding: 1.5rem 3rem;
}
.summary-card {
  padding: 1.5rem;
  border-radius: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  transition: all 0.3s;
}
.summary-card:hover {
  transform: translateY(-4px);
  border-color: var(--accent);
}
.s-lbl {
  font-size: 0.75rem;
  font-weight: 900;
  color: var(--text-mute);
  letter-spacing: 0.1em;
}
.s-val {
  font-size: 1.75rem;
  font-weight: 950;
  color: var(--text-main);
}
.s-val.success {
  color: var(--success);
}
.s-val.warning {
  color: var(--warning);
}

.health-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 72px);
  overflow-y: auto;
  background: var(--bg-main);
}
.health-grid {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  padding: 0 3rem 3rem 3rem;
}
.chart-container {
  background: var(--bg-sidebar);
  padding: 2rem;
  border-radius: 24px;
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.chart-header h3 {
  font-size: 1.1rem;
  font-weight: 900;
  color: var(--text-main);
  margin: 0;
}
.unit {
  font-size: 0.75rem;
  color: var(--text-mute);
  font-weight: 800;
}
.chart-wrapper {
  min-height: 350px;
  position: relative;
}
.chart-placeholder {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-mute);
  opacity: 0.5;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(10px);
  z-index: 5000;
}
.custom-modal {
  width: 100%;
  max-width: 500px;
  background: var(--bg-card);
  border-radius: 28px;
  border: 1px solid var(--border);
  overflow: hidden;
  animation: modalIn 0.4s cubic-bezier(0.23, 1, 0.32, 1);
}
.modal-header {
  padding: 1.5rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--border);
}
.modal-header h3 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 900;
  color: var(--text-main);
}
.close-btn {
  background: none;
  border: none;
  color: var(--text-mute);
  font-size: 1.5rem;
  cursor: pointer;
}
.modal-body {
  padding: 2rem;
}
.modal-hint {
  font-size: 0.85rem;
  color: var(--text-dim);
  margin-bottom: 2rem;
  line-height: 1.5;
}
.range-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}
.date-group {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.date-group label {
  font-size: 0.65rem;
  font-weight: 900;
  color: var(--text-mute);
  letter-spacing: 0.1em;
}
.input-box {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 0.75rem 1rem;
}
.input-box input {
  width: 100%;
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 0.9rem;
  font-weight: 700;
  outline: none;
}
.modal-error {
  margin-top: 1.5rem;
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 12px;
  color: var(--error);
  font-size: 0.85rem;
  font-weight: 700;
  text-align: center;
}
.modal-footer {
  padding: 1.5rem 2rem;
  background: var(--bg-input);
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
}
.btn-cancel {
  padding: 0.75rem 1.5rem;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 12px;
  color: var(--text-main);
  font-weight: 700;
  cursor: pointer;
}
.btn-apply {
  padding: 0.75rem 1.5rem;
  background: var(--accent);
  border: none;
  border-radius: 12px;
  color: #fff;
  font-weight: 800;
  cursor: pointer;
  box-shadow: 0 10px 20px rgba(99, 102, 241, 0.3);
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(15px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1400px) {
  .stats-summary {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 1024px) {
  .main-header {
    padding: 0 1.25rem;
  }
  .search-wrapper, .divider {
    display: none !important;
  }
  .stats-summary {
    padding: 1rem 1.25rem;
    gap: 1rem;
  }
  .health-grid {
    padding: 0 1.25rem 2rem 1.25rem;
  }
  .header-stats-group {
    display: none !important;
  }
}

@media (max-width: 768px) {
  .stats-summary {
    grid-template-columns: 1fr;
    padding: 1rem 1.25rem;
    gap: 0.75rem;
  }
  .health-sub-bar {
    padding: 1.5rem 1.25rem;
    flex-direction: column;
    align-items: flex-start;
    text-align: left;
    gap: 1.5rem;
  }
  .h-left {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }
  .h-filters {
    width: 100%;
    gap: 0.7rem;
    padding: 0.85rem;
    grid-template-columns: 1fr;
    justify-items: stretch;
  }
  .range-copy {
    width: 100%;
    min-width: 0;
    padding-right: 0;
  }
  .filter-pills {
    width: 100%;
    flex: 1 1 100%;
    overflow-x: visible;
    flex-wrap: wrap;
  }
  .range-pill {
    flex: 1 1 calc(50% - 0.35rem);
    min-width: 0;
    padding: 0.7rem 0.75rem;
  }
  .custom-range-btn {
    width: 100%;
    justify-content: center;
  }
  .chart-container {
    padding: 1.25rem;
    border-radius: 20px;
  }
  .chart-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }
  .chart-wrapper {
    min-height: 250px;
  }
  .range-grid {
    grid-template-columns: 1fr;
  }
  .health-grid {
    padding: 0 1.25rem 2rem 1.25rem;
    gap: 1.25rem;
  }
}

@media (max-width: 480px) {
  .range-pill {
    flex-basis: 100%;
  }
  .custom-range-btn {
    width: 100%;
  }
  .health-sub-bar {
    padding-left: 1rem;
    padding-right: 1rem;
  }
}

</style>
