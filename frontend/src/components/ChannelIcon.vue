<template>
  <BrandIcon :name="iconName" :size="size" :colored="colored" />
</template>

<script setup>
import { computed } from "vue";
import BrandIcon from "./BrandIcon.vue";
import { channelIconMap } from "../utils/brandIcons";

const props = defineProps({
  type: { type: String, required: true },
  size: { type: [Number, String], default: 20 },
  colored: { type: Boolean, default: true },
});

const iconName = computed(() => channelIconMap[props.type] || "smtp");
</script>
