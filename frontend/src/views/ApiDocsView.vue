<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 px-6 py-7 text-white dark:border-dark-700">
          <div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <p class="text-sm font-medium uppercase tracking-[0.24em] text-cyan-200">
                {{ t('apiDocs.eyebrow') }}
              </p>
              <h1 class="mt-2 text-3xl font-bold tracking-tight">
                {{ t('apiDocs.title') }}
              </h1>
              <p class="mt-3 max-w-3xl text-sm leading-6 text-slate-300">
                {{ t('apiDocs.description') }}
              </p>
            </div>
            <router-link to="/keys" class="btn bg-white text-slate-950 hover:bg-cyan-50">
              {{ t('apiDocs.createKey') }}
            </router-link>
          </div>
        </div>

        <div class="grid gap-4 border-b border-gray-100 p-6 dark:border-dark-700 md:grid-cols-3">
          <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('apiDocs.step1Title') }}</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('apiDocs.step1Body') }}</p>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('apiDocs.step2Title') }}</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('apiDocs.step2Body') }}</p>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('apiDocs.step3Title') }}</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('apiDocs.step3Body') }}</p>
          </div>
        </div>

        <article class="api-docs markdown-body p-6" v-html="renderedDoc"></article>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'
import { sub2apiUserApiDoc } from '@/content/sub2apiUserApiDoc'

const { t } = useI18n()

marked.setOptions({
  breaks: true,
  gfm: true
})

const renderedDoc = computed(() => {
  const html = marked.parse(sub2apiUserApiDoc) as string
  return DOMPurify.sanitize(html)
})
</script>

<style scoped>
.api-docs :deep(h1) {
  @apply mb-4 text-3xl font-bold text-gray-950 dark:text-white;
}

.api-docs :deep(h2) {
  @apply mt-8 border-t border-gray-100 pt-6 text-xl font-semibold text-gray-950 dark:border-dark-700 dark:text-white;
}

.api-docs :deep(h3) {
  @apply mt-6 text-lg font-semibold text-gray-900 dark:text-white;
}

.api-docs :deep(p) {
  @apply my-3 leading-7 text-gray-700 dark:text-dark-200;
}

.api-docs :deep(ul),
.api-docs :deep(ol) {
  @apply my-3 space-y-2 pl-6 text-gray-700 dark:text-dark-200;
}

.api-docs :deep(ul) {
  @apply list-disc;
}

.api-docs :deep(ol) {
  @apply list-decimal;
}

.api-docs :deep(code) {
  @apply rounded bg-slate-100 px-1.5 py-0.5 font-mono text-sm text-slate-900 dark:bg-dark-700 dark:text-cyan-100;
}

.api-docs :deep(pre) {
  @apply my-4 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-sm shadow-inner;
}

.api-docs :deep(pre code) {
  @apply bg-transparent p-0 text-slate-100;
}

.api-docs :deep(blockquote) {
  @apply my-4 rounded-r-xl border-l-4 border-cyan-500 bg-cyan-50 px-4 py-3 text-cyan-950 dark:bg-cyan-950/30 dark:text-cyan-100;
}
</style>
