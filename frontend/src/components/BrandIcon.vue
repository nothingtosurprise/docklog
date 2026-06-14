<template>
  <svg
    v-if="icon"
    class="brand-icon"
    :class="[`brand-icon-${name}`, { colored }]"
    :viewBox="icon.viewBox"
    :width="size"
    :height="size"
    aria-hidden="true"
    role="img"
  >
    <path
      v-for="(path, index) in icon.paths"
      :key="index"
      :d="path.d"
      :fill="colored ? path.fill : 'currentColor'"
    />
  </svg>
</template>

<script setup>
import { computed } from "vue";
import { brandIcons } from "../utils/brandIcons";

const props = defineProps({
  name: { type: String, required: true },
  size: { type: [Number, String], default: 20 },
  colored: { type: Boolean, default: true },
});

const icon = computed(() => brandIcons[props.name] ?? null);
</script>

<style scoped>
.brand-icon {
  display: block;
  flex-shrink: 0;
}
</style>
