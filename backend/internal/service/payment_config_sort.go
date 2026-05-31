package service

func compactProductSortUpdates(updates []ProductSortOrderUpdate) ([]int64, map[int64]int) {
	sortOrderByID := make(map[int64]int, len(updates))
	ids := make([]int64, 0, len(updates))
	for _, update := range updates {
		if update.ID <= 0 {
			continue
		}
		if _, exists := sortOrderByID[update.ID]; !exists {
			ids = append(ids, update.ID)
		}
		sortOrderByID[update.ID] = update.SortOrder
	}
	return ids, sortOrderByID
}
