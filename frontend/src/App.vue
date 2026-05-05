<template>
  <div :data-theme="sharedState.theme">
    <MainLayout v-if="route.meta.layout === 'main'">
      <router-view v-slot="{ Component }">
        <Transition name="fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </router-view>
    </MainLayout>
    <router-view v-else v-slot="{ Component }">
      <Transition name="fade" mode="out-in">
        <component :is="Component" />
      </Transition>
    </router-view>
  </div>
</template>

<script setup>
import { onMounted } from 'vue';
import { useRoute } from 'vue-router';
import MainLayout from './components/MainLayout.vue';
import { sharedState } from './utils/sharedState';

const route = useRoute();

onMounted(() => {
  document.documentElement.setAttribute('data-theme', sharedState.theme);
});
</script>

<style>
/* Transitions */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
