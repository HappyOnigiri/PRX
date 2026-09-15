package app

import (
	"fmt"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// applyTaskLabelOverrides は scope 固有の文字・色を項目ごとに部分更新する。
func applyTaskLabelOverrides(
	current *domain.TaskLabelOverrides,
	update domain.TaskLabelOverridesUpdate,
	scope string,
) error {
	if current == nil {
		return fmt.Errorf("task label overrides target is nil")
	}
	if *current == nil {
		*current = domain.TaskLabelOverrides{}
	}
	for key, item := range update {
		normalizedKey := domain.TaskLabelKey(strings.ToLower(strings.TrimSpace(string(key))))
		if !isTaskLabelKey(normalizedKey) {
			return domain.NewError(
				domain.DomainErrorCodeInvalidTaskLabel,
				"%s task label key %q is invalid",
				scope,
				key,
			)
		}
		value := (*current)[normalizedKey]
		if item.Text != nil {
			text, err := domain.NormalizeTaskLabelText(*item.Text)
			if err != nil {
				return domain.NewError(
					domain.DomainErrorCodeInvalidTaskLabel,
					"%s task label %s text: %s",
					scope,
					key,
					err,
				)
			}
			value.Text = text
		}
		if item.Color != nil {
			color, err := domain.NormalizeTaskLabelColor(*item.Color)
			if err != nil {
				return domain.NewError(
					domain.DomainErrorCodeInvalidTaskLabel,
					"%s task label %s color: %s",
					scope,
					key,
					err,
				)
			}
			value.Color = color
		}
		if value == (domain.TaskLabelOverride{}) {
			delete(*current, normalizedKey)
		} else {
			(*current)[normalizedKey] = value
		}
	}
	return nil
}

func isTaskLabelKey(key domain.TaskLabelKey) bool {
	for _, candidate := range domain.TaskLabelKeys() {
		if key == candidate {
			return true
		}
	}
	return false
}
