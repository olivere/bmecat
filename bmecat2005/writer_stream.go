package bmecat2005

import "context"

// StreamProducts adapts a pull-style producer to the (<-chan *Product,
// <-chan error) pair a CatalogWriter's Products method must return. It confines
// the channel bookkeeping the interface otherwise demands — a buffered error
// channel, sending at most one error before stopping, and selecting on
// ctx.Done so a canceled context unblocks the producer — to one place, so a
// Products implementation cannot get the contract subtly wrong.
//
// produce is called once and streams products by calling yield once per
// product. yield forwards the product to Writer and returns a non-nil error
// when ctx is canceled, letting the producer stop early:
//
//	func (c myCatalog) Products(ctx context.Context) (<-chan *bmecat2005.Product, <-chan error) {
//		return bmecat2005.StreamProducts(ctx, func(yield func(*bmecat2005.Product) error) error {
//			for rows.Next() {
//				if err := yield(buildProduct(rows)); err != nil {
//					return err // ctx canceled; stop producing
//				}
//			}
//			return rows.Err()
//		})
//	}
//
// Returning a non-nil error from produce — from yield or the producer itself,
// such as rows.Err — stops the write and is reported by Writer.Do. A nil
// product is skipped.
//
// Writer.Do does not cancel ctx itself, so if Do can return before the producer
// is drained (an encoding error mid-stream), pass a cancelable context and
// cancel it once Do returns; the pending yield then unblocks instead of leaking
// the producer goroutine.
func StreamProducts(ctx context.Context, produce func(yield func(*Product) error) error) (<-chan *Product, <-chan error) {
	out := make(chan *Product)
	errc := make(chan error, 1)
	go func() {
		yield := func(p *Product) error {
			if p == nil {
				return nil
			}
			select {
			case out <- p:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		// On error, errc receives it and out is left open: Writer's product loop
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
