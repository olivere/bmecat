package bmecat12

import "context"

// StreamArticles adapts a pull-style producer to the (<-chan *Article,
// <-chan error) pair a CatalogWriter's Articles method must return. It confines
// the channel bookkeeping the interface otherwise demands — a buffered error
// channel, sending at most one error before stopping, and selecting on
// ctx.Done so a canceled context unblocks the producer — to one place, so an
// Articles implementation cannot get the contract subtly wrong.
//
// produce is called once and streams articles by calling yield once per
// article. yield forwards the article to Writer and returns a non-nil error
// when the context is canceled (Writer cancels it, if it was made cancelable,
// once Do returns), letting the producer stop early:
//
//	func (c myCatalog) Articles(ctx context.Context) (<-chan *bmecat12.Article, <-chan error) {
//		return bmecat12.StreamArticles(ctx, func(yield func(*bmecat12.Article) error) error {
//			for rows.Next() {
//				if err := yield(buildArticle(rows)); err != nil {
//					return err // downstream stopped; stop producing
//				}
//			}
//			return rows.Err()
//		})
//	}
//
// Returning a non-nil error from produce — from yield or the producer itself,
// such as rows.Err — stops the write and is reported by Writer.Do. A nil
// article is skipped.
func StreamArticles(ctx context.Context, produce func(yield func(*Article) error) error) (<-chan *Article, <-chan error) {
	out := make(chan *Article)
	errc := make(chan error, 1)
	go func() {
		yield := func(a *Article) error {
			if a == nil {
				return nil
			}
			select {
			case out <- a:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		// On error, errc receives it and out is left open: Writer's article loop
		// returns via the error channel, and leaving out open keeps a clean EOF
		// from racing the error in that select. out is closed only on clean
		// completion, when no error is sent.
		if err := produce(yield); err != nil {
			errc <- err
			return
		}
		close(out)
	}()
	return out, errc
}
