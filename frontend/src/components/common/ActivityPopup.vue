<template>
  <Teleport to="body">
    <Transition name="activity-popup">
      <div
        v-if="showPopup"
        class="fixed inset-0 z-[115] flex items-start justify-center overflow-y-auto bg-black/65 p-4 pt-[8vh] backdrop-blur-sm"
        @click.self="closeForSession"
      >
        <section
          class="w-full max-w-[680px] overflow-hidden rounded-3xl bg-white shadow-2xl ring-1 ring-black/5 dark:bg-dark-800 dark:ring-white/10"
          aria-modal="true"
          role="dialog"
          :aria-label="t('activities.popup.title')"
          data-testid="activity-popup"
        >
          <header class="relative overflow-hidden border-b border-primary-100 bg-primary-50/70 px-6 py-6 dark:border-primary-500/20 dark:bg-primary-500/10 sm:px-8">
            <div class="pointer-events-none absolute -right-10 -top-12 h-40 w-40 rounded-full bg-primary-300/25 blur-3xl dark:bg-primary-500/15"></div>
            <div class="relative flex items-start justify-between gap-5">
              <div class="flex min-w-0 items-start gap-4">
                <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary-600 text-white shadow-lg shadow-primary-500/25">
                  <Icon name="sparkles" size="lg" />
                </span>
                <div>
                  <h2 class="text-xl font-black text-gray-950 dark:text-white sm:text-2xl">
                    {{ t('activities.popup.title') }}
                  </h2>
                  <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                    {{ t('activities.popup.subtitle', { count: popupActivities.length }) }}
                  </p>
                </div>
              </div>
              <button
                type="button"
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-gray-500 transition hover:bg-white/80 hover:text-gray-800 dark:text-gray-300 dark:hover:bg-dark-700"
                :aria-label="t('common.close')"
                @click="closeForSession"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </header>

          <div class="max-h-[52vh] space-y-3 overflow-y-auto px-6 py-6 sm:px-8">
            <button
              v-for="activity in popupActivities"
              :key="activity.id"
              type="button"
              class="group flex w-full items-center gap-4 rounded-2xl border border-gray-200 p-4 text-left transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-700 dark:hover:border-primary-500/40 dark:hover:bg-primary-500/5"
              :data-testid="`activity-popup-item-${activity.id}`"
              @click="openActivities(activity.id)"
            >
              <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                <Icon name="gift" size="md" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block font-black text-gray-950 dark:text-white">
                  {{ activityTitle(activity) }}
                </span>
                <span class="mt-1 block text-sm leading-5 text-gray-500 dark:text-gray-400">
                  {{ activitySummary(activity) }}
                </span>
              </span>
              <Icon name="chevronRight" size="md" class="shrink-0 text-gray-400 transition-transform group-hover:translate-x-1 group-hover:text-primary-500" />
            </button>
          </div>

          <footer class="flex flex-col-reverse gap-3 border-t border-gray-100 bg-gray-50/70 px-6 py-5 dark:border-dark-700 dark:bg-dark-900/30 sm:flex-row sm:justify-end sm:px-8">
            <button type="button" class="btn btn-secondary" @click="closeForSession">
              {{ t('activities.popup.later') }}
            </button>
            <button type="button" class="btn btn-primary" data-testid="activity-popup-view-all" @click="openActivities()">
              {{ t('activities.popup.viewAll') }}
              <Icon name="arrowRight" size="sm" />
            </button>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useActivityStore } from '@/stores/activities'
import { useAnnouncementStore } from '@/stores/announcements'
import type { UserActivity } from '@/types/payment'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const activityStore = useActivityStore()
const announcementStore = useAnnouncementStore()
const { popupActivities } = storeToRefs(activityStore)

const showPopup = computed(() =>
  popupActivities.value.length > 0
  && !announcementStore.hasPendingPopups
  && route.path !== '/activities'
)

function activityTitle(activity: UserActivity): string {
  if (activity.type === 'first_recharge') return t('activities.firstRecharge.title')
  if (activity.type === 'daily_check_in') return t('activities.dailyCheckIn.title')
  return t('activities.unknownTitle')
}

function activitySummary(activity: UserActivity): string {
  if (activity.type === 'first_recharge') {
    return activity.first_recharge?.purchase_mode === 'product_link'
      ? t('activities.firstRecharge.productLinkSummary')
      : t('activities.firstRecharge.internalPaymentSummary')
  }
  if (activity.type === 'daily_check_in') return t('activities.dailyCheckIn.summary')
  return t('activities.unknownSummary')
}

function closeForSession() {
  activityStore.dismissPopupForSession()
}

function openActivities(activityId?: string) {
  activityStore.dismissPopupForSession()
  router.push({
    path: '/activities',
    query: activityId ? { activity: activityId } : undefined,
  })
}

watch(showPopup, (visible) => {
  if (visible) {
    document.body.style.overflow = 'hidden'
  } else if (!announcementStore.hasPendingPopups) {
    document.body.style.overflow = ''
  }
})

onBeforeUnmount(() => {
  if (showPopup.value) document.body.style.overflow = ''
})
</script>

<style scoped>
.activity-popup-enter-active,
.activity-popup-leave-active {
  transition: opacity 0.2s ease;
}

.activity-popup-enter-from,
.activity-popup-leave-to {
  opacity: 0;
}

.activity-popup-enter-active section,
.activity-popup-leave-active section {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.activity-popup-enter-from section,
.activity-popup-leave-to section {
  opacity: 0;
  transform: translateY(-10px) scale(0.97);
}
</style>
