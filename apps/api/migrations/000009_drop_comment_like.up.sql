-- Retire the local mirror of comment likes.
--
-- 000007 created it because community's post projection carried no reaction
-- fields. It now does: every read face returns reaction_count, fills
-- viewer_reacted for the viewer_id the caller names, and the toggle answers
-- with the count it took in the same transaction. A mirror beside that is a
-- second writer on one fact, and it drifted the first time the two writes
-- disagreed -- with nothing to reconcile it.
--
-- Nothing is carried over, and nothing needs to be: every row here was a copy
-- of a reaction community still holds. Production had no rows to copy anyway
-- -- no comment had been posted yet when the mirror was retired.

DROP TABLE IF EXISTS comment_like;
