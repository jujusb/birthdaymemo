import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  // 多选标签筛选：空数组 = 全部
  const selectedTagIds = ref<number[]>([])
  // increment to trigger tag list refresh
  const tagRefreshKey = ref(0)
  // increment to trigger birthday list / upcoming refresh
  const birthdayRefreshKey = ref(0)

  function toggleSelectedTag(id: number) {
    const idx = selectedTagIds.value.indexOf(id)
    if (idx >= 0) {
      selectedTagIds.value.splice(idx, 1)
    } else {
      selectedTagIds.value.push(id)
    }
  }

  function clearSelectedTags() {
    selectedTagIds.value = []
  }

  function bumpTags() {
    tagRefreshKey.value++
  }

  // 新建/编辑/删除生日后调用，触发顶栏与侧栏刷新
  function bumpBirthdays() {
    birthdayRefreshKey.value++
  }

  return {
    selectedTagIds,
    tagRefreshKey,
    birthdayRefreshKey,
    toggleSelectedTag,
    clearSelectedTags,
    bumpTags,
    bumpBirthdays,
  }
})
