---
version: 1
slug: "frontend-src-routes-studio-profile-page-svelte"
primary_target: "frontend/src/routes/studio/profile/+page.svelte"
related_targets: ["frontend/src/routes/authors/[nickname]/+page.svelte"]
---

# Профиль автора

Маршрут: `/studio/profile`. Режим: Operate. Автор оформляет страницу перед публикацией или обновляет её и проверяет, как читатели увидят правки.

## Direction contract

THESIS: Автор выбирает точный кадр для каждого изображения и проверяет всю шапку с текущими полями.

OWN-WORLD: Manrope, светлая панель студии, синие основные действия, компактные контурные кнопки и существующая типографика публичной страницы.

STORY: Выбрать файл, передвинуть или приблизить его в рамке, подтвердить кадр и увидеть статус рядом с изображением. Текст сохраняется по кнопке «Сохранить». Первое сохранение публикует профиль и открывает загрузку изображений.

FIRST VIEWPORT: Заголовок и ссылка «Как это видят читатели», затем широкая обложка и квадратный аватар. Текстовые поля и кнопка сохранения идут следом. На узком экране ссылка переносится под заголовок.

FORM: Согласованное расширение существующей студии. Рамки 4:1 и 1:1, кадрирование мышью, касанием и клавиатурой. Предпросмотр в той же вкладке с возвратом к заполненной форме.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance.
