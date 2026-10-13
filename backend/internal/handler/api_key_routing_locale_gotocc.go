package handler

import "github.com/LuckyKuang/sub2api-plus/internal/service"

func localizeRoutingPriorityGroups(groups []service.AutoRoutePriorityGroup, language string) {
	for i := range groups {
		groups[i].Name = groups[i].I18n.LocalizeName(language, groups[i].Name)
	}
}
