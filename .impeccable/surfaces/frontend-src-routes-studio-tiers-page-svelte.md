---
version: 1
slug: "frontend-src-routes-studio-tiers-page-svelte"
primary_target: "frontend/src/routes/studio/tiers/+page.svelte"
related_targets: ["frontend/src/routes/studio/tiers/TierDialog.svelte"]
---

# Уровни подписки

Маршрут `/studio/tiers`. Режим Operate. Автор управляет ценой, описанием и порядком уровней подписки.

## Direction contract

THESIS. Порядок уровней определяет доступ к постам. Строки показывают, когда цена расходится с порядком или ниже минимума для подписки.

OWN-WORLD. Manrope, белые панели Студии, синий цвет основного действия и тонкие разделители строк.

STORY. Автор добавляет или изменяет уровень в диалоге. Он проверяет черновик порядка перед сохранением и подтверждает удаление или архивацию в именованном диалоге.

FIRST VIEWPORT. Заголовок и счётчик стоят рядом с кнопкой «Новый уровень». Под ними расположены подсказка о доступе и кнопка «Сохранить порядок». В строке после ручки идут название и описание, справа видны цена и действия. На узком экране цена и действия стоят под названием. Ручку используемого уровня заменяет замок.

FORM. Ручка и команды «Выше» и «Ниже» используют одинаковые границы перемещения. При конфликте сохранения уведомление сопровождает обновление списка. Серые строки архива открываются под активными уровнями. seed: none (unavailable)

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
