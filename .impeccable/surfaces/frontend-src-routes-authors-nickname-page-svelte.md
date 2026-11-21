---
version: 1
slug: "frontend-src-routes-authors-nickname-page-svelte"
primary_target: "frontend/src/routes/authors/[nickname]/+page.svelte"
related_targets: ["frontend/src/routes/authors/[nickname]/SubscribeDialog.svelte"]
---

# Публичный профиль автора

Маршрут `/authors/:nickname`. Режим Operate. Читатель знакомится с автором, сравнивает уровни и просматривает доступные анонсы публикаций.

## Direction contract

THESIS: Обложка и имя задают характер страницы. Уровни удобно сравнить по цене и описанию, а публикации образуют спокойный список.

OWN-WORLD: Белая основа, Manrope, существующие синие действия, тонкие границы и прохладный фон для статусов. Цвет обложки выбирает автор.

STORY: Узнать автора, найти подходящий уровень и проверить сумму перед оплатой. Читатели с подпиской видят свой уровень. В каждой публикации видны условия доступа.

FIRST VIEWPORT: Ссылка в каталог, широкая обложка, квадратный аватар на нижнем краю, имя и описание. Ниже начинаются карточки уровней. На телефоне карточки идут одной колонкой.

FORM: Расширение существующей страницы. Порядок уровней задаёт автор. Модальное окно сохраняет выбранный уровень и явно показывает сумму и автопродление.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
