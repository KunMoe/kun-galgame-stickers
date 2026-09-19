package service

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/perm"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) List(ctx context.Context, q dto.ListQuery, v Viewer) (*dto.PackListPage, *errors.AppError) {
	statuses := []int16{model.PackPublished}
	if q.OwnerUID > 0 && (q.OwnerUID == v.UID || perm.Can(v.Roles, perm.PackViewHidden)) {
		statuses = []int16{model.PackDraft, model.PackPublished, model.PackHidden}
	}

	rows, total, err := s.packs.List(repository.ListParams{
		OwnerUID:       q.OwnerUID,
		Statuses:       statuses,
		OfficialOnly:   q.OfficialOnly,
		SFWOnly:        q.Rating == dto.RatingFilterSFW,
		Search:         q.Search,
		TagSlug:        q.Tag,
		LinkedOnly:     q.LinkedOnly,
		CatalogWork:    q.CatalogWorkID,
		AnyCatalogWork: q.AnyCatalogWorkID,
		Order:          orderFor(q.Sort),
		Offset:         q.Offset,
		Limit:          q.Limit,
	})
	if err != nil {
		return nil, errors.ErrInternal("failed to list packs")
	}

	packs, appErr := s.hydrate(ctx, rows)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.PackListPage{Packs: packs, Total: total}, nil
}

// Every order ends with id: uuidv7 is monotonic, so it is a stable tiebreaker.
// Without one, the seven official packs share a published_at and paging through
// ties can repeat or skip rows.
func orderFor(sort string) string {
	switch sort {
	case dto.SortHot:
		return "download_count DESC, view_count DESC, published_at DESC NULLS LAST, id DESC"
	case dto.SortUpdated:
		return "updated_at DESC, id DESC"
	default:
		return "published_at DESC NULLS LAST, created_at DESC, id DESC"
	}
}

// hydrate turns a page of rows into DTOs with four batch lookups, whatever the
// page size. The previous implementation ran one count over the whole table
// plus one cover query per pack.
func (s *Service) hydrate(ctx context.Context, rows []model.Pack) ([]dto.Pack, *errors.AppError) {
	out := make([]dto.Pack, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}

	packIDs := make([]uuid.UUID, 0, len(rows))
	coverIDs := make([]uuid.UUID, 0, len(rows))
	ownerIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		packIDs = append(packIDs, row.ID)
		ownerIDs = append(ownerIDs, row.OwnerUID)
		if row.CoverStickerID != nil {
			coverIDs = append(coverIDs, *row.CoverStickerID)
		}
	}

	covers, err := s.stickers.ByIDs(coverIDs)
	if err != nil {
		return nil, errors.ErrInternal("failed to load covers")
	}
	tags, err := s.tags.ForPacks(packIDs)
	if err != nil {
		return nil, errors.ErrInternal("failed to load tags")
	}
	authors := s.users.Users(ctx, ownerIDs)

	for _, row := range rows {
		var cover *model.Sticker
		if row.CoverStickerID != nil {
			if found, ok := covers[*row.CoverStickerID]; ok {
				cover = &found
			}
		}
		author, ok := authors[row.OwnerUID]
		if !ok {
			author = userclient.Placeholder(row.OwnerUID)
		}
		out = append(out, s.packDTO(row, cover, tags[row.ID], author))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, v Viewer) (*dto.PackDetail, *errors.AppError) {
	pack, appErr := s.visiblePack(id, v)
	if appErr != nil {
		return nil, appErr
	}

	rows, err := s.stickers.ListByPack(pack.ID)
	if err != nil {
		return nil, errors.ErrInternal("failed to load stickers")
	}
	packs, appErr := s.hydrate(ctx, []model.Pack{*pack})
	if appErr != nil {
		return nil, appErr
	}

	stickers := make([]dto.Sticker, 0, len(rows))
	for _, row := range rows {
		stickers = append(stickers, s.stickerDTO(row))
	}

	if pack.Status == model.PackPublished {
		_ = s.packs.BumpView(pack.ID)
	}
	works, characters := distinctCatalog(stickers)
	s.fillWorkCovers(ctx, works)
	return &dto.PackDetail{
		Pack:       packs[0],
		Stickers:   stickers,
		Works:      works,
		Characters: characters,
	}, nil
}

// fillWorkCovers fetches the cover art for the games a pack's stickers point
// at. Only the pack's own declared game has a cover cached -- a sticker stores
// its game's id, name and rating but not a second copy of the art -- so
// without this a mixed pack lists its games as bare text. One batch call for
// the page, and a failure costs the covers, not the list.
func (s *Service) fillWorkCovers(ctx context.Context, works []dto.CatalogWork) {
	missing := make([]string, 0, len(works))
	for _, work := range works {
		if work.CoverURL == "" {
			missing = append(missing, strconv.FormatInt(work.ID, 10))
		}
	}
	if len(missing) == 0 || !s.catalog.Configured() {
		return
	}
	covers, err := s.catalog.WorksByIDs(ctx, missing)
	if err != nil {
		return
	}
	for i := range works {
		if works[i].CoverURL != "" {
			continue
		}
		if full, ok := covers[strconv.FormatInt(works[i].ID, 10)]; ok {
			works[i].CoverURL = imageURL(full.Cover)
		}
	}
}

// distinctCatalog collects the games and characters a pack's stickers point
// at, first-appearance order. The seven official packs each span dozens of
// games, so the detail page shows what is actually inside rather than the one
// game the pack may have declared.
func distinctCatalog(stickers []dto.Sticker) ([]dto.CatalogWork, []dto.CatalogCharacter) {
	works := make([]dto.CatalogWork, 0, 4)
	characters := make([]dto.CatalogCharacter, 0, 8)
	seenWork := map[int64]bool{}
	seenCharacter := map[int64]bool{}
	for _, sticker := range stickers {
		if w := sticker.CatalogWork; w != nil && !seenWork[w.ID] {
			seenWork[w.ID] = true
			works = append(works, *w)
		}
		if ch := sticker.CatalogCharacter; ch != nil && !seenCharacter[ch.ID] {
			seenCharacter[ch.ID] = true
			characters = append(characters, *ch)
		}
	}
	return works, characters
}

func (s *Service) visiblePack(id uuid.UUID, v Viewer) (*model.Pack, *errors.AppError) {
	pack, err := s.packs.Get(id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrPackNotFound()
		}
		return nil, errors.ErrInternal("failed to load pack")
	}
	if pack.Status == model.PackPublished {
		return pack, nil
	}
	if pack.Status == model.PackRemoved || !v.canSeeUnpublished(pack) {
		// 404 rather than 403: a stranger should not learn that a draft exists.
		return nil, errors.ErrPackNotFound()
	}
	return pack, nil
}

// An official sticker's hash is embedded in post content on other sites -- the forum
// alone had 1620 such embeds. A hard delete drops the hash out of this site's daily
// reference ping, and the image service then collects the bytes on its own TTL, so
// every one of those posts breaks retroactively. Unpublishing (status -> PackDraft)
// takes a pack out of the picker while keeping its rows, and therefore its ping.
func refuseOfficialDelete(pack *model.Pack) *errors.AppError {
	if !pack.IsOfficial {
		return nil
	}
	return errors.ErrForbidden(
		"official stickers are retired by unpublishing the pack, never deleted: " +
			"other sites' posts reference these images by hash")
}

func (s *Service) editablePack(id uuid.UUID, v Viewer) (*model.Pack, *errors.AppError) {
	pack, err := s.packs.Get(id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrPackNotFound()
		}
		return nil, errors.ErrInternal("failed to load pack")
	}
	if !v.canEdit(pack) {
		if v.canSeeUnpublished(pack) || pack.Status == model.PackPublished {
			return nil, errors.ErrNotOwner()
		}
		return nil, errors.ErrPackNotFound()
	}
	return pack, nil
}

func (s *Service) CreatePack(ctx context.Context, v Viewer, req dto.CreatePackRequest) (*dto.Pack, *errors.AppError) {
	count, err := s.packs.CountByOwner(v.UID)
	if err != nil {
		return nil, errors.ErrInternal("failed to count packs")
	}
	if count >= MaxPacksPerUser {
		return nil, errors.ErrPackLimit(MaxPacksPerUser)
	}

	title := sanitizeML(req.Title)
	if !hasText(title) {
		return nil, errors.ErrInvalidParams("a title in at least one language is required")
	}
	description := sanitizeML(req.Description)

	work, appErr := s.resolveWork(ctx, req.CatalogWorkID)
	if appErr != nil {
		return nil, appErr
	}

	pack := &model.Pack{
		OwnerUID:          v.UID,
		Status:            model.PackDraft,
		ContentRating:     rating(req.ContentRating),
		Title:             title,
		Description:       description,
		CatalogWorkID:     work.ID,
		CatalogWorkName:   work.Name,
		CatalogWorkCover:  work.Cover,
		CatalogWorkRating: work.Rating,
		SearchText:        searchText(title, description, work.Name),
	}
	if err := s.packs.Create(pack); err != nil {
		return nil, errors.ErrInternal("failed to create pack")
	}
	if appErr := s.applyTags(pack.ID, req.Tags); appErr != nil {
		return nil, appErr
	}
	return s.single(ctx, pack)
}

func (s *Service) PatchPack(ctx context.Context, id uuid.UUID, v Viewer, req dto.PatchPackRequest) (*dto.Pack, *errors.AppError) {
	pack, appErr := s.editablePack(id, v)
	if appErr != nil {
		return nil, appErr
	}

	if req.Title != nil {
		title := sanitizeML(*req.Title)
		if !hasText(title) {
			return nil, errors.ErrInvalidParams("a title in at least one language is required")
		}
		pack.Title = title
	}
	if req.Description != nil {
		pack.Description = sanitizeML(*req.Description)
	}
	if req.ContentRating != nil {
		pack.ContentRating = rating(req.ContentRating)
	}
	if req.CoverStickerID != nil {
		coverID, ok := parseUUID(*req.CoverStickerID)
		if !ok {
			return nil, errors.ErrInvalidParams("cover_sticker_id is not a uuid")
		}
		cover, err := s.stickers.Get(coverID)
		if err != nil || cover.PackID != pack.ID {
			return nil, errors.ErrInvalidParams("cover sticker does not belong to this pack")
		}
		pack.CoverStickerID = &coverID
	}
	if req.CatalogWorkID != nil {
		work, appErr := s.resolveWork(ctx, req.CatalogWorkID)
		if appErr != nil {
			return nil, appErr
		}
		pack.CatalogWorkID = work.ID
		pack.CatalogWorkName = work.Name
		pack.CatalogWorkCover = work.Cover
		pack.CatalogWorkRating = work.Rating
	}
	pack.SearchText = s.searchTextFor(pack)

	if err := s.packs.Save(pack); err != nil {
		return nil, errors.ErrInternal("failed to save pack")
	}
	if req.Tags != nil {
		if appErr := s.applyTags(pack.ID, *req.Tags); appErr != nil {
			return nil, appErr
		}
	}
	return s.single(ctx, pack)
}

func (s *Service) SetPublished(ctx context.Context, id uuid.UUID, v Viewer, publish bool) (*dto.Pack, *errors.AppError) {
	pack, appErr := s.editablePack(id, v)
	if appErr != nil {
		return nil, appErr
	}

	if publish {
		if pack.StickerCount < 1 {
			return nil, errors.ErrPackNeedsSticker()
		}
		pack.Status = model.PackPublished
		if pack.PublishedAt == nil {
			now := time.Now()
			pack.PublishedAt = &now
		}
	} else {
		pack.Status = model.PackHidden
	}

	if err := s.packs.Save(pack); err != nil {
		return nil, errors.ErrInternal("failed to save pack")
	}
	// pack_count only counts published packs, so the tag cloud moves with this.
	_ = s.tags.RecountForPack(pack.ID)
	return s.single(ctx, pack)
}

func (s *Service) DeletePack(id uuid.UUID, v Viewer) *errors.AppError {
	pack, err := s.packs.Get(id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.ErrPackNotFound()
		}
		return errors.ErrInternal("failed to load pack")
	}
	if !(pack.OwnerUID == v.UID || perm.Can(v.Roles, perm.PackDeleteAny)) {
		return errors.ErrNotOwner()
	}
	if appErr := refuseOfficialDelete(pack); appErr != nil {
		return appErr
	}

	var tagIDs []uuid.UUID
	if err := s.packs.DB().Table("pack_tag").Where("pack_id = ?", id).
		Pluck("tag_id", &tagIDs).Error; err != nil {
		return errors.ErrInternal("failed to load tags")
	}
	// Stickers and pack_tag rows go with the pack (ON DELETE CASCADE). The
	// image bytes are deliberately left alone: the image service deduplicates
	// across every site in the ecosystem, so a hash this pack used may still
	// back someone else's image. Dropping it from the daily reference ping is
	// what releases it, on the image service's own TTL.
	if err := s.packs.Delete(id); err != nil {
		return errors.ErrInternal("failed to delete pack")
	}
	_ = s.tags.Recount(tagIDs)
	return nil
}

func (s *Service) PopularTags(limit int) ([]dto.Tag, *errors.AppError) {
	rows, err := s.tags.Popular(0, limit)
	if err != nil {
		return nil, errors.ErrInternal("failed to load tags")
	}
	return tagDTOs(rows), nil
}

func (s *Service) applyTags(packID uuid.UUID, labels []string) *errors.AppError {
	if labels == nil {
		return nil
	}
	if len(labels) > MaxTagsPerPack {
		return errors.ErrTagLimit(MaxTagsPerPack)
	}
	if err := s.tags.SetPackTags(packID, labels); err != nil {
		return errors.ErrInternal("failed to save tags")
	}
	return nil
}

func (s *Service) single(ctx context.Context, pack *model.Pack) (*dto.Pack, *errors.AppError) {
	fresh, err := s.packs.Get(pack.ID)
	if err != nil {
		return nil, errors.ErrInternal("failed to reload pack")
	}
	packs, appErr := s.hydrate(ctx, []model.Pack{*fresh})
	if appErr != nil {
		return nil, appErr
	}
	return &packs[0], nil
}

// searchTextFor rebuilds a pack's search column from the pack itself plus the
// games its stickers point at. A mixed pack is findable by every game inside
// it, not only by the one it declared.
func (s *Service) searchTextFor(pack *model.Pack) string {
	fields := []datatypes.JSON{pack.Title, pack.Description, pack.CatalogWorkName}
	if names, err := s.stickers.DistinctWorkNames(pack.ID); err == nil {
		fields = append(fields, names...)
	}
	return searchText(fields...)
}

// SyncPackSearchText is called after a sticker's game link changes: the pack's
// search column includes its stickers' games, so it goes stale otherwise.
func (s *Service) syncPackSearchText(packID uuid.UUID) {
	pack, err := s.packs.Get(packID)
	if err != nil {
		return
	}
	pack.SearchText = s.searchTextFor(pack)
	_ = s.packs.Save(pack)
}

func rating(in *int16) int16 {
	if in != nil && *in == model.RatingNSFW {
		return model.RatingNSFW
	}
	return model.RatingSFW
}
